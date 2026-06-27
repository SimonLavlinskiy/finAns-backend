package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAnalyticsHandler_GetExpensesCalendar_InvalidLevel(t *testing.T) {
	h := NewAnalyticsHandler(service.NewAnalyticsService(nil, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/expenses-calendar?level=week&year=2026", nil)
	req = req.WithContext(middleware.WithProjectID(req.Context(), 1))
	rec := httptest.NewRecorder()

	h.GetExpensesCalendar(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestAnalyticsHandler_GetExpensesCalendar_DayWithoutMonth(t *testing.T) {
	h := NewAnalyticsHandler(service.NewAnalyticsService(nil, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/expenses-calendar?level=day&year=2026", nil)
	req = req.WithContext(middleware.WithProjectID(req.Context(), 1))
	rec := httptest.NewRecorder()

	h.GetExpensesCalendar(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}
