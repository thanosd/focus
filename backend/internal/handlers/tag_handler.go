package handlers

import (
	"net/http"

	"github.com/thanosd/focus/backend/internal/api"
	"github.com/thanosd/focus/backend/internal/services"
)

// TagHandler serves /api/tags.
type TagHandler struct {
	tags *services.TagService
}

// NewTagHandler constructs a TagHandler.
func NewTagHandler(tags *services.TagService) *TagHandler { return &TagHandler{tags: tags} }

// HandleList lists tags.
func (h *TagHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	tags, err := h.tags.List(r.Context(), user.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITags(tags))
}

// HandleCreate creates a tag.
func (h *TagHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	var req api.CreateTagRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	color := ""
	if req.Color != nil {
		color = *req.Color
	}
	tag, err := h.tags.Create(r.Context(), user.ID, req.Name, color)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAPITag(*tag))
}

// HandleUpdate renames/recolors a tag.
func (h *TagHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "tagId")
	if !ok {
		return
	}
	var req api.UpdateTagRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	tag, err := h.tags.Update(r.Context(), user.ID, id, services.TagPatch{Name: req.Name, Color: req.Color})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITag(*tag))
}

// HandleDelete deletes a tag.
func (h *TagHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "tagId")
	if !ok {
		return
	}
	if err := h.tags.Delete(r.Context(), user.ID, id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.OkResponse{Ok: true})
}
