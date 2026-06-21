package handler

import (
	"net/http"
	"strconv"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

type MandatoryPaymentHandler struct {
	svc *service.MandatoryPaymentService
}

func NewMandatoryPaymentHandler(svc *service.MandatoryPaymentService) *MandatoryPaymentHandler {
	return &MandatoryPaymentHandler{svc: svc}
}

func (h *MandatoryPaymentHandler) List(w http.ResponseWriter, r *http.Request) {
	payments, err := h.svc.List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, payments)
}

func (h *MandatoryPaymentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	p, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, p)
}

func (h *MandatoryPaymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateMandatoryPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	p, err := h.svc.Create(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, p)
}

func (h *MandatoryPaymentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	var req dto.UpdateMandatoryPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	p, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, p)
}

func (h *MandatoryPaymentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
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

func (h *MandatoryPaymentHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	p, err := h.svc.Duplicate(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, p)
}

func (h *MandatoryPaymentHandler) MarkPaid(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	p, err := h.svc.MarkPaid(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, p)
}
