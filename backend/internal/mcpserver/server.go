// Package mcpserver exposes Focus to MCP clients (Claude Code, Claude
// Desktop, claude.ai connectors) over the streamable HTTP transport.
//
// Authentication is a personal API token (created in Settings) sent as
// `Authorization: Bearer fcs_...`. Every tool runs as that token's user
// and goes through the same services as the web UI.
package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/ports"
	"github.com/thanosd/focus/backend/internal/services"
	"github.com/thanosd/focus/backend/internal/services/dateparse"
	"github.com/thanosd/focus/backend/internal/services/repeatparse"
)

// Deps are the services the tools call.
type Deps struct {
	Auth     *services.AuthService
	Tasks    *services.TaskService
	Projects *services.ProjectService
	Tags     *services.TagService
	Users    ports.UserRepository
	Version  string
}

// Server wraps the MCP server and its HTTP handler.
type Server struct {
	deps   Deps
	server *mcp.Server
}

// New builds the MCP server with every Focus tool registered.
func New(deps Deps) *Server {
	s := &Server{deps: deps}
	s.server = mcp.NewServer(&mcp.Implementation{Name: "focus", Version: deps.Version}, &mcp.ServerOptions{
		Instructions: "Focus is the user's personal task manager (OmniFocus-style). Tasks live in the inbox (no project) or in a project; projects nest up to three levels (bucket > project > sub-project; buckets like 'Personal' or 'Work' are just containers); tags cross-cut; flagged = urgent. Deferred tasks are hidden until defer_until. Use list_tasks with view=inbox to see unprocessed captures, view=available for what can be worked on now. Dates accept natural language (\"1w\", \"next monday\", \"in 3 days\").",
	})
	s.registerTools()
	return s
}

// Handler returns the bearer-protected streamable HTTP handler for /mcp.
func (s *Server) Handler() http.Handler {
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s.server }, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
		Logger:       slog.Default(),
	})
	verify := func(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		user, err := s.deps.Auth.UserFromAPIToken(ctx, token)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, auth.ErrInvalidToken
		}
		return &auth.TokenInfo{UserID: user.ID, Expiration: time.Now().Add(24 * time.Hour)}, nil
	}
	return auth.RequireBearerToken(verify, nil)(h)
}

// user resolves the request's token to its user.
func (s *Server) user(ctx context.Context, req *mcp.CallToolRequest) (*domain.User, error) {
	if req == nil || req.Extra == nil || req.Extra.TokenInfo == nil || req.Extra.TokenInfo.UserID == "" {
		return nil, errors.New("unauthenticated")
	}
	u, err := s.deps.Users.GetByID(ctx, req.Extra.TokenInfo.UserID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("user not found")
	}
	return u, nil
}

// ── Output shapes (compact, LLM-friendly) ────────────────────────────

type taskOut struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Note        string   `json:"note,omitempty"`
	Status      string   `json:"status"`
	Flagged     bool     `json:"flagged"`
	ProjectID   *string  `json:"project_id,omitempty"`
	ProjectName string   `json:"project_name,omitempty"`
	Tags        []string `json:"tags"`
	DeferUntil  *string  `json:"defer_until,omitempty"`
	DueAt       *string  `json:"due_at,omitempty"`
	Repeat      *string  `json:"repeat,omitempty"`
	NextDates   []string `json:"next_occurrences,omitempty"`
	IsAvailable bool     `json:"is_available"`
	CompletedAt *string  `json:"completed_at,omitempty"`
}

type projectOut struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Note               string  `json:"note,omitempty"`
	Status             string  `json:"status"`
	ParentID           *string `json:"parent_id,omitempty"`
	Sequential         bool    `json:"sequential"`
	ReviewIntervalDays int     `json:"review_interval_days"`
	LastReviewedAt     *string `json:"last_reviewed_at,omitempty"`
	NextReviewAt       *string `json:"next_review_at,omitempty"`
	RemainingTasks     int     `json:"remaining_task_count"`
	AvailableTasks     int     `json:"available_task_count"`
}

