package service

import (
	"context"
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExpensesCalendar_InvalidLevel(t *testing.T) {
	svc := NewAnalyticsService(nil, nil, nil, nil, nil)
	_, err := svc.GetExpensesCalendar(context.Background(), "week", 2026, 6, int64(1))

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, ve.Fields, "level")
}

func TestGetExpensesCalendar_DayWithoutMonth(t *testing.T) {
	svc := NewAnalyticsService(nil, nil, nil, nil, nil)
	_, err := svc.GetExpensesCalendar(context.Background(), "day", 2026, 0, int64(1))

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, ve.Fields, "month")
}

func TestBuildTagBreakdown_RollupToRootTag(t *testing.T) {
	// "Еда" (root) -> "Кафе" (subtag); transaction tagged "Кафе" must roll up into "Еда".
	rootOf := map[int64]tagInfo{
		1: {rootID: 1, name: "Еда", color: "#FF0000"},
		2: {rootID: 1, name: "Еда", color: "#FF0000"}, // "Кафе" rolls up to root id 1
		3: {rootID: 3, name: "Транспорт", color: "#00FF00"},
	}
	txs := []domain.TransactionBrief{
		{ID: 101, Title: "Обед", Amount: 50000, TagID: 1},
		{ID: 102, Title: "Кофе", Amount: 30000, TagID: 2},
		{ID: 103, Title: "Такси", Amount: 20000, TagID: 3},
	}

	breakdown := buildTagBreakdown(txs, rootOf, 100000)

	require.Len(t, breakdown, 2)

	byRoot := make(map[int64]domain.CalendarTagBreakdown)
	for _, b := range breakdown {
		byRoot[b.TagID] = b
	}

	require.Contains(t, byRoot, int64(1))
	assert.Equal(t, "Еда", byRoot[1].Name)
	assert.Equal(t, int64(80000), byRoot[1].Amount) // 50000 (id=1) + 30000 (id=2, subtag) rolled up
	assert.InDelta(t, 80.0, byRoot[1].Percent, 0.001)

	require.Contains(t, byRoot, int64(3))
	assert.Equal(t, int64(20000), byRoot[3].Amount)
	assert.InDelta(t, 20.0, byRoot[3].Percent, 0.001)

	var totalPercent float64
	for _, b := range breakdown {
		totalPercent += b.Percent
	}
	assert.InDelta(t, 100.0, totalPercent, 0.001)
}

func TestBuildTagBreakdown_ZeroDayTotal(t *testing.T) {
	breakdown := buildTagBreakdown(nil, map[int64]tagInfo{}, 0)
	assert.Empty(t, breakdown)
}

func TestBuildFreedomIndexResponse_FillsGapMonths(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := []domain.FreedomIndexPoint{
		{Month: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), SimpleAmount: 30000, TotalExpense: 100000},
		// февраль отсутствует — не было расходов
		{Month: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), SimpleAmount: 0, TotalExpense: 50000},
	}

	resp := buildFreedomIndexResponse(points, since, 3)

	require.Len(t, resp, 3)
	require.NotNil(t, resp[0].Share)
	assert.InDelta(t, 0.3, *resp[0].Share, 0.0001)

	assert.Nil(t, resp[1].Share, "февраль без расходов должен быть nil, а не 0")
	assert.Equal(t, int64(0), resp[1].TotalExpense)

	require.NotNil(t, resp[2].Share)
	assert.InDelta(t, 0.0, *resp[2].Share, 0.0001)
}

func TestBuildSankeyResult_IncomeToExpenseToSubtag(t *testing.T) {
	// Тег 1 "Еда" (root), тег 2 "Кафе" (подтег 1), тег 10 "Зарплата" (root доход)
	rootOf := map[int64]tagInfo{
		1:  {rootID: 1, name: "Еда"},
		2:  {rootID: 1, name: "Еда"},
		10: {rootID: 10, name: "Зарплата"},
	}
	nameByID := map[int64]string{1: "Еда", 2: "Кафе", 10: "Зарплата"}

	incomeSums := []domain.TagAmount{{TagID: 10, Amount: 100000}}
	expenseSums := []domain.TagAmount{
		{TagID: 1, Amount: 25000}, // напрямую на "Еда"
		{TagID: 2, Amount: 5000},  // на подтег "Кафе"
	}

	result := buildSankeyResult(incomeSums, expenseSums, rootOf, nameByID)

	flowsByPair := map[string]int64{}
	for _, f := range result.Flows {
		flowsByPair[f.From+"->"+f.To] = f.Amount
	}

	assert.Equal(t, int64(100000), flowsByPair["income:10->income_hub"])
	assert.Equal(t, int64(30000), flowsByPair["income_hub->expense:1"], "поток в узел Еда = 25000 прямых + 5000 подтега")
	assert.Equal(t, int64(5000), flowsByPair["expense:1->expense:2"])
	assert.Equal(t, int64(70000), flowsByPair["income_hub->remainder"], "остаток = 100000 - 30000")
	assert.Nil(t, result.DeficitAmount)
}

