package service

import (
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestPeriodBounds_Custom(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	l := domain.SpendingLimit{PeriodType: domain.LimitPeriodCustom, PeriodStart: &start, PeriodEnd: &end}

	from, to := periodBounds(l)

	assert.Equal(t, start, from)
	assert.Equal(t, time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC), to, "to должен быть period_end+1 день (полуоткрытый интервал, period_end включительно)")
}

func TestPeriodBounds_Month(t *testing.T) {
	l := domain.SpendingLimit{PeriodType: domain.LimitPeriodMonth}
	from, to := periodBounds(l)

	now := time.Now().UTC()
	assert.Equal(t, 1, from.Day())
	assert.Equal(t, now.Month(), from.Month())
	assert.Equal(t, from.AddDate(0, 1, 0), to)
}

func TestPeriodBounds_Week(t *testing.T) {
	l := domain.SpendingLimit{PeriodType: domain.LimitPeriodWeek}
	from, to := periodBounds(l)

	assert.Equal(t, time.Monday, from.Weekday())
	assert.Equal(t, from.AddDate(0, 0, 7), to)
	assert.True(t, !from.After(time.Now().UTC()), "начало недели не должно быть в будущем")
}
