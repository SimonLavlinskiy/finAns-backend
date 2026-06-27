package handler

import (
	"encoding/json"
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

type TagHandler struct {
	svc *service.TagService
}

func NewTagHandler(svc *service.TagService) *TagHandler {
	return &TagHandler{svc: svc}
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	tree, err := h.svc.ListTree(r.Context(), projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, tree)
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	var req dto.CreateTagRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	tag, err := h.svc.Create(r.Context(), req, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, tag)
}

func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	var req dto.UpdateTagRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	tag, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, tag)
}

func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	cascade := r.URL.Query().Get("cascade") == "true"
	if err := h.svc.Delete(r.Context(), id, cascade); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TagHandler) Usage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	usage, err := h.svc.GetUsage(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, usage)
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
