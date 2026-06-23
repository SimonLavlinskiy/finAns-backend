package domain

import "time"

const (
	PlannedExpensePriorityLow    = "low"
	PlannedExpensePriorityMedium = "medium"
	PlannedExpensePriorityHigh   = "high"

	PlannedExpenseStatusActive   = "active"
	PlannedExpenseStatusArchived = "archived"
)

type PlannedExpense struct {
	ID          int64
	CategoryID  int64
	Title       string
	CostKopecks *int64
	DueDate     *time.Time
	URL         *string
	Priority    string
	Status      string
	CreatedAt   time.Time
	ArchivedAt  *time.Time
}
