package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/thanosd/focus/backend/internal/config"
	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/ports"
)

// Auth error codes surfaced to the frontend via /login?error=<code>.
const (
	AuthErrNotAllowed    = "not_allowed"
	AuthErrOAuthFailed   = "oauth_failed"
	AuthErrStateMismatch = "state_mismatch"
)

// ErrEmailNotAllowed is returned when a Google account is not on the allowlist.
var ErrEmailNotAllowed = errors.New("email is not on the sign-in allowlist")

// ErrInvalidState is returned when the OAuth state nonce is missing or expired.
var ErrInvalidState = errors.New("invalid or expired sign-in state")

// AuthService owns Google sign-in, sessions and API tokens.
type AuthService struct {
	cfg        *config.Config
	users      ports.UserRepository
	sessions   ports.SessionRepository
	states     ports.OAuthStateRepository
	tokens     ports.APITokenRepository
	oauth      *oauth2.Config
	verifier   *oidc.IDTokenVerifier
	providerOK bool
}

// NewAuthService wires the Google OIDC provider. Discovery needs network
// access; a failure is returned so the server refuses to start half-configured.
func NewAuthService(ctx context.Context, cfg *config.Config, users ports.UserRepository, sessions ports.SessionRepository, states ports.OAuthStateRepository, tokens ports.APITokenRepository) (*AuthService, error) {
	s := &AuthService{cfg: cfg, users: users, sessions: sessions, states: states, tokens: tokens}
	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		return s, nil // token/session paths still work; sign-in reports not configured
	}
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("google oidc discovery: %w", err)
	}
	s.oauth = &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
	s.verifier = provider.Verifier(&oidc.Config{ClientID: cfg.GoogleClientID})
	s.providerOK = true
	return s, nil
}

// SignInConfigured reports whether Google OAuth is wired.
func (s *AuthService) SignInConfigured() bool { return s.providerOK }

// BeginGoogleLogin stores a state nonce and returns the Google URL.
func (s *AuthService) BeginGoogleLogin(ctx context.Context, redirectTo string) (string, error) {
	if !s.providerOK {
		return "", errors.New("google sign-in is not configured")
	}
	state, err := randomToken(24)
	if err != nil {
		return "", err
	}
	if err := s.states.Create(ctx, &domain.OAuthState{
		State:      state,
		RedirectTo: safeRedirect(redirectTo),
		ExpiresAt:  time.Now().Add(10 * time.Minute),
	}); err != nil {
		return "", fmt.Errorf("store oauth state: %w", err)
	}
	return s.oauth.AuthCodeURL(state, oauth2.AccessTypeOnline), nil
}

// GoogleClaims are the ID token fields we read.
type GoogleClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// CompleteGoogleLogin exchanges the code, verifies the identity, enforces
// the allowlist, upserts the user and opens a session. It returns the
// session ID and the post-login redirect path.
func (s *AuthService) CompleteGoogleLogin(ctx context.Context, state, code string) (sessionID, redirectTo string, err error) {
	if !s.providerOK {
		return "", "", errors.New("google sign-in is not configured")
	}
	st, err := s.states.Consume(ctx, state)
	if err != nil {
		return "", "", fmt.Errorf("consume oauth state: %w", err)
	}
	if st == nil {
		return "", "", ErrInvalidState
	}
	redirectTo = st.RedirectTo

	token, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		return "", redirectTo, fmt.Errorf("exchange code: %w", err)
	}
	rawID, ok := token.Extra("id_token").(string)
	if !ok || rawID == "" {
		return "", redirectTo, errors.New("google response had no id_token")
	}
	idToken, err := s.verifier.Verify(ctx, rawID)
	if err != nil {
		return "", redirectTo, fmt.Errorf("verify id_token: %w", err)
	}
	var claims GoogleClaims
	if err := idToken.Claims(&claims); err != nil {
		return "", redirectTo, fmt.Errorf("decode claims: %w", err)
	}
	if !claims.EmailVerified || claims.Email == "" {
		return "", redirectTo, errors.New("google account email is not verified")
	}
	if !s.cfg.IsEmailAllowed(claims.Email) {
		log.Printf("auth: sign-in refused for %s (not on allowlist)", claims.Email)
		return "", redirectTo, ErrEmailNotAllowed
	}

	user, err := s.users.UpsertFromGoogle(ctx, claims.Email, claims.Sub, claims.Name, claims.Picture)
	if err != nil {
		return "", redirectTo, fmt.Errorf("upsert user: %w", err)
	}
	session := &domain.Session{UserID: user.ID, ExpiresAt: time.Now().Add(s.SessionDuration())}
	if err := s.sessions.Create(ctx, session); err != nil {
		return "", redirectTo, fmt.Errorf("create session: %w", err)
	}
	if err := s.users.UpdateLastLoginAt(ctx, user.ID, time.Now()); err != nil {
		log.Printf("auth: failed to record last_login_at for %s: %v", user.ID, err)
	}
	return session.ID, redirectTo, nil
}

