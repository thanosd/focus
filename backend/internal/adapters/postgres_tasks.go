package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/ports"
)

// TaskRepo is the Postgres TaskRepository.
type TaskRepo struct{ db *sql.DB }

// NewTaskRepository constructs a TaskRepo.
func NewTaskRepository(db *sql.DB) ports.TaskRepository { return &TaskRepo{db: db} }

// taskBase selects every task column plus the derived project name and
// availability. $1 = user_id, $2 = "now". Availability is computed in
// SQL so every list view and the project counts agree on the rule:
// active, not deferred past now, inbox or active project, and — for
// sequential projects — first in sort order among the project's active
// tasks.
const taskBase = `
SELECT t.id, t.user_id, t.project_id, COALESCE(p.name, '') AS project_name, t.title, t.note, t.status, t.flagged,
       t.defer_until, t.due_at, t.repeat_rule, t.sort_order, t.completed_at, t.dropped_at, t.created_at, t.updated_at,
       (t.status = 'active'
         AND (t.defer_until IS NULL OR t.defer_until <= $2)
         AND (t.project_id IS NULL OR (p.status = 'active' AND (NOT p.sequential OR t.id = (
              SELECT t2.id FROM tasks t2 WHERE t2.project_id = t.project_id AND t2.status = 'active'
              ORDER BY t2.sort_order, t2.created_at LIMIT 1))))) AS is_available
FROM tasks t LEFT JOIN projects p ON p.id = t.project_id
WHERE t.user_id = $1`

func scanTask(row interface{ Scan(...any) error }) (*domain.Task, error) {
	var t domain.Task
	var rule []byte
	err := row.Scan(&t.ID, &t.UserID, &t.ProjectID, &t.ProjectName, &t.Title, &t.Note, &t.Status, &t.Flagged,
		&t.DeferUntil, &t.DueAt, &rule, &t.SortOrder, &t.CompletedAt, &t.DroppedAt, &t.CreatedAt, &t.UpdatedAt, &t.IsAvailable)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(rule) > 0 {
		var r domain.RepeatRule
		if err := json.Unmarshal(rule, &r); err == nil {
			t.RepeatRule = &r
		}
	}
	t.Tags = []domain.Tag{}
	return &t, nil
}

// Create inserts a task at the end of its container's sort order.
func (r *TaskRepo) Create(ctx context.Context, t *domain.Task) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.Status == "" {
		t.Status = domain.TaskActive
	}
	rule, err := marshalRule(t.RepeatRule)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO tasks (id, user_id, project_id, title, note, status, flagged, defer_until, due_at, repeat_rule, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			COALESCE((SELECT MAX(sort_order) + 1 FROM tasks WHERE user_id = $2 AND project_id IS NOT DISTINCT FROM $3), 0))`,
		t.ID, t.UserID, t.ProjectID, t.Title, t.Note, t.Status, t.Flagged, t.DeferUntil, t.DueAt, rule)
	return err
}

func marshalRule(rule *domain.RepeatRule) (any, error) {
	if rule == nil {
		return nil, nil
	}
	b, err := json.Marshal(rule)
	if err != nil {
		return nil, fmt.Errorf("marshal repeat rule: %w", err)
	}
	return string(b), nil
}

// GetByID loads one task (with tags) scoped to the user.
func (r *TaskRepo) GetByID(ctx context.Context, userID, id string) (*domain.Task, error) {
	t, err := scanTask(r.db.QueryRowContext(ctx, taskBase+` AND t.id = $3`, userID, time.Now(), id))
	if err != nil || t == nil {
		return t, err
	}
	if err := r.attachTags(ctx, []*domain.Task{t}); err != nil {
		return nil, err
	}
	return t, nil
}

// List applies a TaskFilter on top of taskBase.
func (r *TaskRepo) List(ctx context.Context, userID string, f domain.TaskFilter) ([]domain.Task, error) {
	args := []any{userID, time.Now()}
	conds := []string{}
	order := "x.sort_order, x.created_at"
	limit := ""

	next := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	switch f.View {
	case domain.ViewInbox:
		conds = append(conds, "x.project_id IS NULL", "x.status = 'active'")
	case domain.ViewAvailable, "":
		conds = append(conds, "x.is_available")
	case domain.ViewFlagged:
		conds = append(conds, "x.status = 'active'", "x.flagged")
		order = "x.due_at ASC NULLS LAST, x.sort_order, x.created_at"
	case domain.ViewDue:
		conds = append(conds, "x.status = 'active'", "x.due_at IS NOT NULL")
		order = "x.due_at ASC, x.sort_order"
	case domain.ViewCompleted:
		conds = append(conds, "x.status IN ('completed', 'dropped')")
		order = "COALESCE(x.completed_at, x.dropped_at) DESC"
		limit = " LIMIT 200"
	case domain.ViewAll:
		conds = append(conds, "x.status = 'active'")
	default:
		return nil, fmt.Errorf("%w: unknown view %q", domain.ErrValidation, f.View)
	}
	if f.ProjectID != nil {
		conds = append(conds, "x.project_id = "+next(*f.ProjectID))
	}
	if f.TagID != nil {
		conds = append(conds, "EXISTS (SELECT 1 FROM task_tags tt WHERE tt.task_id = x.id AND tt.tag_id = "+next(*f.TagID)+")")
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		p := next("%" + q + "%")
		conds = append(conds, "(x.title ILIKE "+p+" OR x.note ILIKE "+p+")")
	}

	query := "SELECT * FROM (" + taskBase + ") x WHERE " + strings.Join(conds, " AND ") + " ORDER BY " + order + limit
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := []domain.Task{}
	ptrs := []*domain.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		ptrs = append(ptrs, &out[i])
	}
	if err := r.attachTags(ctx, ptrs); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *TaskRepo) attachTags(ctx context.Context, tasks []*domain.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	ids := make([]string, 0, len(tasks))
	byID := make(map[string]*domain.Task, len(tasks))
	for _, t := range tasks {
		ids = append(ids, t.ID)
		byID[t.ID] = t
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT tt.task_id, g.id, g.user_id, g.name, g.color, g.sort_order, g.created_at, g.updated_at
		FROM task_tags tt JOIN tags g ON g.id = tt.tag_id
		WHERE tt.task_id = ANY($1) ORDER BY g.sort_order, LOWER(g.name)`, pq.Array(ids))
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var taskID string
		var g domain.Tag
		if err := rows.Scan(&taskID, &g.ID, &g.UserID, &g.Name, &g.Color, &g.SortOrder, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return err
		}
		if t, ok := byID[taskID]; ok {
			t.Tags = append(t.Tags, g)
		}
	}
	return rows.Err()
}

