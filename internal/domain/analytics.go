package domain

import (
	"context"
	"time"
)

type CalendarLevel string

const (
	CalendarLevelDay   CalendarLevel = "day"
	CalendarLevelMonth CalendarLevel = "month"
)

type CalendarTagBreakdown struct {
	TagID   int64
	Name    string
	Color   string
	Amount  int64
	Percent float64
}

type CalendarTransactionBrief struct {
	ID     int64
	Title  string
	Amount int64
}

type CalendarItem struct {
	Key            string
	Amount         int64
	HasData        bool
	IsCurrent      bool
	BreakdownByTag []CalendarTagBreakdown
	Transactions   []CalendarTransactionBrief
}

type CalendarResult struct {
	Level       CalendarLevel
	Year        int
	Month       int
	Total       int64
	HasPrevious bool
	Items       []CalendarItem
}

// DailySum — сумма расходов за один день месяца.
type DailySum struct {
	Day    int
	Amount int64
}

// MonthlySum — сумма расходов за один месяц года.
type MonthlySum struct {
	Month  int
	Amount int64
}

// TransactionBrief — минимальные данные транзакции для разбивки календаря по дням.
type TransactionBrief struct {
	ID     int64
	Title  string
	Amount int64
	Day    int
	TagID  int64
}

type AnalyticsRepository interface {
	SumByDayInMonth(ctx context.Context, year, month int) ([]DailySum, error)
	SumByMonthInYear(ctx context.Context, year int) ([]MonthlySum, error)
	ListExpenseTransactionsInMonth(ctx context.Context, year, month int) ([]TransactionBrief, error)
	ExistsExpenseBefore(ctx context.Context, before time.Time) (bool, error)
}