func fmtTime(t *time.Time, loc *time.Location) *string {
	if t == nil {
		return nil
	}
	s := t.In(loc).Format(time.RFC3339)
	return &s
}

func toTaskOut(t *domain.Task, loc *time.Location) taskOut {
	tags := make([]string, 0, len(t.Tags))
	for _, g := range t.Tags {
		tags = append(tags, g.Name)
	}
	out := taskOut{
		ID: t.ID, Title: t.Title, Note: t.Note, Status: string(t.Status), Flagged: t.Flagged,
		ProjectID: t.ProjectID, ProjectName: t.ProjectName, Tags: tags,
		DeferUntil: fmtTime(t.DeferUntil, loc), DueAt: fmtTime(t.DueAt, loc),
		IsAvailable: t.IsAvailable, CompletedAt: fmtTime(t.CompletedAt, loc),
	}
	if t.RepeatRule != nil {
		r := t.RepeatRule.Describe()
		out.Repeat = &r
		base := time.Now().In(loc)
		if t.RepeatRule.From == domain.RepeatFromDue {
			if t.DueAt != nil {
				base = t.DueAt.In(loc)
			} else if t.DeferUntil != nil {
				base = t.DeferUntil.In(loc)
			}
		}
		for _, u := range t.RepeatRule.Upcoming(base, 3) {
			out.NextDates = append(out.NextDates, u.In(loc).Format(time.RFC3339))
		}
	}
	return out
}

func toProjectOut(p *domain.Project, loc *time.Location) projectOut {
	return projectOut{
		ID: p.ID, Name: p.Name, Note: p.Note, Status: string(p.Status), ParentID: p.ParentID,
		Sequential: p.Sequential, ReviewIntervalDays: p.ReviewIntervalDays,
		LastReviewedAt: fmtTime(p.LastReviewedAt, loc), NextReviewAt: fmtTime(p.NextReviewAt, loc),
		RemainingTasks: p.RemainingTaskCount, AvailableTasks: p.AvailableTaskCount,
	}
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
}

func textResult(s string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}, nil, nil
}

// toolErr turns a service error into a tool-level error result so the
// model sees the message instead of a protocol failure.
func toolErr(err error) (*mcp.CallToolResult, any, error) {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, domain.ErrValidation.Error()+": ")
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: msg}}}, nil, nil
}

// ── Tool inputs ──────────────────────────────────────────────────────

type listTasksIn struct {
	View               string `json:"view,omitempty" jsonschema:"Which list: inbox (unprocessed captures), available (default; actionable now), flagged (urgent), due (has a due date), completed (recent), all (every active task)"`
	ProjectID          string `json:"project_id,omitempty" jsonschema:"Only tasks in this project"`
	Project            string `json:"project,omitempty" jsonschema:"Only tasks in the project with this name (case-insensitive) — alternative to project_id"`
	IncludeSubprojects bool   `json:"include_subprojects,omitempty" jsonschema:"With a project, also include tasks of all its sub-projects"`
	Tag                string `json:"tag,omitempty" jsonschema:"Only tasks carrying this tag name"`
	Query              string `json:"query,omitempty" jsonschema:"Case-insensitive search in title and note"`
}

type taskIDIn struct {
	TaskID string `json:"task_id" jsonschema:"The task's ID"`
}

