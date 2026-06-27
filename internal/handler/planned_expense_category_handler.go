package handler

import (
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
)

type PlannedExpenseCategoryHandler struct {
	svc *service.PlannedExpenseCategoryService
}

func NewPlannedExpenseCategoryHandler(svc *service.PlannedExpenseCategoryService) *PlannedExpenseCategoryHandler {
	return &PlannedExpenseCategoryHandler{svc: svc}
}

// List godoc
// @Summary      List planned expense categories
// @Tags         planned-expense-categories
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/v1/planned-expense-categories [get]
func (h *PlannedExpenseCategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	categories, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, categories)
}

// Create godoc
// @Summary      Create a planned expense category
// @Tags         planned-expense-categories
// @Accept       json
// @Produce      json
// @Param        request body dto.CreatePlannedExpenseCategoryRequest true "Category"
// @Success      201 {object} map[string]interface{}
// @Router       /api/v1/planned-expense-categories [post]
func (h *PlannedExpenseCategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	var req dto.CreatePlannedExpenseCategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	cat, err := h.svc.Create(r.Context(), req, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, cat)
}

// Reorder godoc
// @Summary      Persist a new drag-and-drop order for category cards
// @Tags         planned-expense-categories
// @Accept       json
// @Param        request body dto.ReorderPlannedExpenseCategoriesRequest true "Full ordered list of category IDs"
// @Success      204
// @Router       /api/v1/planned-expense-categories/reorder [patch]
func (h *PlannedExpenseCategoryHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	var req dto.ReorderPlannedExpenseCategoriesRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if err := h.svc.Reorder(r.Context(), req.IDs, projectID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
