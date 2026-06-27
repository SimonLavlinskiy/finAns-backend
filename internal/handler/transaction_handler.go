package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

type TransactionHandler struct {
	svc *service.TransactionService
}

func NewTransactionHandler(svc *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	f := parseTransactionFilters(r)
	items, result, err := h.svc.List(r.Context(), f, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	totalPages := int(result.Total) / f.PerPage
	if int(result.Total)%f.PerPage != 0 {
		totalPages++
	}
	httputil.WriteList(w, http.StatusOK, items, httputil.PaginationMeta{
		Page: f.Page, PerPage: f.PerPage, Total: result.Total, TotalPages: totalPages,
	})
}

func (h *TransactionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	item, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, item)
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	var req dto.CreateTransactionRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	item, err := h.svc.Create(r.Context(), req, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, item)
}

func (h *TransactionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	var req dto.UpdateTransactionRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	item, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, item)
}

func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h *TransactionHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
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
	item, err := h.svc.Duplicate(r.Context(), id, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, item)
}

func (h *TransactionHandler) Suggestions(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	q := r.URL.Query().Get("q")
	items, err := h.svc.Suggestions(r.Context(), q, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, items)
}

func parseTransactionFilters(r *http.Request) domain.TransactionFilters {
	q := r.URL.Query()
	f := domain.TransactionFilters{
		Search:      q.Get("search"),
		Category:    q.Get("category"),
		Specificity: q.Get("specificity"),
		SortBy:      q.Get("sort_by"),
		SortOrder:   q.Get("sort_order"),
	}
	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			f.Page = p
		}
	}
	if v := q.Get("per_page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			f.PerPage = p
		}
	}
	if v := q.Get("amount_min"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.AmountMin = &n
		}
	}
	if v := q.Get("amount_max"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.AmountMax = &n
		}
	}
	if v := q.Get("date_from"); v != "" {
		if d, err := time.Parse("2006-01-02", v); err == nil {
			f.DateFrom = &d
		}
	}
	if v := q.Get("date_to"); v != "" {
		if d, err := time.Parse("2006-01-02", v); err == nil {
			f.DateTo = &d
		}
	}
	if v := q.Get("tag_ids"); v != "" {
		for _, part := range strings.Split(v, ",") {
			if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
				f.TagIDs = append(f.TagIDs, id)
			}
		}
	}
	return f
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func writeServiceError(w http.ResponseWriter, err error) {
	var ve *apperrors.ValidationError
	var nf *apperrors.NotFoundError
	var ue *apperrors.UnauthorizedError
	var fe *apperrors.ForbiddenError
	var ce *apperrors.ConflictError
	switch {
	case errors.As(err, &ve):
		httputil.WriteValidationError(w, ve.Message, ve.Fields)
	case errors.As(err, &nf):
		httputil.WriteError(w, http.StatusNotFound, "NOT_FOUND", nf.Error())
	case errors.As(err, &ue):
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", ue.Error())
	case errors.As(err, &fe):
		httputil.WriteError(w, http.StatusForbidden, "FORBIDDEN", fe.Error())
	case errors.As(err, &ce):
		httputil.WriteError(w, http.StatusConflict, ce.Code, ce.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