type createTaskIn struct {
	Title     string   `json:"title" jsonschema:"Task title"`
	Note      string   `json:"note,omitempty" jsonschema:"Longer note / details"`
	ProjectID string   `json:"project_id,omitempty" jsonschema:"Project to file the task under; omit (and omit project) for the inbox"`
	Project   string   `json:"project,omitempty" jsonschema:"Project name instead of project_id (case-insensitive match)"`
	Flagged   bool     `json:"flagged,omitempty" jsonschema:"Mark as urgent"`
	Defer     string   `json:"defer,omitempty" jsonschema:"Defer until — natural language (\"1w\", \"next monday\", \"oct 5\") or RFC 3339"`
	Due       string   `json:"due,omitempty" jsonschema:"Due date — natural language or RFC 3339"`
	Repeat    string   `json:"repeat,omitempty" jsonschema:"Repeat schedule in plain English: \"first of every month\", \"every other friday\", \"weekdays\", \"every 2 weeks after completion\". Calendar-anchored schedules set a due date when the task has none."`
	Tags      []string `json:"tags,omitempty" jsonschema:"Tag names; missing tags are created"`
}

type updateTaskIn struct {
	TaskID    string   `json:"task_id" jsonschema:"The task's ID"`
	Title     *string  `json:"title,omitempty"`
	Note      *string  `json:"note,omitempty"`
	ProjectID *string  `json:"project_id,omitempty" jsonschema:"Move to this project"`
	Project   *string  `json:"project,omitempty" jsonschema:"Move to the project with this name; use \"inbox\" to move back to the inbox"`
	Flagged   *bool    `json:"flagged,omitempty"`
	Defer     *string  `json:"defer,omitempty" jsonschema:"New defer date (natural language or RFC 3339); empty string clears"`
	Due       *string  `json:"due,omitempty" jsonschema:"New due date (natural language or RFC 3339); empty string clears"`
	Repeat    *string  `json:"repeat,omitempty" jsonschema:"New repeat schedule in plain English (\"first of every month\", \"every monday\", \"every 3 months after completion\"); empty string clears"`
	Tags      []string `json:"tags,omitempty" jsonschema:"Replace the tag set with these names (empty list clears)"`
}

type reorderTasksIn struct {
	TaskIDs []string `json:"task_ids" jsonschema:"Task IDs in the desired order (first = top). For sequential projects this decides which task is available."`
}

type deferTaskIn struct {
	TaskID string `json:"task_id" jsonschema:"The task's ID"`
	Until  string `json:"until" jsonschema:"When to defer until: \"1d\", \"1w\", \"1m\", \"tomorrow\", \"next monday\", \"in 3 days\", \"mid october\", or an RFC 3339 timestamp"`
}

type listProjectsIn struct {
	Status string `json:"status,omitempty" jsonschema:"active, on_hold, completed, dropped, or all; default lists active + on_hold"`
}

type projectIDIn struct {
	ProjectID string `json:"project_id" jsonschema:"The project's ID"`
}

type createProjectIn struct {
	Name               string `json:"name" jsonschema:"Project name"`
	Note               string `json:"note,omitempty"`
	ParentID           string `json:"parent_id,omitempty" jsonschema:"Parent project ID (three levels max: bucket > project > sub-project)"`
	Parent             string `json:"parent,omitempty" jsonschema:"Parent project name instead of parent_id"`
	Sequential         bool   `json:"sequential,omitempty" jsonschema:"Only the first remaining task is available at a time"`
	ReviewIntervalDays int    `json:"review_interval_days,omitempty" jsonschema:"How often to review (default 7)"`
}

type updateProjectIn struct {
	ProjectID          string  `json:"project_id" jsonschema:"The project's ID"`
	Name               *string `json:"name,omitempty"`
	Note               *string `json:"note,omitempty"`
	Status             *string `json:"status,omitempty" jsonschema:"active, on_hold, completed or dropped"`
	Sequential         *bool   `json:"sequential,omitempty"`
	ReviewIntervalDays *int    `json:"review_interval_days,omitempty"`
}

type reorderProjectsIn struct {
	ProjectIDs []string `json:"project_ids" jsonschema:"Sibling project IDs in the desired order (first = top)"`
}

type createTagIn struct {
	Name  string `json:"name" jsonschema:"Tag name"`
	Color string `json:"color,omitempty" jsonschema:"#rrggbb"`
}

