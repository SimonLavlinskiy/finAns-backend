package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTransactionFilters_Defaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	f := parseTransactionFilters(req)

	assert.Equal(t, "", f.Search)
	assert.Equal(t, "", f.Category)
	assert.Equal(t, "", f.Specificity)
	assert.Nil(t, f.AmountMin)
	assert.Nil(t, f.AmountMax)
	assert.Nil(t, f.DateFrom)
	assert.Nil(t, f.DateTo)
	assert.Empty(t, f.TagIDs)
}

func TestParseTransactionFilters_WithParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/transactions?search=coffee&category=expense&specificity=simple&page=2&per_page=20&amount_min=500&amount_max=9999&date_from=2026-01-01&date_to=2026-12-31&tag_ids=1,2,3&sort_by=date&sort_order=asc",
		nil,
	)
	f := parseTransactionFilters(req)

	assert.Equal(t, "coffee", f.Search)
	assert.Equal(t, "expense", f.Category)
	assert.Equal(t, "simple", f.Specificity)
	assert.Equal(t, 2, f.Page)
	assert.Equal(t, 20, f.PerPage)
	require.NotNil(t, f.AmountMin)
	assert.Equal(t, int64(500), *f.AmountMin)
	require.NotNil(t, f.AmountMax)
	assert.Equal(t, int64(9999), *f.AmountMax)
	require.NotNil(t, f.DateFrom)
	assert.Equal(t, "2026-01-01", f.DateFrom.Format("2006-01-02"))
	require.NotNil(t, f.DateTo)
	assert.Equal(t, "2026-12-31", f.DateTo.Format("2006-01-02"))
	assert.Equal(t, []int64{1, 2, 3}, f.TagIDs)
	assert.Equal(t, "date", f.SortBy)
	assert.Equal(t, "asc", f.SortOrder)
}

func TestParseTransactionFilters_InvalidNumerics(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/transactions?page=bad&per_page=x&amount_min=abc&amount_max=xyz&tag_ids=1,,bad,3",
		nil,
	)
	f := parseTransactionFilters(req)

	// invalid page/per_page — defaults to zero
	assert.Equal(t, 0, f.Page)
	assert.Equal(t, 0, f.PerPage)
	assert.Nil(t, f.AmountMin)
	assert.Nil(t, f.AmountMax)
	// Only valid tag IDs are included
	assert.Equal(t, []int64{1, 3}, f.TagIDs)
}

func TestParseID_Valid(t *testing.T) {
	id, err := parseID("42")
	require.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestParseID_Invalid(t *testing.T) {
	_, err := parseID("not-a-number")
	assert.Error(t, err)
}

func TestWriteServiceError_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	err := &apperrors.ValidationError{Message: "bad input", Fields: map[string]string{"title": "required"}}
	writeServiceError(w, err)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestWriteServiceError_NotFoundError(t *testing.T) {
	w := httptest.NewRecorder()
	err := &apperrors.NotFoundError{Resource: "transaction"}
	writeServiceError(w, err)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWriteServiceError_UnauthorizedError(t *testing.T) {
	w := httptest.NewRecorder()
	err := &apperrors.UnauthorizedError{Message: "no token"}
	writeServiceError(w, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWriteServiceError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	writeServiceError(w, assert.AnError)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
