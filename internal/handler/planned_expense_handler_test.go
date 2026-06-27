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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPlannedExpenseSvc struct {
	listActiveResult   []dto.PlannedExpenseCategoryResponse
	listActiveErr      error
	listArchivedResult []dto.PlannedExpenseResponse
	listArchivedErr    error
	createResult       dto.PlannedExpenseResponse
	createErr          error
	updateResult       dto.PlannedExpenseResponse
	updateErr          error
	deleteErr          error
	completeResult     dto.PlannedExpenseResponse
	completeErr        error
}

func (s *stubPlannedExpenseSvc) ListActive(_ context.Context) ([]dto.PlannedExpenseCategoryResponse, error) {
	return s.listActiveResult, s.listActiveErr
}

func (s *stubPlannedExpenseSvc) ListArchived(_ context.Context) ([]dto.PlannedExpenseResponse, error) {
	return s.listArchivedResult, s.listArchivedErr
}

func (s *stubPlannedExpenseSvc) Create(_ context.Context, _ dto.CreatePlannedExpenseRequest) (dto.PlannedExpenseResponse, error) {
	return s.createResult, s.createErr
}

func (s *stubPlannedExpenseSvc) Update(_ context.Context, _ int64, _ dto.UpdatePlannedExpenseRequest) (dto.PlannedExpenseResponse, error) {
	return s.updateResult, s.updateErr
}

func (s *stubPlannedExpenseSvc) Delete(_ context.Context, _ int64) error {
	return s.deleteErr
}

func (s *stubPlannedExpenseSvc) Complete(_ context.Context, _ int64) (dto.PlannedExpenseResponse, error) {
	return s.completeResult, s.completeErr
}

// minimalPlannedExpenseHandler дублирует логику PlannedExpenseHandler через интерфейс.
type minimalPlannedExpenseHandler struct {
	stub *stubPlannedExpenseSvc
}

func (h *minimalPlannedExpenseHandler) list(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("status") == "archived" {
		items, err := h.stub.ListArchived(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": items})
		return
	}
	categories, err := h.stub.ListActive(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": categories})
}

func (h *minimalPlannedExpenseHandler) create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePlannedExpenseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "invalid json"}})
		return
	}
	e, err := h.stub.Create(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": e})
}

func (h *minimalPlannedExpenseHandler) complete(w http.ResponseWriter, r *http.Request) {
	id := int64(1)
	e, err := h.stub.Complete(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": e})
}

func (h *minimalPlannedExpenseHandler) deleteOne(w http.ResponseWriter, r *http.Request) {
	id := int64(1)
	if err := h.stub.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func TestPlannedExpenseHandler_List_Active(t *testing.T) {
	stub := &stubPlannedExpenseSvc{
		listActiveResult: []dto.PlannedExpenseCategoryResponse{
			{ID: 1, Name: "Электроника", Items: []dto.PlannedExpenseResponse{{ID: 10, Title: "Наушники"}}},
		},
	}
	h := &minimalPlannedExpenseHandler{stub: stub}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planned-expenses", nil)
	rec := httptest.NewRecorder()

	h.list(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string][]dto.PlannedExpenseCategoryResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp["data"], 1)
	assert.Equal(t, "Наушники", resp["data"][0].Items[0].Title)
}

func TestPlannedExpenseHandler_List_Archived(t *testing.T) {
	stub := &stubPlannedExpenseSvc{
		listArchivedResult: []dto.PlannedExpenseResponse{{ID: 5, Title: "Пылесос", Status: "archived"}},
	}
	h := &minimalPlannedExpenseHandler{stub: stub}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planned-expenses?status=archived", nil)
	rec := httptest.NewRecorder()

	h.list(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string][]dto.PlannedExpenseResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp["data"], 1)
	assert.Equal(t, "Пылесос", resp["data"][0].Title)
}

func TestPlannedExpenseHandler_Create_ValidationError(t *testing.T) {
	stub := &stubPlannedExpenseSvc{
		createErr: &apperrors.ValidationError{
			Message: "validation failed",
			Fields:  map[string]string{"title": "обязательное поле"},
		},
	}
	h := &minimalPlannedExpenseHandler{stub: stub}

	body := `{"priority":"medium","category_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/planned-expenses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.create(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	errObj := resp["error"].(map[string]any)
	fields := errObj["fields"].(map[string]any)
	assert.Equal(t, "обязательное поле", fields["title"])
}

func TestPlannedExpenseHandler_Complete_NotFound(t *testing.T) {
	stub := &stubPlannedExpenseSvc{
		completeErr: &apperrors.NotFoundError{Resource: "planned_expense"},
	}
	h := &minimalPlannedExpenseHandler{stub: stub}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/planned-expenses/1/complete", nil)
	rec := httptest.NewRecorder()

	h.complete(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPlannedExpenseHandler_Delete_NoContent(t *testing.T) {
	stub := &stubPlannedExpenseSvc{}
	h := &minimalPlannedExpenseHandler{stub: stub}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/planned-expenses/1", nil)
	rec := httptest.NewRecorder()

	h.deleteOne(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
}