type parseDateIn struct {
	Input string `json:"input" jsonschema:"The phrase to resolve"`
	Kind  string `json:"kind,omitempty" jsonschema:"defer (default; start of day) or due (17:00)"`
}

type emptyIn struct{}

// ── Registration ─────────────────────────────────────────────────────

func (s *Server) registerTools() {
	mcp.AddTool(s.server, &mcp.Tool{Name: "list_tasks", Description: "List tasks. Default view is 'available' (actionable now). Use view=inbox for unprocessed captures, flagged for urgent, due for dated tasks."}, s.listTasks)
	mcp.AddTool(s.server, &mcp.Tool{Name: "get_task", Description: "Get one task by ID, including note, tags, dates and repeat rule."}, s.getTask)
	mcp.AddTool(s.server, &mcp.Tool{Name: "create_task", Description: "Create a task. Without a project it goes to the inbox. Dates accept natural language."}, s.createTask)
	mcp.AddTool(s.server, &mcp.Tool{Name: "update_task", Description: "Update a task's title, note, project, flag, dates, repeat rule or tags. Only provided fields change; empty string clears a date/repeat."}, s.updateTask)
	mcp.AddTool(s.server, &mcp.Tool{Name: "complete_task", Description: "Mark a task completed. Repeating tasks spawn and return their next occurrence."}, s.completeTask)
	mcp.AddTool(s.server, &mcp.Tool{Name: "drop_task", Description: "Drop (abandon) a task without completing it."}, s.dropTask)
	mcp.AddTool(s.server, &mcp.Tool{Name: "reopen_task", Description: "Make a completed or dropped task active again."}, s.reopenTask)
	mcp.AddTool(s.server, &mcp.Tool{Name: "delete_task", Description: "Permanently delete a task."}, s.deleteTask)
	mcp.AddTool(s.server, &mcp.Tool{Name: "reorder_tasks", Description: "Set the order of tasks within a project or the inbox (first ID is first). In sequential projects the first active task is the available one."}, s.reorderTasks)
	mcp.AddTool(s.server, &mcp.Tool{Name: "defer_task", Description: "Defer a task until a date/time given in natural language (\"1w\", \"next monday\", \"in 3 days\", \"mid october\")."}, s.deferTask)
	mcp.AddTool(s.server, &mcp.Tool{Name: "list_projects", Description: "List projects with their parent_id (up to three levels: bucket > project > sub-project), status, review dates and task counts."}, s.listProjects)
	mcp.AddTool(s.server, &mcp.Tool{Name: "get_project", Description: "Get a project with its active tasks and sub-projects."}, s.getProject)
	mcp.AddTool(s.server, &mcp.Tool{Name: "create_project", Description: "Create a project, optionally nested under a parent (three levels max)."}, s.createProject)
	mcp.AddTool(s.server, &mcp.Tool{Name: "update_project", Description: "Rename a project, change its note, status (active/on_hold/completed/dropped), sequential flag or review interval."}, s.updateProject)
	mcp.AddTool(s.server, &mcp.Tool{Name: "reorder_projects", Description: "Set the display order of sibling projects (first ID is first)."}, s.reorderProjects)
	mcp.AddTool(s.server, &mcp.Tool{Name: "list_reviews", Description: "List projects that are due for review (oldest first) and the upcoming review schedule."}, s.listReviews)
	mcp.AddTool(s.server, &mcp.Tool{Name: "mark_project_reviewed", Description: "Mark a project as reviewed now and schedule its next review."}, s.markReviewed)
	mcp.AddTool(s.server, &mcp.Tool{Name: "list_tags", Description: "List tags with active task counts."}, s.listTags)
	mcp.AddTool(s.server, &mcp.Tool{Name: "create_tag", Description: "Create a tag."}, s.createTag)
	mcp.AddTool(s.server, &mcp.Tool{Name: "parse_date", Description: "Resolve a natural-language date phrase to a timestamp in the user's timezone without changing anything."}, s.parseDate)
	mcp.AddTool(s.server, &mcp.Tool{Name: "get_counts", Description: "Badge counts: inbox, flagged, due soon, overdue, reviews due."}, s.getCounts)
}

