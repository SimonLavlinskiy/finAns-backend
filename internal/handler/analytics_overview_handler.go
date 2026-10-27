package handler

import (
	"net/http"
	"strconv"

	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
)

// AnalyticsOverviewHandler — эндпоинты нового раздела «Аналитика».
type AnalyticsOverviewHandler struct {
	svc *service.AnalyticsOverviewService
}

func NewAnalyticsOverviewHandler(svc *service.AnalyticsOverviewService) *AnalyticsOverviewHandler {
	return &AnalyticsOverviewHandler{svc: svc}
}

// GetMonth godoc
// @Summary      Сводка месяца: доходы, расходы, категории и подкатегории
// @Tags         analytics
// @Produce      json
// @Param        month query string true "месяц YYYY-MM (не в будущем)"
// @Success      200 {object} dto.MonthSummaryResponse
// @Failure      422 {object} map[string]interface{}
// @Router       /api/v1/analytics/month [get]
func (h *AnalyticsOverviewHandler) GetMonth(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.MonthSummary(r.Context(), r.URL.Query().Get("month"), projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}

// GetYear godoc
// @Summary      Доходы и расходы по месяцам года (до текущего месяца)
// @Tags         analytics
// @Produce      json
// @Param        year query int true "год"
// @Success      200 {object} dto.YearSummaryResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      422 {object} map[string]interface{}
// @Router       /api/v1/analytics/year [get]
func (h *AnalyticsOverviewHandler) GetYear(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid year")
		return
	}
	resp, err := h.svc.YearSummary(r.Context(), year, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}

// GetRange godoc
// @Summary      Диапазон данных: первый месяц с операциями, текущий месяц, годы с данными
// @Tags         analytics
// @Produce      json
// @Success      200 {object} dto.AnalyticsRangeResponse
// @Router       /api/v1/analytics/range [get]
func (h *AnalyticsOverviewHandler) GetRange(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDOrError(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.Range(r.Context(), projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, resp)
}
