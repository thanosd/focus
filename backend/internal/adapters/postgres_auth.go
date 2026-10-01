package adapters

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/ports"
)

// ── Users ────────────────────────────────────────────────────────────

// UserRepo is the Postgres UserRepository.
type UserRepo struct{ db *sql.DB }

// NewUserRepository constructs a UserRepo.
func NewUserRepository(db *sql.DB) ports.UserRepository { return &UserRepo{db: db} }

const userColumns = `id, email, COALESCE(google_sub, ''), name, picture_url, timezone, last_login_at, created_at, updated_at`

func scanUser(row interface{ Scan(...any) error }) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.GoogleSub, &u.Name, &u.PictureURL, &u.Timezone, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByID loads a user by primary key.
func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

// GetByEmail loads a user by (case-insensitive) email.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE LOWER(email) = LOWER($1)`, email))
}

// UpsertFromGoogle creates the user on first sign-in and refreshes the
// profile fields on every subsequent one.
func (r *UserRepo) UpsertFromGoogle(ctx context.Context, email, sub, name, picture string) (*domain.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, `
		INSERT INTO users (id, email, google_sub, name, picture_url)
		VALUES ($1, LOWER($2), $3, $4, $5)
		ON CONFLICT (email) DO UPDATE SET
			google_sub = EXCLUDED.google_sub,
			name = CASE WHEN EXCLUDED.name <> '' THEN EXCLUDED.name ELSE users.name END,
			picture_url = CASE WHEN EXCLUDED.picture_url <> '' THEN EXCLUDED.picture_url ELSE users.picture_url END
		RETURNING `+userColumns, uuid.New().String(), email, nullIfEmpty(sub), name, picture))
}

// UpdateTimezone stores the user's IANA timezone.
func (r *UserRepo) UpdateTimezone(ctx context.Context, id, timezone string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET timezone = $2 WHERE id = $1`, id, timezone)
	return err
}

// UpdateLastLoginAt stamps the last successful sign-in.
func (r *UserRepo) UpdateLastLoginAt(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET last_login_at = $2 WHERE id = $1`, id, at)
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ── Sessions ─────────────────────────────────────────────────────────

// SessionRepo is the Postgres SessionRepository.
type SessionRepo struct{ db *sql.DB }

// NewSessionRepository constructs a SessionRepo.
func NewSessionRepository(db *sql.DB) ports.SessionRepository { return &SessionRepo{db: db} }

// Create inserts a session, generating the ID when empty.
func (r *SessionRepo) Create(ctx context.Context, s *domain.Session) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	s.CreatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `INSERT INTO sessions (id, user_id, expires_at, created_at) VALUES ($1, $2, $3, $4)`,
		s.ID, s.UserID, s.ExpiresAt, s.CreatedAt)
	return err
}

// GetValid returns the session only when it has not expired.
func (r *SessionRepo) GetValid(ctx context.Context, id string) (*domain.Session, error) {
	var s domain.Session
	err := r.db.QueryRowContext(ctx, `SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = $1 AND expires_at > NOW()`, id).
		Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Delete removes a session.
func (r *SessionRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

// DeleteExpired garbage-collects expired sessions.
func (r *SessionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= NOW()`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ── OAuth states ─────────────────────────────────────────────────────

// OAuthStateRepo is the Postgres OAuthStateRepository.
type OAuthStateRepo struct{ db *sql.DB }

// NewOAuthStateRepository constructs an OAuthStateRepo.
func NewOAuthStateRepository(db *sql.DB) ports.OAuthStateRepository { return &OAuthStateRepo{db: db} }

// Create stores a nonce.
func (r *OAuthStateRepo) Create(ctx context.Context, s *domain.OAuthState) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO oauth_states (state, redirect_to, expires_at) VALUES ($1, $2, $3)`,
		s.State, s.RedirectTo, s.ExpiresAt)
	return err
}

// Consume atomically deletes and returns a live nonce.
func (r *OAuthStateRepo) Consume(ctx context.Context, state string) (*domain.OAuthState, error) {
	var s domain.OAuthState
	err := r.db.QueryRowContext(ctx, `DELETE FROM oauth_states WHERE state = $1 AND expires_at > NOW() RETURNING state, redirect_to, expires_at`, state).
		Scan(&s.State, &s.RedirectTo, &s.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteExpired garbage-collects expired nonces.
func (r *OAuthStateRepo) DeleteExpired(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM oauth_states WHERE expires_at <= NOW()`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ── API tokens ───────────────────────────────────────────────────────

// APITokenRepo is the Postgres APITokenRepository.
type APITokenRepo struct{ db *sql.DB }

// NewAPITokenRepository constructs an APITokenRepo.
func NewAPITokenRepository(db *sql.DB) ports.APITokenRepository { return &APITokenRepo{db: db} }

const apiTokenColumns = `id, user_id, name, token_hash, token_prefix, last_used_at, revoked_at, created_at`

func scanAPIToken(row interface{ Scan(...any) error }) (*domain.APIToken, error) {
	var t domain.APIToken
	err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.TokenPrefix, &t.LastUsedAt, &t.RevokedAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Create inserts a token record.
func (r *APITokenRepo) Create(ctx context.Context, t *domain.APIToken) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.CreatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.UserID, t.Name, t.TokenHash, t.TokenPrefix, t.CreatedAt)
	return err
}

// ListByUser returns the user's live tokens, newest first.
func (r *APITokenRepo) ListByUser(ctx context.Context, userID string) ([]domain.APIToken, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+apiTokenColumns+` FROM api_tokens WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.APIToken
	for rows.Next() {
		t, err := scanAPIToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// GetActiveByHash looks up a non-revoked token by its SHA-256 hash.
func (r *APITokenRepo) GetActiveByHash(ctx context.Context, hash string) (*domain.APIToken, error) {
	return scanAPIToken(r.db.QueryRowContext(ctx, `SELECT `+apiTokenColumns+` FROM api_tokens WHERE token_hash = $1 AND revoked_at IS NULL`, hash))
}

// TouchLastUsed records token use.
func (r *APITokenRepo) TouchLastUsed(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE api_tokens SET last_used_at = $2 WHERE id = $1`, id, at)
	return err
}

// Revoke soft-deletes a token owned by the user.
func (r *APITokenRepo) Revoke(ctx context.Context, userID, id string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE api_tokens SET revoked_at = NOW() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
