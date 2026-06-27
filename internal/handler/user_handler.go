package handler

import (
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, users)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	user, err := h.svc.Create(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, user)
}
