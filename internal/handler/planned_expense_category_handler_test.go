package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPlannedExpenseCategorySvc struct {
	listResult   []domain.PlannedExpenseCategory
	listErr      error
	createResult domain.PlannedExpenseCategory
	createErr    error
	reorderErr   error
}

func (s *stubPlannedExpenseCategorySvc) List(_ context.Context) ([]domain.PlannedExpenseCategory, error) {
	return s.listResult, s.listErr
}

func (s *stubPlannedExpenseCategorySvc) Create(_ context.Context, _ dto.CreatePlannedExpenseCategoryRequest) (domain.PlannedExpenseCategory, error) {
	return s.createResult, s.createErr
}

func (s *stubPlannedExpenseCategorySvc) Reorder(_ context.Context, _ []int64) error {
	return s.reorderErr
}

// minimalPlannedExpenseCategoryHandler дублирует логику PlannedExpenseCategoryHandler через интерфейс.
type minimalPlannedExpenseCategoryHandler struct {
	stub *stubPlannedExpenseCategorySvc
}

func (h *minimalPlannedExpenseCategoryHandler) list(w http.ResponseWriter, r *http.Request) {
	categories, err := h.stub.List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": categories})
}

func (h *minimalPlannedExpenseCategoryHandler) create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePlannedExpenseCategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "invalid json"}})
		return
	}
	cat, err := h.stub.Create(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": cat})
}

func (h *minimalPlannedExpenseCategoryHandler) reorder(w http.ResponseWriter, r *http.Request) {
	var req dto.ReorderPlannedExpenseCategoriesRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "invalid json"}})
		return
	}
	if err := h.stub.Reorder(r.Context(), req.IDs); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func TestPlannedExpenseCategoryHandler_List_OK(t *testing.T) {
	stub := &stubPlannedExpenseCategorySvc{
		listResult: []domain.PlannedExpenseCategory{{ID: 1, Name: "Электроника", Color: "#112250"}},
	}
	h := &minimalPlannedExpenseCategoryHandler{stub: stub}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planned-expense-categories", nil)
	rec := httptest.NewRecorder()

	h.list(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string][]domain.PlannedExpenseCategory
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Len(t, resp["data"], 1)
	assert.Equal(t, "Электроника", resp["data"][0].Name)
}

func TestPlannedExpenseCategoryHandler_Create_ValidationError(t *testing.T) {
	stub := &stubPlannedExpenseCategorySvc{
		createErr: &apperrors.ValidationError{
			Message: "validation failed",
			Fields:  map[string]string{"color": "недопустимый цвет"},
		},
	}
	h := &minimalPlannedExpenseCategoryHandler{stub: stub}

	body := `{"name":"Электроника","color":"#ABCDEF"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/planned-expense-categories", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.create(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	errObj := resp["error"].(map[string]any)
	fields := errObj["fields"].(map[string]any)
	assert.Equal(t, "недопустимый цвет", fields["color"])
}

func TestPlannedExpenseCategoryHandler_Reorder_NoContent(t *testing.T) {
	stub := &stubPlannedExpenseCategorySvc{}
	h := &minimalPlannedExpenseCategoryHandler{stub: stub}

	body := `{"ids":[2,1,3]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/planned-expense-categories/reorder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.reorder(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestPlannedExpenseCategoryHandler_Reorder_ValidationError(t *testing.T) {
	stub := &stubPlannedExpenseCategorySvc{
		reorderErr: &apperrors.ValidationError{
			Message: "validation failed",
			Fields:  map[string]string{"ids": "должен содержать все категории без пропусков и дублей"},
		},
	}
	h := &minimalPlannedExpenseCategoryHandler{stub: stub}

	body := `{"ids":[1,1]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/planned-expense-categories/reorder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.reorder(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}
