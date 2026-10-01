package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/thanosd/focus/backend/internal/api"
	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/services"
)

// ProjectHandler serves /api/projects and /api/reviews.
type ProjectHandler struct {
	projects *services.ProjectService
}

// NewProjectHandler constructs a ProjectHandler.
func NewProjectHandler(projects *services.ProjectService) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

// HandleList lists projects.
func (h *ProjectHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	projects, err := h.projects.List(r.Context(), user.ID, r.URL.Query().Get("status"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIProjects(projects))
}

// HandleCreate creates a project.
func (h *ProjectHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	var req api.CreateProjectRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	in := services.CreateProjectInput{Name: req.Name, ParentID: uuidStr(req.ParentId)}
	if req.Note != nil {
		in.Note = *req.Note
	}
	if req.Sequential != nil {
		in.Sequential = *req.Sequential
	}
	if req.ReviewIntervalDays != nil {
		in.ReviewIntervalDays = *req.ReviewIntervalDays
	}
	p, err := h.projects.Create(r.Context(), user.ID, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAPIProject(p))
}

// HandleGet returns a project with tasks and children.
func (h *ProjectHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "projectId")
	if !ok {
		return
	}
	d, err := h.projects.GetDetail(r.Context(), user.ID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.ProjectDetail{
		Project:  toAPIProject(d.Project),
		Tasks:    toAPITasks(d.Tasks),
		Children: toAPIProjects(d.Children),
	})
}

// updateProjectBody mirrors api.UpdateProjectRequest but keeps parent_id
// as RawMessage so "null" (move to top level) and "absent" differ.
type updateProjectBody struct {
	Name               *string            `json:"name"`
	Note               *string            `json:"note"`
	ParentID           json.RawMessage    `json:"parent_id"`
	Status             *api.ProjectStatus `json:"status"`
	Sequential         *bool              `json:"sequential"`
	ReviewIntervalDays *int               `json:"review_interval_days"`
	SortOrder          *int               `json:"sort_order"`
}

// HandleUpdate patches a project.
func (h *ProjectHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "projectId")
	if !ok {
		return
	}
	var body updateProjectBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	patch := services.ProjectPatch{
		Name:               body.Name,
		Note:               body.Note,
		Sequential:         body.Sequential,
		ReviewIntervalDays: body.ReviewIntervalDays,
		SortOrder:          body.SortOrder,
	}
	if body.Status != nil {
		s := domain.ProjectStatus(*body.Status)
		patch.Status = &s
	}
	if len(body.ParentID) > 0 {
		patch.SetParent = true
		if string(body.ParentID) != "null" {
			var s string
			if err := json.Unmarshal(body.ParentID, &s); err != nil {
				writeJSONError(w, http.StatusBadRequest, "parent_id must be a UUID or null")
				return
			}
			patch.ParentID = &s
		}
	}
	p, err := h.projects.Update(r.Context(), user.ID, id, patch)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIProject(p))
}

// HandleDelete deletes a project.
func (h *ProjectHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "projectId")
	if !ok {
		return
	}
	if err := h.projects.Delete(r.Context(), user.ID, id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.OkResponse{Ok: true})
}

// HandleReview marks a project reviewed.
func (h *ProjectHandler) HandleReview(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "projectId")
	if !ok {
		return
	}
	p, err := h.projects.MarkReviewed(r.Context(), user.ID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIProject(p))
}

// HandleReviews lists due and upcoming reviews.
func (h *ProjectHandler) HandleReviews(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	due, upcoming, err := h.projects.Reviews(r.Context(), user.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.ReviewResponse{Due: toAPIProjects(due), Upcoming: toAPIProjects(upcoming)})
}