// ── Tool implementations ─────────────────────────────────────────────

func (s *Server) resolveProjectID(ctx context.Context, userID, id, name string) (*string, error) {
	if id != "" {
		if _, err := s.deps.Projects.Get(ctx, userID, id); err != nil {
			return nil, err
		}
		return &id, nil
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.EqualFold(name, "inbox") {
		return nil, nil
	}
	projects, err := s.deps.Projects.List(ctx, userID, "all")
	if err != nil {
		return nil, err
	}
	var partial *string
	for i := range projects {
		if strings.EqualFold(projects[i].Name, name) {
			return &projects[i].ID, nil
		}
		if partial == nil && strings.Contains(strings.ToLower(projects[i].Name), strings.ToLower(name)) {
			partial = &projects[i].ID
		}
	}
	if partial != nil {
		return partial, nil
	}
	return nil, fmt.Errorf("%w: no project named %q", domain.ErrValidation, name)
}

func (s *Server) parseWhen(ctx context.Context, user *domain.User, input string, kind dateparse.Kind) (*time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return &t, nil
	}
	res, err := s.deps.Tasks.ParseDate(ctx, dateparse.Request{Input: input, Kind: kind, Location: user.Location()})
	if err != nil {
		return nil, err
	}
	return &res.At, nil
}

// parseRepeat resolves a repeat phrase through the shared parser so MCP
// and the UI accept the same vocabulary ("first of every month", "every
// other friday", "every 2 weeks after completion").
func (s *Server) parseRepeat(ctx context.Context, user *domain.User, in string) (*repeatparse.Result, error) {
	if strings.TrimSpace(in) == "" {
		return nil, nil
	}
	return s.deps.Tasks.ParseRepeat(ctx, repeatparse.Request{Input: in, Location: user.Location()})
}

func (s *Server) listTasks(ctx context.Context, req *mcp.CallToolRequest, in listTasksIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	f := domain.TaskFilter{View: domain.TaskView(in.View), Query: in.Query}
	if in.ProjectID != "" || in.Project != "" {
		pid, err := s.resolveProjectID(ctx, user.ID, in.ProjectID, in.Project)
		if err != nil {
			return toolErr(err)
		}
		f.ProjectID = pid
		f.IncludeSubprojects = in.IncludeSubprojects
	}
	if in.Tag != "" {
		tags, err := s.deps.Tags.List(ctx, user.ID)
		if err != nil {
			return nil, nil, err
		}
		for i := range tags {
			if strings.EqualFold(tags[i].Name, in.Tag) {
				f.TagID = &tags[i].ID
			}
		}
		if f.TagID == nil {
			return toolErr(fmt.Errorf("no tag named %q", in.Tag))
		}
	}
	tasks, err := s.deps.Tasks.List(ctx, user.ID, f)
	if err != nil {
		return toolErr(err)
	}
	loc := user.Location()
	out := make([]taskOut, 0, len(tasks))
	for i := range tasks {
		out = append(out, toTaskOut(&tasks[i], loc))
	}
	return jsonResult(map[string]any{"count": len(out), "now": time.Now().In(loc).Format(time.RFC3339), "tasks": out})
}

func (s *Server) getTask(ctx context.Context, req *mcp.CallToolRequest, in taskIDIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	t, err := s.deps.Tasks.Get(ctx, user.ID, in.TaskID)
	if err != nil {
		return toolErr(err)
	}
	return jsonResult(toTaskOut(t, user.Location()))
}

