package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
)

type AnalyticsHandler struct {
	svc *service.AnalyticsService
}

func NewAnalyticsHandler(svc *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

// GetExpensesCalendar godoc
// @Summary      Expenses calendar aggregation
// @Tags         analytics
// @Produce      json
// @Param        level query string true "day or month"
// @Param        year query int true "year"
// @Param        month query int false "month (1-12), required for level=day"
// @Success      200 {object} map[string]interface{}
// @Router       /api/v1/analytics/expenses-calendar [get]
func (h *AnalyticsHandler) GetExpensesCalendar(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}

	q := r.URL.Query()
	level := q.Get("level")

	year, err := strconv.Atoi(q.Get("year"))
	if err != nil {
		year = time.Now().UTC().Year()
	}

	month := 0
	if v := q.Get("month"); v != "" {
		if m, err := strconv.Atoi(v); err == nil {
			month = m
		}
	}

	result, err := h.svc.GetExpensesCalendar(r.Context(), level, year, month, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteData(w, http.StatusOK, dto.CalendarResultToResponse(result))
}

func projectIDOrError(w http.ResponseWriter, r *http.Request) (int64, bool) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return 0, false
	}
	return projectID, true
}

func (h *AnalyticsHandler) GetFreedomIndex(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	months := 12
	if v := r.URL.Query().Get("months"); v != "" {
		if m, err := strconv.Atoi(v); err == nil {
			months = m
		}
	}
	resp, err := h.svc.GetFreedomIndex(r.Context(), months, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}

func (h *AnalyticsHandler) GetFixedCostRatio(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	now := time.Now().UTC()
	year, err := strconv.Atoi(q.Get("year"))
	if err != nil {
		year = now.Year()
	}
	month, err := strconv.Atoi(q.Get("month"))
	if err != nil {
		month = int(now.Month())
	}
	resp, err := h.svc.GetFixedCostRatio(r.Context(), year, month, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}

func (h *AnalyticsHandler) GetIncomeExpenseSankey(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	level := q.Get("level")
	now := time.Now().UTC()
	year, err := strconv.Atoi(q.Get("year"))
	if err != nil {
		year = now.Year()
	}
	month := 0
	if v := q.Get("month"); v != "" {
		if m, err := strconv.Atoi(v); err == nil {
			month = m
		}
	} else {
		month = int(now.Month())
	}
	result, err := h.svc.GetIncomeExpenseSankey(r.Context(), level, year, month, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, dto.SankeyResultToResponse(result))
}

func (h *AnalyticsHandler) GetBalanceForecast(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			days = d
		}
	}
	resp, err := h.svc.GetBalanceForecast(r.Context(), days, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}

func (h *AnalyticsHandler) GetSavingsGoals(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.GetSavingsGoals(r.Context(), projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}

func (h *AnalyticsHandler) GetSpendingMoodCalendar(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	now := time.Now().UTC()
	year, err := strconv.Atoi(q.Get("year"))
	if err != nil {
		year = now.Year()
	}
	month, err := strconv.Atoi(q.Get("month"))
	if err != nil {
		month = int(now.Month())
	}
	resp, err := h.svc.GetSpendingMoodCalendar(r.Context(), year, month, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}

func (h *AnalyticsHandler) GetInsights(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.GetInsights(r.Context(), projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}

func (h *AnalyticsHandler) GetPeriodComparison(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	mode := r.URL.Query().Get("mode")
	resp, err := h.svc.GetPeriodComparison(r.Context(), mode, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}
