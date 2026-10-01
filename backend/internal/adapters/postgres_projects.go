package adapters

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/ports"
)

// ProjectRepo is the Postgres ProjectRepository.
type ProjectRepo struct{ db *sql.DB }

// NewProjectRepository constructs a ProjectRepo.
func NewProjectRepository(db *sql.DB) ports.ProjectRepository { return &ProjectRepo{db: db} }

// projectSelect returns every project column plus the derived depth and
// task counts. available_task_count applies the same availability rule
// as the task repository (active, not deferred, project active, and
// first-in-order for sequential projects).
const projectSelect = `
SELECT p.id, p.user_id, p.parent_id, p.name, p.note, p.status, p.sequential,
       p.review_interval_days, p.last_reviewed_at, p.next_review_at,
       CASE WHEN p.parent_id IS NULL THEN 0 ELSE 1 END AS depth,
       p.sort_order, p.completed_at, p.created_at, p.updated_at,
       (SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id AND t.status = 'active') AS remaining_task_count,
       (SELECT COUNT(*) FROM tasks t
          WHERE t.project_id = p.id AND t.status = 'active'
            AND (t.defer_until IS NULL OR t.defer_until <= NOW())
            AND p.status = 'active'
            AND (NOT p.sequential OR t.id = (
                SELECT t2.id FROM tasks t2 WHERE t2.project_id = p.id AND t2.status = 'active'
                ORDER BY t2.sort_order, t2.created_at LIMIT 1))) AS available_task_count
FROM projects p`

func scanProject(row interface{ Scan(...any) error }) (*domain.Project, error) {
	var p domain.Project
	err := row.Scan(&p.ID, &p.UserID, &p.ParentID, &p.Name, &p.Note, &p.Status, &p.Sequential,
		&p.ReviewIntervalDays, &p.LastReviewedAt, &p.NextReviewAt, &p.Depth,
		&p.SortOrder, &p.CompletedAt, &p.CreatedAt, &p.UpdatedAt,
		&p.RemainingTaskCount, &p.AvailableTaskCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanProjects(rows *sql.Rows) ([]domain.Project, error) {
	defer func() { _ = rows.Close() }()
	out := []domain.Project{}
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// Create inserts a project.
func (r *ProjectRepo) Create(ctx context.Context, p *domain.Project) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	if p.Status == "" {
		p.Status = domain.ProjectActive
	}
	if p.ReviewIntervalDays <= 0 {
		p.ReviewIntervalDays = 7
	}
	if p.NextReviewAt == nil {
		next := time.Now().AddDate(0, 0, p.ReviewIntervalDays)
		p.NextReviewAt = &next
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO projects (id, user_id, parent_id, name, note, status, sequential, review_interval_days, next_review_at, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
			COALESCE((SELECT MAX(sort_order) + 1 FROM projects WHERE user_id = $2 AND parent_id IS NOT DISTINCT FROM $3), 0))`,
		p.ID, p.UserID, p.ParentID, p.Name, p.Note, p.Status, p.Sequential, p.ReviewIntervalDays, p.NextReviewAt)
	return err
}

// GetByID loads one project scoped to the user.
func (r *ProjectRepo) GetByID(ctx context.Context, userID, id string) (*domain.Project, error) {
	return scanProject(r.db.QueryRowContext(ctx, projectSelect+` WHERE p.user_id = $1 AND p.id = $2`, userID, id))
}

// List returns projects in the given statuses ordered by depth then sort order.
func (r *ProjectRepo) List(ctx context.Context, userID string, statuses []domain.ProjectStatus) ([]domain.Project, error) {
	ss := make([]string, 0, len(statuses))
	for _, s := range statuses {
		ss = append(ss, string(s))
	}
	rows, err := r.db.QueryContext(ctx, projectSelect+` WHERE p.user_id = $1 AND p.status = ANY($2) ORDER BY depth, p.sort_order, p.created_at`, userID, pq.Array(ss))
	if err != nil {
		return nil, err
	}
	return scanProjects(rows)
}

// ListChildren returns the direct children of a project (any status).
func (r *ProjectRepo) ListChildren(ctx context.Context, userID, parentID string) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(ctx, projectSelect+` WHERE p.user_id = $1 AND p.parent_id = $2 ORDER BY p.sort_order, p.created_at`, userID, parentID)
	if err != nil {
		return nil, err
	}
	return scanProjects(rows)
}

// Update writes every mutable column.
func (r *ProjectRepo) Update(ctx context.Context, p *domain.Project) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE projects SET parent_id = $3, name = $4, note = $5, status = $6, sequential = $7,
			review_interval_days = $8, last_reviewed_at = $9, next_review_at = $10, sort_order = $11, completed_at = $12
		WHERE id = $1 AND user_id = $2`,
		p.ID, p.UserID, p.ParentID, p.Name, p.Note, p.Status, p.Sequential,
		p.ReviewIntervalDays, p.LastReviewedAt, p.NextReviewAt, p.SortOrder, p.CompletedAt)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete removes a project; children and tasks cascade.
func (r *ProjectRepo) Delete(ctx context.Context, userID, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListReviewDue returns projects whose review is overdue, oldest first.
func (r *ProjectRepo) ListReviewDue(ctx context.Context, userID string, now time.Time) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(ctx, projectSelect+`
		WHERE p.user_id = $1 AND p.status IN ('active', 'on_hold') AND p.next_review_at IS NOT NULL AND p.next_review_at <= $2
		ORDER BY p.next_review_at ASC, p.created_at`, userID, now)
	if err != nil {
		return nil, err
	}
	return scanProjects(rows)
}

// ListReviewUpcoming returns projects not yet due for review, soonest first.
func (r *ProjectRepo) ListReviewUpcoming(ctx context.Context, userID string, now time.Time) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(ctx, projectSelect+`
		WHERE p.user_id = $1 AND p.status IN ('active', 'on_hold') AND (p.next_review_at IS NULL OR p.next_review_at > $2)
		ORDER BY p.next_review_at ASC NULLS LAST, p.created_at`, userID, now)
	if err != nil {
		return nil, err
	}
	return scanProjects(rows)
}

// CountReviewDue counts overdue reviews.
func (r *ProjectRepo) CountReviewDue(ctx context.Context, userID string, now time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE user_id = $1 AND status IN ('active', 'on_hold') AND next_review_at IS NOT NULL AND next_review_at <= $2`, userID, now).Scan(&n)
	return n, err
}
