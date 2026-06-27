package handler

import (
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

type ProjectHandler struct {
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	var req dto.CreateProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	project, err := h.svc.Create(r.Context(), user.ID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, project)
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	projects, err := h.svc.ListForUser(r.Context(), user.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, projects)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	project, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, project)
}

func (h *ProjectHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	members, err := h.svc.ListMembers(r.Context(), projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, members)
}

func (h *ProjectHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	var req dto.AddMemberRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if err := h.svc.AddMember(r.Context(), projectID, user.ID, req.Username); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	targetUserID, err := parseID(chi.URLParam(r, "userID"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid userID")
		return
	}
	if err := h.svc.RemoveMember(r.Context(), projectID, user.ID, targetUserID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
