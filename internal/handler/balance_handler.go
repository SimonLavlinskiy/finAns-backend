package handler

import (
	"encoding/json"
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
)

type BalanceHandler struct {
	svc *service.BalanceService
}

func NewBalanceHandler(svc *service.BalanceService) *BalanceHandler {
	return &BalanceHandler{svc: svc}
}

func (h *BalanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	bal, err := h.svc.Get(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, bal)
}

func (h *BalanceHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req dto.UpdateBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	bal, err := h.svc.UpdateFromRequest(r.Context(), req)
	if err != nil {
		if err.Error() == "balance or initial_amount required" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, bal)
}
