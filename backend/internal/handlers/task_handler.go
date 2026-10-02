package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/thanosd/focus/backend/internal/api"
	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/services"
	"github.com/thanosd/focus/backend/internal/services/dateparse"
	"github.com/thanosd/focus/backend/internal/services/repeatparse"
)

// TaskHandler serves /api/tasks, /api/dates/parse and /api/counts.
type TaskHandler struct {
	tasks *services.TaskService
}

// NewTaskHandler constructs a TaskHandler.
func NewTaskHandler(tasks *services.TaskService) *TaskHandler { return &TaskHandler{tasks: tasks} }

// HandleList lists tasks for a view/filter.
func (h *TaskHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	q := r.URL.Query()
	f := domain.TaskFilter{View: domain.TaskView(q.Get("view")), Query: q.Get("q")}
	if v := q.Get("project_id"); v != "" {
		f.ProjectID = &v
	}
	if v := q.Get("tag_id"); v != "" {
		f.TagID = &v
	}
	tasks, err := h.tasks.List(r.Context(), user.ID, f)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITasks(tasks, user.Location()))
}

// HandleCreate creates a task.
func (h *TaskHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	var req api.CreateTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	in := services.CreateTaskInput{
		Title:      req.Title,
		ProjectID:  uuidStr(req.ProjectId),
		DeferUntil: req.DeferUntil,
		DueAt:      req.DueAt,
		RepeatRule: fromAPIRepeat(req.RepeatRule),
		TagIDs:     uuidStrs(req.TagIds),
	}
	if req.Note != nil {
		in.Note = *req.Note
	}
	if req.Flagged != nil {
		in.Flagged = *req.Flagged
	}
	t, err := h.tasks.Create(r.Context(), user.ID, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAPITask(t, user.Location()))
}

// HandleGet returns one task.
func (h *TaskHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "taskId")
	if !ok {
		return
	}
	t, err := h.tasks.Get(r.Context(), user.ID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITask(t, user.Location()))
}

// updateTaskBody keeps nullable fields as RawMessage so an explicit
// null clears the field while absence leaves it alone.
type updateTaskBody struct {
	Title      *string         `json:"title"`
	Note       *string         `json:"note"`
	ProjectID  json.RawMessage `json:"project_id"`
	Flagged    *bool           `json:"flagged"`
	DeferUntil json.RawMessage `json:"defer_until"`
	DueAt      json.RawMessage `json:"due_at"`
	RepeatRule json.RawMessage `json:"repeat_rule"`
	TagIDs     *[]string       `json:"tag_ids"`
	SortOrder  *int            `json:"sort_order"`
}

func rawIsNull(raw json.RawMessage) bool {
	return strings.TrimSpace(string(raw)) == "null"
}

func rawTime(raw json.RawMessage) (*time.Time, error) {
	if rawIsNull(raw) {
		return nil, nil
	}
	var t time.Time
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// HandleUpdate patches a task.
func (h *TaskHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "taskId")
	if !ok {
		return
	}
	var body updateTaskBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	patch := services.TaskPatch{Title: body.Title, Note: body.Note, Flagged: body.Flagged, SortOrder: body.SortOrder}
	if body.TagIDs != nil {
		patch.TagIDs = *body.TagIDs
		if patch.TagIDs == nil {
			patch.TagIDs = []string{}
		}
	}
	if len(body.ProjectID) > 0 {
		patch.SetProject = true
		if !rawIsNull(body.ProjectID) {
			var s string
			if err := json.Unmarshal(body.ProjectID, &s); err != nil {
				writeJSONError(w, http.StatusBadRequest, "project_id must be a UUID or null")
				return
			}
			patch.ProjectID = &s
		}
	}
	var err error
	if len(body.DeferUntil) > 0 {
		patch.SetDefer = true
		if patch.DeferUntil, err = rawTime(body.DeferUntil); err != nil {
			writeJSONError(w, http.StatusBadRequest, "defer_until must be an RFC 3339 timestamp or null")
			return
		}
	}
	if len(body.DueAt) > 0 {
		patch.SetDue = true
		if patch.DueAt, err = rawTime(body.DueAt); err != nil {
			writeJSONError(w, http.StatusBadRequest, "due_at must be an RFC 3339 timestamp or null")
			return
		}
	}
	if len(body.RepeatRule) > 0 {
		patch.SetRepeat = true
		if !rawIsNull(body.RepeatRule) {
			var rr api.RepeatRule
			if err := json.Unmarshal(body.RepeatRule, &rr); err != nil {
				writeJSONError(w, http.StatusBadRequest, "repeat_rule must be an object or null")
				return
			}
			patch.RepeatRule = fromAPIRepeat(&rr)
		}
	}
	t, err := h.tasks.Update(r.Context(), user.ID, id, patch)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITask(t, user.Location()))
}

// HandleDelete deletes a task.
func (h *TaskHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "taskId")
	if !ok {
		return
	}
	if err := h.tasks.Delete(r.Context(), user.ID, id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.OkResponse{Ok: true})
}