// Update writes every mutable column.
func (r *TaskRepo) Update(ctx context.Context, t *domain.Task) error {
	rule, err := marshalRule(t.RepeatRule)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE tasks SET project_id = $3, title = $4, note = $5, status = $6, flagged = $7, defer_until = $8, due_at = $9,
			repeat_rule = $10, sort_order = $11, completed_at = $12, dropped_at = $13
		WHERE id = $1 AND user_id = $2`,
		t.ID, t.UserID, t.ProjectID, t.Title, t.Note, t.Status, t.Flagged, t.DeferUntil, t.DueAt,
		rule, t.SortOrder, t.CompletedAt, t.DroppedAt)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// SetTags replaces the task's tag set. Tags not owned by the user are ignored.
func (r *TaskRepo) SetTags(ctx context.Context, userID, taskID string, tagIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_tags WHERE task_id = $1`, taskID); err != nil {
		return err
	}
	if len(tagIDs) > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO task_tags (task_id, tag_id)
			SELECT $1, g.id FROM tags g WHERE g.user_id = $2 AND g.id = ANY($3)
			ON CONFLICT DO NOTHING`, taskID, userID, pq.Array(tagIDs)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Reorder writes sort_order = index for the listed tasks in one statement.
func (r *TaskRepo) Reorder(ctx context.Context, userID string, taskIDs []string) error {
	if len(taskIDs) == 0 {
		return nil
	}
	positions := make([]int, len(taskIDs))
	for i := range taskIDs {
		positions[i] = i
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE tasks t SET sort_order = o.pos
		FROM UNNEST($2::uuid[], $3::int[]) AS o(id, pos)
		WHERE t.id = o.id AND t.user_id = $1`, userID, pq.Array(taskIDs), pq.Array(positions))
	return err
}

// Delete removes a task.
func (r *TaskRepo) Delete(ctx context.Context, userID, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Counts computes the sidebar badges in one round trip.
func (r *TaskRepo) Counts(ctx context.Context, userID string, now time.Time) (*domain.Counts, error) {
	var c domain.Counts
	err := r.db.QueryRowContext(ctx, `
		SELECT
		  (SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND status = 'active' AND project_id IS NULL),
		  (SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND status = 'active' AND flagged),
		  (SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND status = 'active' AND due_at IS NOT NULL AND due_at > $2 AND due_at <= $2 + INTERVAL '24 hours'),
		  (SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND status = 'active' AND due_at IS NOT NULL AND due_at <= $2),
		  (SELECT COUNT(*) FROM projects p WHERE p.user_id = $1 AND p.status IN ('active', 'on_hold') AND p.next_review_at IS NOT NULL AND p.next_review_at <= $2
		     AND `+reviewableCondition+`)`,
		userID, now).Scan(&c.Inbox, &c.Flagged, &c.DueSoon, &c.Overdue, &c.ReviewDue)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
