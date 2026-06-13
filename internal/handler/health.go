package handler

import (
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
)

type HealthHandler struct {
	svc *service.HealthService
}

func NewHealthHandler(svc *service.HealthService) *HealthHandler {
	return &HealthHandler{svc: svc}
}

// HealthCheck returns API and database status.
//
//	@Summary		Health check
//	@Description	Returns service and database connectivity status
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	dto.HealthResponse
//	@Router			/api/v1/health [get]
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	resp := h.svc.Check(r.Context())
	httputil.WriteJSON(w, http.StatusOK, resp)
}

// NotFound handles unknown routes.
func NotFound(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteError(w, http.StatusNotFound, "not_found", "resource not found")
}

// MethodNotAllowed handles unsupported methods.
func MethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}
