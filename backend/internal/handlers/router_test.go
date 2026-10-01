package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thanosd/focus/backend/internal/adapters"
	"github.com/thanosd/focus/backend/internal/api"
	"github.com/thanosd/focus/backend/internal/config"
	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/handlers"
	"github.com/thanosd/focus/backend/internal/mcpserver"
	"github.com/thanosd/focus/backend/internal/services"
	"github.com/thanosd/focus/backend/internal/services/dateparse"
	"github.com/thanosd/focus/backend/internal/testdb"
)

func passthrough(next http.Handler) http.Handler { return next }

func TestRouterEndToEnd(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	cfg := &config.Config{Environment: "development", SessionExpireHours: 1, FrontendURL: "http://localhost:3000", AuthAllowedEmails: []string{"me@example.com"}}

	users := adapters.NewUserRepository(db)
	sessions := adapters.NewSessionRepository(db)
	states := adapters.NewOAuthStateRepository(db)
	tokens := adapters.NewAPITokenRepository(db)
	projects := adapters.NewProjectRepository(db)
	tasks := adapters.NewTaskRepository(db)
	tags := adapters.NewTagRepository(db)

	auth, err := services.NewAuthService(ctx, cfg, users, sessions, states, tokens)
	if err != nil {
		t.Fatal(err)
	}
	taskSvc := services.NewTaskService(tasks, projects, dateparse.NewParser(nil))
	projSvc := services.NewProjectService(projects, tasks)
	tagSvc := services.NewTagService(tags)
	mcp := mcpserver.New(mcpserver.Deps{Auth: auth, Tasks: taskSvc, Projects: projSvc, Tags: tagSvc, Users: users, Version: "test"})

	router := handlers.NewRouter(handlers.Deps{
		Auth:     handlers.NewAuthHandler(cfg, auth),
		Tasks:    handlers.NewTaskHandler(taskSvc),
		Projects: handlers.NewProjectHandler(projSvc),
		Tags:     handlers.NewTagHandler(tagSvc),
		MCP:      mcp.Handler(),
		CORS:     passthrough,
		CSRF:     passthrough,
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Unauthenticated
	resp, _ := http.Get(srv.URL + "/api/tasks")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	resp, _ = http.Get(srv.URL + "/health")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health: %d", resp.StatusCode)
	}

	// Seed a user + session directly (Google round trip needs the network).
	user, err := users.UpsertFromGoogle(ctx, "me@example.com", "sub", "Me", "")
	if err != nil {
		t.Fatal(err)
	}
	sess := &domain.Session{UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)}
	if err := sessions.Create(ctx, sess); err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{Name: services.SessionCookieName, Value: sess.ID}

	do := func(method, path string, body any, out any) int {
		t.Helper()
		var rdr *strings.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rdr = strings.NewReader(string(b))
		} else {
			rdr = strings.NewReader("")
		}
		req, _ := http.NewRequest(method, srv.URL+path, rdr)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		if out != nil {
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil && resp.StatusCode < 300 {
				t.Fatalf("%s %s: decode: %v", method, path, err)
			}
		}
		return resp.StatusCode
	}

	var me api.MeResponse
	if code := do("GET", "/api/auth/me", nil, &me); code != 200 || string(me.User.Email) != "me@example.com" {
		t.Fatalf("me: %d %+v", code, me)
	}

	var created api.Task
	if code := do("POST", "/api/tasks", map[string]any{"title": "Call the bank", "flagged": true}, &created); code != 201 {
		t.Fatalf("create task: %d", code)
	}
	if created.ProjectId != nil || !created.Flagged || !created.IsAvailable {
		t.Fatalf("created task: %+v", created)
	}

	var inbox []api.Task
	if code := do("GET", "/api/tasks?view=inbox", nil, &inbox); code != 200 || len(inbox) != 1 {
		t.Fatalf("inbox: %d %d", code, len(inbox))
	}

	var project api.Project
	if code := do("POST", "/api/projects", map[string]any{"name": "Finance"}, &project); code != 201 {
		t.Fatalf("create project: %d", code)
	}

	// PATCH with explicit null clears; moving to a project removes from inbox.
	var updated api.Task
	code := do("PATCH", "/api/tasks/"+created.Id.String(), map[string]any{"project_id": project.Id.String(), "due_at": "2030-01-02T17:00:00Z"}, &updated)
	if code != 200 || updated.ProjectId == nil || updated.DueAt == nil {
		t.Fatalf("patch: %d %+v", code, updated)
	}
	var cleared api.Task
	code = do("PATCH", "/api/tasks/"+created.Id.String(), map[string]any{"due_at": nil}, &cleared)
	if code != 200 || cleared.DueAt != nil || cleared.ProjectId == nil {
		t.Fatalf("patch null: %d %+v", code, cleared)
	}

	var deferred api.Task
	if code := do("POST", "/api/tasks/"+created.Id.String()+"/defer", map[string]any{"input": "1w", "timezone": "America/New_York"}, &deferred); code != 200 || deferred.DeferUntil == nil || deferred.IsAvailable {
		t.Fatalf("defer: %d %+v", code, deferred)
	}
	var bad api.Error
	if code := do("POST", "/api/tasks/"+created.Id.String()+"/defer", map[string]any{"input": "when pigs fly"}, &bad); code != 400 || !strings.Contains(bad.Message, "pigs") {
		t.Fatalf("defer nonsense: %d %+v", code, bad)
	}

	var parsed api.ParseDateResponse
	if code := do("POST", "/api/dates/parse", map[string]any{"input": "next friday", "kind": "due", "timezone": "UTC"}, &parsed); code != 200 || parsed.Source != "rules" || parsed.ResolvedAt.Hour() != 17 {
		t.Fatalf("parse: %d %+v", code, parsed)
	}

	var counts api.Counts
	if code := do("GET", "/api/counts", nil, &counts); code != 200 || counts.Inbox != 0 || counts.Flagged != 1 {
		t.Fatalf("counts: %d %+v", code, counts)
	}

	var done api.CompleteTaskResponse
	if code := do("POST", "/api/tasks/"+created.Id.String()+"/complete", nil, &done); code != 200 || done.Task.Status != api.TaskStatusCompleted || done.NextTask != nil {
		t.Fatalf("complete: %d %+v", code, done)
	}

	// API token + MCP endpoint auth
	var tok api.CreateApiTokenResponse
	if code := do("POST", "/api/api-tokens", map[string]any{"name": "cli"}, &tok); code != 201 || !strings.HasPrefix(tok.Token, "fcs_") {
		t.Fatalf("token: %d %+v", code, tok)
	}
	mcpReq := func(bearer string) int {
		body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`
		req, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	if code := mcpReq(""); code != http.StatusUnauthorized {
		t.Fatalf("mcp without token: %d", code)
	}
	if code := mcpReq("fcs_bogus"); code != http.StatusUnauthorized {
		t.Fatalf("mcp bad token: %d", code)
	}
	if code := mcpReq(tok.Token); code != http.StatusOK {
		t.Fatalf("mcp initialize: %d", code)
	}
}
