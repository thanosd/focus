// Package handlers contains the HTTP layer: JSON helpers, middleware and
// one handler type per resource family.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/thanosd/focus/backend/internal/domain"
)

// writeJSONError writes { "message": "..." } per the OpenAPI Error schema.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
	}{Message: message})
}

// writeJSON encodes v with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("handlers: encode response: %v", err)
	}
}

// writeServiceError maps domain sentinel errors to HTTP statuses. Any
// other error is a 500 with a generic message (details go to the log).
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "Not found")
	case errors.Is(err, domain.ErrValidation):
		writeJSONError(w, http.StatusBadRequest, strings.TrimPrefix(err.Error(), domain.ErrValidation.Error()+": "))
	case errors.Is(err, domain.ErrForbidden):
		writeJSONError(w, http.StatusForbidden, "Forbidden")
	default:
		log.Printf("handlers: internal error: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Internal server error")
	}
}

// decodeJSON reads a JSON body (max 1 MiB) into v.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v)
}

// pathUUID validates a {id} path value.
func pathUUID(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	v := r.PathValue(name)
	if _, err := uuid.Parse(v); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid "+name)
		return "", false
	}
	return v, true
}

type ctxKey int

const userKey ctxKey = iota

// WithUser stores the authenticated user on the context.
func WithUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFromContext returns the authenticated user (nil when absent).
func UserFromContext(ctx context.Context) *domain.User {
	u, _ := ctx.Value(userKey).(*domain.User)
	return u
}

// mustUser returns the context user or writes a 401.
func mustUser(w http.ResponseWriter, r *http.Request) *domain.User {
	u := UserFromContext(r.Context())
	if u == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
	}
	return u
}

// CORSMiddleware adds CORS headers for the allowed browser origins.
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
				w.Header().Add("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ── OpenAPI type conversion helpers ──────────────────────────────────

func uuidVal(s string) openapi_types.UUID {
	u, _ := uuid.Parse(s)
	return u
}

func uuidPtr(s *string) *openapi_types.UUID {
	if s == nil {
		return nil
	}
	u := uuidVal(*s)
	return &u
}

func uuidStr(u *openapi_types.UUID) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}

func uuidStrs(us *[]openapi_types.UUID) []string {
	if us == nil {
		return nil
	}
	out := make([]string, 0, len(*us))
	for _, u := range *us {
		out = append(out, u.String())
	}
	return out
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
