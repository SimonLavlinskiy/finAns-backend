package domain

import "time"

const (
	LimitPeriodMonth  = "month"
	LimitPeriodWeek   = "week"
	LimitPeriodCustom = "custom"
)

type SpendingLimit struct {
	ID          int64
	TagID       int64
	Amount      int64
	PeriodType  string
	PeriodStart *time.Time
	PeriodEnd   *time.Time
	ProjectID   int64
	CreatedAt   time.Time
}
