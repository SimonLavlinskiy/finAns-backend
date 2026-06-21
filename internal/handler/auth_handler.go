package handler

import (
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
)

type AuthHandler struct {
	svc           *service.AuthService
	secureCookies bool
}

func NewAuthHandler(svc *service.AuthService, secureCookies bool) *AuthHandler {
	return &AuthHandler{svc: svc, secureCookies: secureCookies}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}

	token, err := h.svc.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(service.SessionTTL.Seconds()),
	})

	httputil.WriteData(w, http.StatusOK, dto.UserResponse{Login: req.Login})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	httputil.WriteData(w, http.StatusOK, dto.UserResponse{Login: claims.Login})
}