func (s *Server) createTask(ctx context.Context, req *mcp.CallToolRequest, in createTaskIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	pid, err := s.resolveProjectID(ctx, user.ID, in.ProjectID, in.Project)
	if err != nil {
		return toolErr(err)
	}
	deferAt, err := s.parseWhen(ctx, user, in.Defer, dateparse.KindDefer)
	if err != nil {
		return toolErr(err)
	}
	dueAt, err := s.parseWhen(ctx, user, in.Due, dateparse.KindDue)
	if err != nil {
		return toolErr(err)
	}
	parsedRepeat, err := s.parseRepeat(ctx, user, in.Repeat)
	if err != nil {
		return toolErr(err)
	}
	var rule *domain.RepeatRule
	if parsedRepeat != nil {
		r := parsedRepeat.Rule
		rule = &r
		if dueAt == nil && deferAt == nil && parsedRepeat.FirstOccurrence != nil {
			f := *parsedRepeat.FirstOccurrence
			d := time.Date(f.Year(), f.Month(), f.Day(), 17, 0, 0, 0, user.Location())
			dueAt = &d
		}
	}
	tagIDs, err := s.deps.Tags.EnsureByNames(ctx, user.ID, in.Tags)
	if err != nil {
		return toolErr(err)
	}
	t, err := s.deps.Tasks.Create(ctx, user.ID, services.CreateTaskInput{
		Title: in.Title, Note: in.Note, ProjectID: pid, Flagged: in.Flagged,
		DeferUntil: deferAt, DueAt: dueAt, RepeatRule: rule, TagIDs: tagIDs,
	})
	if err != nil {
		return toolErr(err)
	}
	return jsonResult(toTaskOut(t, user.Location()))
}

func (s *Server) updateTask(ctx context.Context, req *mcp.CallToolRequest, in updateTaskIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	patch := services.TaskPatch{Title: in.Title, Note: in.Note, Flagged: in.Flagged}
	if in.ProjectID != nil || in.Project != nil {
		id, name := "", ""
		if in.ProjectID != nil {
			id = *in.ProjectID
		}
		if in.Project != nil {
			name = *in.Project
		}
		pid, err := s.resolveProjectID(ctx, user.ID, id, name)
		if err != nil {
			return toolErr(err)
		}
		patch.SetProject = true
		patch.ProjectID = pid
	}
	if in.Defer != nil {
		patch.SetDefer = true
		if patch.DeferUntil, err = s.parseWhen(ctx, user, *in.Defer, dateparse.KindDefer); err != nil {
			return toolErr(err)
		}
	}
	if in.Due != nil {
		patch.SetDue = true
		if patch.DueAt, err = s.parseWhen(ctx, user, *in.Due, dateparse.KindDue); err != nil {
			return toolErr(err)
		}
	}
	if in.Repeat != nil {
		patch.SetRepeat = true
		parsedRepeat, err := s.parseRepeat(ctx, user, *in.Repeat)
		if err != nil {
			return toolErr(err)
		}
		if parsedRepeat != nil {
			r := parsedRepeat.Rule
			patch.RepeatRule = &r
		}
	}
	if in.Tags != nil {
		ids, err := s.deps.Tags.EnsureByNames(ctx, user.ID, in.Tags)
		if err != nil {
			return toolErr(err)
		}
		if ids == nil {
			ids = []string{}
		}
		patch.TagIDs = ids
	}
	t, err := s.deps.Tasks.Update(ctx, user.ID, in.TaskID, patch)
	if err != nil {
		return toolErr(err)
	}
	return jsonResult(toTaskOut(t, user.Location()))
}

func (s *Server) completeTask(ctx context.Context, req *mcp.CallToolRequest, in taskIDIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	done, next, err := s.deps.Tasks.Complete(ctx, user, in.TaskID)
	if err != nil {
		return toolErr(err)
	}
	loc := user.Location()
	out := map[string]any{"task": toTaskOut(done, loc)}
	if next != nil {
		out["next_task"] = toTaskOut(next, loc)
	}
	return jsonResult(out)
}