// SessionDuration is how long a browser session lives.
func (s *AuthService) SessionDuration() time.Duration {
	return time.Duration(s.cfg.SessionExpireHours) * time.Hour
}

// ValidateSession resolves a session cookie to its user.
func (s *AuthService) ValidateSession(ctx context.Context, sessionID string) (*domain.User, error) {
	if sessionID == "" {
		return nil, errors.New("missing session")
	}
	session, err := s.sessions.GetValid(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, errors.New("session not found or expired")
	}
	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// UserFromRequest reads the session cookie and resolves the user.
func (s *AuthService) UserFromRequest(r *http.Request) (*domain.User, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil, errors.New("no session cookie")
	}
	return s.ValidateSession(r.Context(), cookie.Value)
}

// SessionCookieName is the browser session cookie.
const SessionCookieName = "session_id"

// Logout deletes the session.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.sessions.Delete(ctx, sessionID)
}

// UpdateTimezone validates and stores an IANA timezone.
func (s *AuthService) UpdateTimezone(ctx context.Context, userID, tz string) error {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return fmt.Errorf("%w: timezone is required", domain.ErrValidation)
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return fmt.Errorf("%w: unknown timezone %q", domain.ErrValidation, tz)
	}
	return s.users.UpdateTimezone(ctx, userID, tz)
}

// ── API tokens (MCP) ─────────────────────────────────────────────────

const apiTokenPrefix = "fcs_"

// CreateAPIToken mints a token. The plaintext is returned exactly once.
func (s *AuthService) CreateAPIToken(ctx context.Context, userID, name string) (plaintext string, token *domain.APIToken, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	secret, err := randomToken(32)
	if err != nil {
		return "", nil, err
	}
	plaintext = apiTokenPrefix + secret
	token = &domain.APIToken{
		UserID:      userID,
		Name:        name,
		TokenHash:   HashAPIToken(plaintext),
		TokenPrefix: plaintext[:len(apiTokenPrefix)+4],
	}
	if err := s.tokens.Create(ctx, token); err != nil {
		return "", nil, fmt.Errorf("create api token: %w", err)
	}
	return plaintext, token, nil
}

// ListAPITokens returns the user's live tokens.
func (s *AuthService) ListAPITokens(ctx context.Context, userID string) ([]domain.APIToken, error) {
	return s.tokens.ListByUser(ctx, userID)
}

// RevokeAPIToken soft-deletes a token.
func (s *AuthService) RevokeAPIToken(ctx context.Context, userID, id string) error {
	return s.tokens.Revoke(ctx, userID, id)
}

// UserFromAPIToken resolves a bearer token to its user, touching
// last_used_at. Returns nil, nil for unknown tokens.
func (s *AuthService) UserFromAPIToken(ctx context.Context, plaintext string) (*domain.User, error) {
	plaintext = strings.TrimSpace(plaintext)
	if !strings.HasPrefix(plaintext, apiTokenPrefix) {
		return nil, nil
	}
	tok, err := s.tokens.GetActiveByHash(ctx, HashAPIToken(plaintext))
	if err != nil || tok == nil {
		return nil, err
	}
	user, err := s.users.GetByID(ctx, tok.UserID)
	if err != nil || user == nil {
		return nil, err
	}
	if err := s.tokens.TouchLastUsed(ctx, tok.ID, time.Now()); err != nil {
		log.Printf("auth: failed to touch api token %s: %v", tok.ID, err)
	}
	return user, nil
}

// HashAPIToken is the stored form of a token.
func HashAPIToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// safeRedirect only allows same-site relative paths.
func safeRedirect(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") || strings.Contains(p, "\\") {
		return "/"
	}
	return p
}
