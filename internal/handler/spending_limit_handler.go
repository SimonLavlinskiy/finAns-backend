package handler

import (
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

type SpendingLimitHandler struct {
	svc *service.SpendingLimitService
}

func NewSpendingLimitHandler(svc *service.SpendingLimitService) *SpendingLimitHandler {
	return &SpendingLimitHandler{svc: svc}
}

func (h *SpendingLimitHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	limits, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, limits)
}

func (h *SpendingLimitHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	var req dto.CreateSpendingLimitRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	l, err := h.svc.Create(r.Context(), req, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, l)
}

func (h *SpendingLimitHandler) Update(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	var req dto.UpdateSpendingLimitRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	l, err := h.svc.Update(r.Context(), id, req, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, l)
}

func (h *SpendingLimitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