func (s *Server) dropTask(ctx context.Context, req *mcp.CallToolRequest, in taskIDIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	t, err := s.deps.Tasks.Drop(ctx, user.ID, in.TaskID)
	if err != nil {
		return toolErr(err)
	}
	return jsonResult(toTaskOut(t, user.Location()))
}

func (s *Server) reopenTask(ctx context.Context, req *mcp.CallToolRequest, in taskIDIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	t, err := s.deps.Tasks.Reopen(ctx, user.ID, in.TaskID)
	if err != nil {
		return toolErr(err)
	}
	return jsonResult(toTaskOut(t, user.Location()))
}

func (s *Server) deleteTask(ctx context.Context, req *mcp.CallToolRequest, in taskIDIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	if err := s.deps.Tasks.Delete(ctx, user.ID, in.TaskID); err != nil {
		return toolErr(err)
	}
	return textResult("deleted")
}

func (s *Server) reorderTasks(ctx context.Context, req *mcp.CallToolRequest, in reorderTasksIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	tasks, err := s.deps.Tasks.Reorder(ctx, user.ID, in.TaskIDs)
	if err != nil {
		return toolErr(err)
	}
	loc := user.Location()
	out := make([]taskOut, 0, len(tasks))
	for i := range tasks {
		out = append(out, toTaskOut(&tasks[i], loc))
	}
	return jsonResult(out)
}

func (s *Server) deferTask(ctx context.Context, req *mcp.CallToolRequest, in deferTaskIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	din := services.DeferInput{Input: in.Until}
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(in.Until)); err == nil {
		din = services.DeferInput{Until: &t}
	}
	t, parsed, err := s.deps.Tasks.Defer(ctx, user, in.TaskID, din)
	if err != nil {
		return toolErr(err)
	}
	out := map[string]any{"task": toTaskOut(t, user.Location())}
	if parsed != nil {
		out["interpretation"] = parsed.Interpretation
		out["source"] = parsed.Source
	}
	return jsonResult(out)
}

func (s *Server) listProjects(ctx context.Context, req *mcp.CallToolRequest, in listProjectsIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	projects, err := s.deps.Projects.List(ctx, user.ID, in.Status)
	if err != nil {
		return toolErr(err)
	}
	loc := user.Location()
	out := make([]projectOut, 0, len(projects))
	for i := range projects {
		out = append(out, toProjectOut(&projects[i], loc))
	}
	return jsonResult(map[string]any{"count": len(out), "projects": out})
}

func (s *Server) getProject(ctx context.Context, req *mcp.CallToolRequest, in projectIDIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	d, err := s.deps.Projects.GetDetail(ctx, user.ID, in.ProjectID)
	if err != nil {
		return toolErr(err)
	}
	loc := user.Location()
	tasks := make([]taskOut, 0, len(d.Tasks))
	for i := range d.Tasks {
		tasks = append(tasks, toTaskOut(&d.Tasks[i], loc))
	}
	children := make([]projectOut, 0, len(d.Children))
	for i := range d.Children {
		children = append(children, toProjectOut(&d.Children[i], loc))
	}
	return jsonResult(map[string]any{"project": toProjectOut(d.Project, loc), "tasks": tasks, "children": children})
}

func (s *Server) createProject(ctx context.Context, req *mcp.CallToolRequest, in createProjectIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	var parent *string
	if in.ParentID != "" || in.Parent != "" {
		parent, err = s.resolveProjectID(ctx, user.ID, in.ParentID, in.Parent)
		if err != nil {
			return toolErr(err)
		}
	}
	p, err := s.deps.Projects.Create(ctx, user.ID, services.CreateProjectInput{
		Name: in.Name, Note: in.Note, ParentID: parent, Sequential: in.Sequential, ReviewIntervalDays: in.ReviewIntervalDays,
	})
	if err != nil {
		return toolErr(err)
	}
	return jsonResult(toProjectOut(p, user.Location()))
}