// HandleComplete completes a task (spawning the next repeat).
func (h *TaskHandler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "taskId")
	if !ok {
		return
	}
	done, next, err := h.tasks.Complete(r.Context(), user, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := api.CompleteTaskResponse{Task: toAPITask(done, user.Location())}
	if next != nil {
		n := toAPITask(next, user.Location())
		resp.NextTask = &n
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleDrop drops a task.
func (h *TaskHandler) HandleDrop(w http.ResponseWriter, r *http.Request) {
	h.simpleAction(w, r, h.tasks.Drop)
}

// HandleReopen reopens a task.
func (h *TaskHandler) HandleReopen(w http.ResponseWriter, r *http.Request) {
	h.simpleAction(w, r, h.tasks.Reopen)
}

func (h *TaskHandler) simpleAction(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, userID, id string) (*domain.Task, error)) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "taskId")
	if !ok {
		return
	}
	t, err := fn(r.Context(), user.ID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITask(t, user.Location()))
}

// HandleReorder applies a drag-and-drop ordering.
func (h *TaskHandler) HandleReorder(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	var req api.ReorderTasksRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	ids := make([]string, 0, len(req.TaskIds))
	for _, id := range req.TaskIds {
		ids = append(ids, id.String())
	}
	tasks, err := h.tasks.Reorder(r.Context(), user.ID, ids)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITasks(tasks, user.Location()))
}

// HandleDefer sets the defer date from a phrase, timestamp, or clears it.
func (h *TaskHandler) HandleDefer(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "taskId")
	if !ok {
		return
	}
	var req api.DeferTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	in := services.DeferInput{Until: req.Until}
	if req.Input != nil {
		in.Input = *req.Input
	}
	if req.Clear != nil {
		in.Clear = *req.Clear
	}
	if req.Timezone != nil && *req.Timezone != "" {
		loc, err := time.LoadLocation(*req.Timezone)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Unknown timezone")
			return
		}
		in.Location = loc
	}
	t, _, err := h.tasks.Defer(r.Context(), user, id, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITask(t, user.Location()))
}

func locationFrom(w http.ResponseWriter, tz *string, fallback *time.Location) (*time.Location, bool) {
	if tz == nil || *tz == "" {
		return fallback, true
	}
	loc, err := time.LoadLocation(*tz)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Unknown timezone")
		return nil, false
	}
	return loc, true
}

// HandleSetRepeat sets, replaces or clears a task's repeat rule.
func (h *TaskHandler) HandleSetRepeat(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "taskId")
	if !ok {
		return
	}
	var req api.SetRepeatRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	loc, ok := locationFrom(w, req.Timezone, user.Location())
	if !ok {
		return
	}
	in := services.RepeatInput{Rule: fromAPIRepeat(req.Rule), Location: loc}
	if req.Input != nil {
		in.Input = *req.Input
	}
	if req.Clear != nil {
		in.Clear = *req.Clear
	}
	t, _, err := h.tasks.SetRepeat(r.Context(), user, id, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITask(t, user.Location()))
}

// HandleParseRepeat resolves a repeat phrase without changing anything.
func (h *TaskHandler) HandleParseRepeat(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	var req api.ParseRepeatRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	loc, ok := locationFrom(w, req.Timezone, user.Location())
	if !ok {
		return
	}
	preq := repeatparse.Request{Input: req.Input, Location: loc}
	if req.Reference != nil {
		preq.Now = *req.Reference
	}
	res, err := h.tasks.ParseRepeat(r.Context(), preq)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	base := time.Now().In(loc)
	if res.FirstOccurrence != nil {
		base = res.FirstOccurrence.Add(-time.Nanosecond)
	}
	out := api.ParseRepeatResponse{
		Rule:            *toAPIRepeat(&res.Rule),
		Description:     res.Description,
		FirstOccurrence: res.FirstOccurrence,
		NextOccurrences: res.Rule.Upcoming(base, upcomingCount),
		Source:          api.ParseRepeatResponseSource(res.Source),
	}
	writeJSON(w, http.StatusOK, out)
}

// HandleParseDate resolves a natural-language phrase.
func (h *TaskHandler) HandleParseDate(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	var req api.ParseDateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	loc := user.Location()
	if req.Timezone != nil && *req.Timezone != "" {
		l, err := time.LoadLocation(*req.Timezone)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Unknown timezone")
			return
		}
		loc = l
	}
	kind := dateparse.KindDefer
	if req.Kind != nil && *req.Kind == api.ParseDateRequestKindDue {
		kind = dateparse.KindDue
	}
	preq := dateparse.Request{Input: req.Input, Kind: kind, Location: loc}
	if req.Reference != nil {
		preq.Now = *req.Reference
	}
	res, err := h.tasks.ParseDate(r.Context(), preq)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.ParseDateResponse{
		ResolvedAt:     res.At,
		Interpretation: res.Interpretation,
		Source:         api.ParseDateResponseSource(res.Source),
	})
}

// HandleCounts returns sidebar badge counts.
func (h *TaskHandler) HandleCounts(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	c, err := h.tasks.Counts(r.Context(), user.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.Counts{Inbox: c.Inbox, Flagged: c.Flagged, DueSoon: c.DueSoon, Overdue: c.Overdue, ReviewDue: c.ReviewDue})
}