func TestBuildSankeyResult_Deficit(t *testing.T) {
	rootOf := map[int64]tagInfo{1: {rootID: 1, name: "Еда"}}
	nameByID := map[int64]string{1: "Еда", 10: "Зарплата"}

	incomeSums := []domain.TagAmount{{TagID: 10, Amount: 10000}}
	expenseSums := []domain.TagAmount{{TagID: 1, Amount: 50000}}

	result := buildSankeyResult(incomeSums, expenseSums, rootOf, nameByID)

	require.NotNil(t, result.DeficitAmount)
	assert.Equal(t, int64(40000), *result.DeficitAmount)

	for _, n := range result.Nodes {
		assert.NotEqual(t, "remainder", n.ID, "узел Остаток не должен появляться при дефиците")
	}
}

func TestMonthOrYearRange_Day(t *testing.T) {
	from, to, err := monthOrYearRange("day", 2026, 7)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), from)
	assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), to)
}

func TestMonthOrYearRange_Year(t *testing.T) {
	from, to, err := monthOrYearRange("month", 2026, 0)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), from)
	assert.Equal(t, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), to)
}

func TestMonthOrYearRange_InvalidLevel(t *testing.T) {
	_, _, err := monthOrYearRange("week", 2026, 1)
	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, ve.Fields, "level")
}

func TestMonthOrYearRange_DayWithoutMonth(t *testing.T) {
	_, _, err := monthOrYearRange("day", 2026, 0)
	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, ve.Fields, "month")
}

func TestProjectMandatoryOccurrences_MultipleOccurrencesInHorizon(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	payments := []domain.MandatoryPayment{
		{Amount: 5000, Recurrence: "weekly", NextPaymentDate: time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)},
	}

	occurrences := projectMandatoryOccurrences(payments, from, 30)

	// Горизонт [from, from+30] включительно: еженедельный платёж с 3 июля
	// наступает 5 раз — 3, 10, 17, 24, 31 июля (31-е попадает точно на границу).
	total := int64(0)
	for _, amt := range occurrences {
		total += amt
	}
	assert.Equal(t, 5, len(occurrences))
	assert.Equal(t, int64(25000), total)
}

func TestProjectMandatoryOccurrences_OutsideHorizonExcluded(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	payments := []domain.MandatoryPayment{
		{Amount: 5000, Recurrence: "monthly", NextPaymentDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
	}

	occurrences := projectMandatoryOccurrences(payments, from, 30)
	assert.Empty(t, occurrences)
}

func TestProjectSavingsGoal_AlreadyAffordable(t *testing.T) {
	cost := int64(50000)
	item := domain.PlannedExpense{ID: 1, Title: "Наушники", CostKopecks: &cost}
	g := projectSavingsGoal(item, 100000, 1000, time.Now())
	assert.True(t, g.AlreadyAffordable)
	assert.Nil(t, g.ProjectedDate)
}

func TestProjectSavingsGoal_PositiveRateProjectsDate(t *testing.T) {
	cost := int64(100000)
	item := domain.PlannedExpense{ID: 1, Title: "Ноутбук", CostKopecks: &cost}
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	g := projectSavingsGoal(item, 0, 1000, now) // 100 дней при ставке 1000/день

	require.NotNil(t, g.ProjectedDate)
	assert.Equal(t, "2026-10-09", *g.ProjectedDate)
	assert.False(t, g.AlreadyAffordable)
}

func TestProjectSavingsGoal_ZeroOrNegativeRateIsUnknown(t *testing.T) {
	cost := int64(100000)
	item := domain.PlannedExpense{ID: 1, Title: "Ноутбук", CostKopecks: &cost}
	g := projectSavingsGoal(item, 0, 0, time.Now())
	assert.Nil(t, g.ProjectedDate)
	assert.False(t, g.AlreadyAffordable)

	g2 := projectSavingsGoal(item, 0, -500, time.Now())
	assert.Nil(t, g2.ProjectedDate)
}

func TestTopNByMax_SortsAndTruncates(t *testing.T) {
	amounts := []domain.PeriodComparisonAmounts{
		{TagID: 1, CurrentAmount: 1000, ComparisonAmount: 500},
		{TagID: 2, CurrentAmount: 100, ComparisonAmount: 9000},
		{TagID: 3, CurrentAmount: 300, ComparisonAmount: 200},
	}
	top := topNByMax(amounts, 2)
	require.Len(t, top, 2)
	assert.Equal(t, int64(2), top[0].TagID) // max=9000
	assert.Equal(t, int64(1), top[1].TagID) // max=1000
}

func TestBiggestSignificantGrowth_ThresholdsBothRequired(t *testing.T) {
	deltas := []domain.TagDelta{
		{TagID: 1, CurrentAmount: 3000_00, AverageAmount: 2500_00}, // +20%, +500₽ — не проходит (дельта < 1000₽)
		{TagID: 2, CurrentAmount: 5000_00, AverageAmount: 3000_00}, // +66%, +2000₽ — проходит
	}
	best, pct := biggestSignificantGrowth(deltas)
	require.NotNil(t, best)
	assert.Equal(t, int64(2), best.TagID)
	assert.InDelta(t, 0.6667, pct, 0.001)
}

func TestBiggestSignificantGrowth_NoneQualifies(t *testing.T) {
	deltas := []domain.TagDelta{
		{TagID: 1, CurrentAmount: 3000_00, AverageAmount: 2500_00}, // +20%, +500₽ — дельта мала
	}
	best, _ := biggestSignificantGrowth(deltas)
	assert.Nil(t, best)
}