func (s *Server) updateProject(ctx context.Context, req *mcp.CallToolRequest, in updateProjectIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	patch := services.ProjectPatch{Name: in.Name, Note: in.Note, Sequential: in.Sequential, ReviewIntervalDays: in.ReviewIntervalDays}
	if in.Status != nil {
		st := domain.ProjectStatus(*in.Status)
		patch.Status = &st
	}
	p, err := s.deps.Projects.Update(ctx, user.ID, in.ProjectID, patch)
	if err != nil {
		return toolErr(err)
	}
	return jsonResult(toProjectOut(p, user.Location()))
}

func (s *Server) reorderProjects(ctx context.Context, req *mcp.CallToolRequest, in reorderProjectsIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	projects, err := s.deps.Projects.Reorder(ctx, user.ID, in.ProjectIDs)
	if err != nil {
		return toolErr(err)
	}
	loc := user.Location()
	out := make([]projectOut, 0, len(projects))
	for i := range projects {
		out = append(out, toProjectOut(&projects[i], loc))
	}
	return jsonResult(out)
}

func (s *Server) listReviews(ctx context.Context, req *mcp.CallToolRequest, _ emptyIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	due, upcoming, err := s.deps.Projects.Reviews(ctx, user.ID)
	if err != nil {
		return toolErr(err)
	}
	loc := user.Location()
	conv := func(ps []domain.Project) []projectOut {
		out := make([]projectOut, 0, len(ps))
		for i := range ps {
			out = append(out, toProjectOut(&ps[i], loc))
		}
		return out
	}
	return jsonResult(map[string]any{"due": conv(due), "upcoming": conv(upcoming)})
}

func (s *Server) markReviewed(ctx context.Context, req *mcp.CallToolRequest, in projectIDIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	p, err := s.deps.Projects.MarkReviewed(ctx, user.ID, in.ProjectID)
	if err != nil {
		return toolErr(err)
	}
	return jsonResult(toProjectOut(p, user.Location()))
}

func (s *Server) listTags(ctx context.Context, req *mcp.CallToolRequest, _ emptyIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	tags, err := s.deps.Tags.List(ctx, user.ID)
	if err != nil {
		return toolErr(err)
	}
	type tagOut struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
		Count int    `json:"active_task_count"`
	}
	out := make([]tagOut, 0, len(tags))
	for _, g := range tags {
		out = append(out, tagOut{ID: g.ID, Name: g.Name, Color: g.Color, Count: g.ActiveTaskCount})
	}
	return jsonResult(out)
}

func (s *Server) createTag(ctx context.Context, req *mcp.CallToolRequest, in createTagIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	g, err := s.deps.Tags.Create(ctx, user.ID, in.Name, in.Color)
	if err != nil {
		return toolErr(err)
	}
	return jsonResult(map[string]any{"id": g.ID, "name": g.Name, "color": g.Color})
}

func (s *Server) parseDate(ctx context.Context, req *mcp.CallToolRequest, in parseDateIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	kind := dateparse.KindDefer
	if in.Kind == "due" {
		kind = dateparse.KindDue
	}
	res, err := s.deps.Tasks.ParseDate(ctx, dateparse.Request{Input: in.Input, Kind: kind, Location: user.Location()})
	if err != nil {
		return toolErr(err)
	}
	return jsonResult(map[string]any{"resolved_at": res.At.Format(time.RFC3339), "interpretation": res.Interpretation, "source": res.Source})
}

func (s *Server) getCounts(ctx context.Context, req *mcp.CallToolRequest, _ emptyIn) (*mcp.CallToolResult, any, error) {
	user, err := s.user(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	c, err := s.deps.Tasks.Counts(ctx, user.ID)
	if err != nil {
		return toolErr(err)
	}
	return jsonResult(map[string]int{"inbox": c.Inbox, "flagged": c.Flagged, "due_soon": c.DueSoon, "overdue": c.Overdue, "review_due": c.ReviewDue})
}
