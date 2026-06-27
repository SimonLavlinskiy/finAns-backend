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
