package handler

import (
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

type PlannedExpenseHandler struct {
	svc *service.PlannedExpenseService
}

func NewPlannedExpenseHandler(svc *service.PlannedExpenseService) *PlannedExpenseHandler {
	return &PlannedExpenseHandler{svc: svc}
}

// List godoc
// @Summary      List planned expenses (active grouped by category, or archived flat)
// @Tags         planned-expenses
// @Produce      json
// @Param        status query string false "active (default) or archived"
// @Success      200 {object} map[string]interface{}
// @Router       /api/v1/planned-expenses [get]
func (h *PlannedExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	if r.URL.Query().Get("status") == "archived" {
		items, err := h.svc.ListArchived(r.Context(), projectID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		httputil.WriteData(w, http.StatusOK, items)
		return
	}

	categories, err := h.svc.ListActive(r.Context(), projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, categories)
}

// Create godoc
// @Summary      Create a planned expense item
// @Tags         planned-expenses
// @Accept       json
// @Produce      json
// @Param        request body dto.CreatePlannedExpenseRequest true "Item"
// @Success      201 {object} map[string]interface{}
// @Router       /api/v1/planned-expenses [post]
func (h *PlannedExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	var req dto.CreatePlannedExpenseRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	e, err := h.svc.Create(r.Context(), req, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, e)
}

// Update godoc
// @Summary      Update a planned expense item
// @Tags         planned-expenses
// @Accept       json
// @Produce      json
// @Param        id path int true "Item ID"
// @Param        request body dto.UpdatePlannedExpenseRequest true "Item"
// @Success      200 {object} map[string]interface{}
// @Router       /api/v1/planned-expenses/{id} [patch]
func (h *PlannedExpenseHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	var req dto.UpdatePlannedExpenseRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	e, err := h.svc.Update(r.Context(), id, req, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, e)
}

// Delete godoc
// @Summary      Delete a planned expense item
// @Tags         planned-expenses
// @Param        id path int true "Item ID"
// @Success      204
// @Router       /api/v1/planned-expenses/{id} [delete]
func (h *PlannedExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

// Complete godoc
// @Summary      Mark a planned expense item as done and move it to the archive
// @Tags         planned-expenses
// @Produce      json
// @Param        id path int true "Item ID"
// @Success      200 {object} map[string]interface{}
// @Router       /api/v1/planned-expenses/{id}/complete [post]
func (h *PlannedExpenseHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	e, err := h.svc.Complete(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, e)
}
