package domain

import "time"

type PlannedExpenseCategory struct {
	ID        int64
	Name      string
	Color     string
	SortOrder int
	CreatedAt time.Time
}
