// Package ports declares the inward-facing repository interfaces that
// services depend on. Postgres implementations live in adapters/.
package ports

import (
	"context"
	"time"

	"github.com/thanosd/focus/backend/internal/domain"
)

// UserRepository persists users.
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	// UpsertFromGoogle creates or refreshes the user row for a Google
	// profile, keyed by email.
	UpsertFromGoogle(ctx context.Context, email, sub, name, picture string) (*domain.User, error)
	UpdateTimezone(ctx context.Context, id, timezone string) error
	UpdateLastLoginAt(ctx context.Context, id string, at time.Time) error
}

// SessionRepository persists browser sessions.
type SessionRepository interface {
	Create(ctx context.Context, s *domain.Session) error
	// GetValid returns nil, nil when the session is missing or expired.
	GetValid(ctx context.Context, id string) (*domain.Session, error)
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// OAuthStateRepository persists sign-in nonces.
type OAuthStateRepository interface {
	Create(ctx context.Context, s *domain.OAuthState) error
	// Consume returns and deletes the state; nil, nil when missing/expired.
	Consume(ctx context.Context, state string) (*domain.OAuthState, error)
	DeleteExpired(ctx context.Context) (int64, error)
}

// APITokenRepository persists MCP API tokens.
type APITokenRepository interface {
	Create(ctx context.Context, t *domain.APIToken) error
	ListByUser(ctx context.Context, userID string) ([]domain.APIToken, error)
	// GetActiveByHash returns nil, nil when no live token matches.
	GetActiveByHash(ctx context.Context, hash string) (*domain.APIToken, error)
	TouchLastUsed(ctx context.Context, id string, at time.Time) error
	Revoke(ctx context.Context, userID, id string) error
}

// ProjectRepository persists projects.
type ProjectRepository interface {
	Create(ctx context.Context, p *domain.Project) error
	GetByID(ctx context.Context, userID, id string) (*domain.Project, error)
	List(ctx context.Context, userID string, statuses []domain.ProjectStatus) ([]domain.Project, error)
	ListChildren(ctx context.Context, userID, parentID string) ([]domain.Project, error)
	Update(ctx context.Context, p *domain.Project) error
	Delete(ctx context.Context, userID, id string) error
	// ListReviewDue returns active/on-hold projects whose next review is
	// at or before now, oldest first.
	ListReviewDue(ctx context.Context, userID string, now time.Time) ([]domain.Project, error)
	// ListReviewUpcoming returns active/on-hold projects not yet due,
	// soonest first.
	ListReviewUpcoming(ctx context.Context, userID string, now time.Time) ([]domain.Project, error)
	CountReviewDue(ctx context.Context, userID string, now time.Time) (int, error)
}

// TaskRepository persists tasks and their tag links.
type TaskRepository interface {
	Create(ctx context.Context, t *domain.Task) error
	GetByID(ctx context.Context, userID, id string) (*domain.Task, error)
	List(ctx context.Context, userID string, f domain.TaskFilter) ([]domain.Task, error)
	Update(ctx context.Context, t *domain.Task) error
	SetTags(ctx context.Context, userID, taskID string, tagIDs []string) error
	Delete(ctx context.Context, userID, id string) error
	Counts(ctx context.Context, userID string, now time.Time) (*domain.Counts, error)
}

// TagRepository persists tags.
type TagRepository interface {
	Create(ctx context.Context, t *domain.Tag) error
	GetByID(ctx context.Context, userID, id string) (*domain.Tag, error)
	GetByName(ctx context.Context, userID, name string) (*domain.Tag, error)
	List(ctx context.Context, userID string) ([]domain.Tag, error)
	Update(ctx context.Context, t *domain.Tag) error
	Delete(ctx context.Context, userID, id string) error
}
