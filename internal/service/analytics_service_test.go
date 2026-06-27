package service

import (
	"context"
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExpensesCalendar_InvalidLevel(t *testing.T) {
	svc := NewAnalyticsService(nil, nil)
	_, err := svc.GetExpensesCalendar(context.Background(), "week", 2026, 6, int64(1))

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, ve.Fields, "level")
}

func TestGetExpensesCalendar_DayWithoutMonth(t *testing.T) {
	svc := NewAnalyticsService(nil, nil)
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
