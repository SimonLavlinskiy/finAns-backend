package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubMandatoryPaymentSvc реализует минимальный stub для тестов хендлера.
type stubMandatoryPaymentSvc struct {
	markPaidResult dto.MandatoryPaymentResponse
	markPaidErr    error
	createErr      error
}

func (s *stubMandatoryPaymentSvc) MarkPaid(_ context.Context, _ int64) (dto.MandatoryPaymentResponse, error) {
	return s.markPaidResult, s.markPaidErr
}

func (s *stubMandatoryPaymentSvc) Create(_ context.Context, _ dto.CreateMandatoryPaymentRequest) (dto.MandatoryPaymentResponse, error) {
	return dto.MandatoryPaymentResponse{}, s.createErr
}

// minimalMandatoryPaymentHandler использует только MarkPaid и Create через интерфейс.
type minimalMandatoryPaymentHandler struct {
	stub *stubMandatoryPaymentSvc
}

func (h *minimalMandatoryPaymentHandler) markPaid(w http.ResponseWriter, r *http.Request) {
	id := int64(1)
	p, err := h.stub.MarkPaid(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": p})
}

func (h *minimalMandatoryPaymentHandler) create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateMandatoryPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "invalid json"}})
		return
	}
	_, err := h.stub.Create(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func chiCtxWithID(id string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		next.ServeHTTP(w, r)
	})
}

func TestMarkPaid_OK(t *testing.T) {
	stub := &stubMandatoryPaymentSvc{
		markPaidResult: dto.MandatoryPaymentResponse{
			ID:              1,
			Title:           "Netflix",
			Amount:          89900,
			NextPaymentDate: "2026-02-10",
		},
	}
	h := &minimalMandatoryPaymentHandler{stub: stub}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mandatory-payments/1/mark-paid", nil)
	rec := httptest.NewRecorder()

	chiCtxWithID("1", http.HandlerFunc(h.markPaid)).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]dto.MandatoryPaymentResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "2026-02-10", resp["data"].NextPaymentDate)
}

func TestCreate_ValidationError(t *testing.T) {
	stub := &stubMandatoryPaymentSvc{
		createErr: &apperrors.ValidationError{
			Message: "validation failed",
			Fields:  map[string]string{"title": "обязательное поле"},
		},
	}
	h := &minimalMandatoryPaymentHandler{stub: stub}

	body := `{"amount":1000,"tag_id":1,"recurrence":"monthly","next_payment_date":"2026-06-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mandatory-payments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.create(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	errObj, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	fields, ok := errObj["fields"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "обязательное поле", fields["title"])
}
