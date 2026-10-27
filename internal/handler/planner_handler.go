package handler

import (
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
)

type PlannerHandler struct {
	svc *service.PlannerService
}

func NewPlannerHandler(svc *service.PlannerService) *PlannerHandler {
	return &PlannerHandler{svc: svc}
}

// Get godoc
// @Summary      Планирование проекта (общее для участников)
// @Tags         planner
// @Produce      json
// @Success      200 {object} dto.PlannerResponse
// @Router       /api/v1/planner [get]
func (h *PlannerHandler) Get(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.Get(r.Context(), projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}

// Put godoc
// @Summary      Сохранить планирование проекта (последняя запись побеждает)
// @Tags         planner
// @Accept       json
// @Produce      json
// @Param        body body dto.PlannerRequest true "Состояние планирования"
// @Success      200 {object} dto.PlannerResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      422 {object} map[string]interface{}
// @Router       /api/v1/planner [put]
func (h *PlannerHandler) Put(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	var req dto.PlannerRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	var userID *int64
	if u, ok := middleware.UserFromContext(r.Context()); ok {
		id := u.ID
		userID = &id
	}
	resp, err := h.svc.Save(r.Context(), req, projectID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}
