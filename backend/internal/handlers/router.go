package handlers

import (
	"encoding/json"
	"net/http"
)

// Deps are the handlers the router mounts.
type Deps struct {
	Auth     *AuthHandler
	Tasks    *TaskHandler
	Projects *ProjectHandler
	Tags     *TagHandler
	// MCP is mounted at /mcp outside the CSRF/session middleware; it
	// carries its own bearer-token auth.
	MCP http.Handler
	// CORS and CSRF wrap every /api route.
	CORS func(http.Handler) http.Handler
	CSRF func(http.Handler) http.Handler
}

// NewRouter builds the http.Handler for the API.
func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()

	// Browser-facing API: CORS → CSRF → (session) → handler.
	public := func(h http.HandlerFunc) http.Handler { return d.CORS(d.CSRF(h)) }
	private := func(h http.HandlerFunc) http.Handler { return d.CORS(d.CSRF(d.Auth.RequireSession(h))) }

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// CSRF priming endpoint kept for client compatibility: the Fetch
	// metadata based middleware doesn't use tokens, so it's always empty.
	mux.Handle("GET /api/csrf", public(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"csrfToken": ""})
	}))

	// Auth
	mux.Handle("GET /api/auth/google/login", public(d.Auth.HandleGoogleLogin))
	mux.Handle("GET /api/auth/google/callback", public(d.Auth.HandleGoogleCallback))
	mux.Handle("POST /api/auth/logout", public(d.Auth.HandleLogout))
	mux.Handle("GET /api/auth/me", public(d.Auth.HandleMe))
	mux.Handle("GET /api/user-preferences", private(d.Auth.HandleGetPreferences))
	mux.Handle("PATCH /api/user-preferences", private(d.Auth.HandleUpdatePreferences))
	mux.Handle("GET /api/api-tokens", private(d.Auth.HandleListAPITokens))
	mux.Handle("POST /api/api-tokens", private(d.Auth.HandleCreateAPIToken))
	mux.Handle("DELETE /api/api-tokens/{tokenId}", private(d.Auth.HandleDeleteAPIToken))

	// Tasks
	mux.Handle("GET /api/tasks", private(d.Tasks.HandleList))
	mux.Handle("POST /api/tasks", private(d.Tasks.HandleCreate))
	mux.Handle("POST /api/tasks/reorder", private(d.Tasks.HandleReorder))
	mux.Handle("GET /api/tasks/{taskId}", private(d.Tasks.HandleGet))
	mux.Handle("PATCH /api/tasks/{taskId}", private(d.Tasks.HandleUpdate))
	mux.Handle("DELETE /api/tasks/{taskId}", private(d.Tasks.HandleDelete))
	mux.Handle("POST /api/tasks/{taskId}/complete", private(d.Tasks.HandleComplete))
	mux.Handle("POST /api/tasks/{taskId}/drop", private(d.Tasks.HandleDrop))
	mux.Handle("POST /api/tasks/{taskId}/reopen", private(d.Tasks.HandleReopen))
	mux.Handle("POST /api/tasks/{taskId}/defer", private(d.Tasks.HandleDefer))
	mux.Handle("POST /api/dates/parse", private(d.Tasks.HandleParseDate))
	mux.Handle("GET /api/counts", private(d.Tasks.HandleCounts))

	// Projects + reviews
	mux.Handle("GET /api/projects", private(d.Projects.HandleList))
	mux.Handle("POST /api/projects", private(d.Projects.HandleCreate))
	mux.Handle("POST /api/projects/reorder", private(d.Projects.HandleReorder))
	mux.Handle("GET /api/projects/{projectId}", private(d.Projects.HandleGet))
	mux.Handle("PATCH /api/projects/{projectId}", private(d.Projects.HandleUpdate))
	mux.Handle("DELETE /api/projects/{projectId}", private(d.Projects.HandleDelete))
	mux.Handle("POST /api/projects/{projectId}/review", private(d.Projects.HandleReview))
	mux.Handle("GET /api/reviews", private(d.Projects.HandleReviews))

	// Tags
	mux.Handle("GET /api/tags", private(d.Tags.HandleList))
	mux.Handle("POST /api/tags", private(d.Tags.HandleCreate))
	mux.Handle("PATCH /api/tags/{tagId}", private(d.Tags.HandleUpdate))
	mux.Handle("DELETE /api/tags/{tagId}", private(d.Tags.HandleDelete))

	// Preflight for any /api path (CORS middleware answers OPTIONS).
	mux.Handle("OPTIONS /api/", d.CORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	// MCP (bearer token auth inside the handler; no cookies, no CSRF).
	if d.MCP != nil {
		mux.Handle("/mcp", d.MCP)
	}

	return mux
}
