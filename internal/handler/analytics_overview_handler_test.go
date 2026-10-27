package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func newOverviewHandler() *AnalyticsOverviewHandler {
	clock := service.FixedClock{T: time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)}
	return NewAnalyticsOverviewHandler(service.NewAnalyticsOverviewService(nil, nil, clock))
}

func withProject(req *http.Request) *http.Request {
	return req.WithContext(middleware.WithProjectID(req.Context(), 1))
}

func TestAnalyticsOverviewHandler_Month_Validation(t *testing.T) {
	for _, q := range []string{"2026-11", "2026-13", "abc", ""} {
		req := withProject(httptest.NewRequest(http.MethodGet, "/api/v1/analytics/month?month="+q, nil))
		rec := httptest.NewRecorder()
		newOverviewHandler().GetMonth(rec, req)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, q)
	}
}

func TestAnalyticsOverviewHandler_RequiresProject(t *testing.T) {
	rec := httptest.NewRecorder()
	newOverviewHandler().GetMonth(rec, httptest.NewRequest(http.MethodGet, "/api/v1/analytics/month?month=2026-09", nil))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAnalyticsOverviewHandler_Year_Validation(t *testing.T) {
	cases := map[string]int{"abc": http.StatusBadRequest, "2027": http.StatusUnprocessableEntity}
	for q, code := range cases {
		req := withProject(httptest.NewRequest(http.MethodGet, "/api/v1/analytics/year?year="+q, nil))
		rec := httptest.NewRecorder()
		newOverviewHandler().GetYear(rec, req)
		require.Equal(t, code, rec.Code, q)
	}
}

func TestPlannerHandler_Put_Validation(t *testing.T) {
	h := NewPlannerHandler(service.NewPlannerService(nil))
	cases := map[string]int{
		`{not json`:                       http.StatusBadRequest,
		`{"unknown": 1}`:                  http.StatusBadRequest,
		`{"days": 40}`:                    http.StatusUnprocessableEntity,
		`{"plans": {"1": -5}}`:            http.StatusUnprocessableEntity,
		`{"payment_overrides": {"x": 1}}`: http.StatusUnprocessableEntity,
	}
	for body, code := range cases {
		req := withProject(httptest.NewRequest(http.MethodPut, "/api/v1/planner", strings.NewReader(body)))
		rec := httptest.NewRecorder()
		h.Put(rec, req)
		require.Equal(t, code, rec.Code, body)
	}
}

func TestSpendingLimitHandler_ByCategory_BadInput(t *testing.T) {
	h := NewSpendingLimitHandler(service.NewSpendingLimitService(nil, nil, nil))
	route := func(method, id, body string) int {
		req := withProject(httptest.NewRequest(method, "/api/v1/spending-limits/by-category/"+id, strings.NewReader(body)))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("categoryId", id)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()
		if method == http.MethodPut {
			h.UpsertByCategory(rec, req)
		} else {
			h.DeleteByCategory(rec, req)
		}
		return rec.Code
	}
	require.Equal(t, http.StatusBadRequest, route(http.MethodPut, "abc", `{"amount":100}`))
	require.Equal(t, http.StatusBadRequest, route(http.MethodPut, "1", `{bad`))
	require.Equal(t, http.StatusBadRequest, route(http.MethodDelete, "abc", ""))
}
