package adapters

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/ports"
)

// TagRepo is the Postgres TagRepository.
type TagRepo struct{ db *sql.DB }

// NewTagRepository constructs a TagRepo.
func NewTagRepository(db *sql.DB) ports.TagRepository { return &TagRepo{db: db} }

const tagSelect = `
SELECT g.id, g.user_id, g.name, g.color, g.sort_order, g.created_at, g.updated_at,
       (SELECT COUNT(*) FROM task_tags tt JOIN tasks t ON t.id = tt.task_id WHERE tt.tag_id = g.id AND t.status = 'active') AS active_task_count
FROM tags g`

func scanTag(row interface{ Scan(...any) error }) (*domain.Tag, error) {
	var g domain.Tag
	err := row.Scan(&g.ID, &g.UserID, &g.Name, &g.Color, &g.SortOrder, &g.CreatedAt, &g.UpdatedAt, &g.ActiveTaskCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// Create inserts a tag.
func (r *TagRepo) Create(ctx context.Context, t *domain.Tag) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.Color == "" {
		t.Color = "#3b82f6"
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tags (id, user_id, name, color, sort_order)
		VALUES ($1, $2, $3, $4, COALESCE((SELECT MAX(sort_order) + 1 FROM tags WHERE user_id = $2), 0))`,
		t.ID, t.UserID, t.Name, t.Color)
	return err
}

// GetByID loads one tag scoped to the user.
func (r *TagRepo) GetByID(ctx context.Context, userID, id string) (*domain.Tag, error) {
	return scanTag(r.db.QueryRowContext(ctx, tagSelect+` WHERE g.user_id = $1 AND g.id = $2`, userID, id))
}

// GetByName loads a tag by case-insensitive name.
func (r *TagRepo) GetByName(ctx context.Context, userID, name string) (*domain.Tag, error) {
	return scanTag(r.db.QueryRowContext(ctx, tagSelect+` WHERE g.user_id = $1 AND LOWER(g.name) = LOWER($2)`, userID, name))
}

// List returns all of the user's tags.
func (r *TagRepo) List(ctx context.Context, userID string) ([]domain.Tag, error) {
	rows, err := r.db.QueryContext(ctx, tagSelect+` WHERE g.user_id = $1 ORDER BY g.sort_order, LOWER(g.name)`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []domain.Tag{}
	for rows.Next() {
		g, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *g)
	}
	return out, rows.Err()
}

// Update renames/recolors a tag.
func (r *TagRepo) Update(ctx context.Context, t *domain.Tag) error {
	res, err := r.db.ExecContext(ctx, `UPDATE tags SET name = $3, color = $4, sort_order = $5 WHERE id = $1 AND user_id = $2`,
		t.ID, t.UserID, t.Name, t.Color, t.SortOrder)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete removes a tag; task links cascade.
func (r *TagRepo) Delete(ctx context.Context, userID, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tags WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
