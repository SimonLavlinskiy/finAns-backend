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

// FreedomIndexPoint — сумма «простых» и общих расходов за один месяц.
type FreedomIndexPoint struct {
	Month        time.Time
	SimpleAmount int64
	TotalExpense int64
}

// TagAmount — сумма транзакций по одному тегу за период.
type TagAmount struct {
	TagID  int64
	Amount int64
}

// DailyMoodPoint — сумма «простых» и общих расходов за один день месяца.
type DailyMoodPoint struct {
	Day          int
	SimpleAmount int64
	TotalExpense int64
}

type SankeyNode struct {
	ID    string
	Label string
}

type SankeyFlow struct {
	From   string
	To     string
	Amount int64
}

type SankeyResult struct {
	Nodes []SankeyNode
	Flows []SankeyFlow
	// DeficitAmount — заполняется, если расход за период превысил доход
	// (в этом случае узел «Остаток» не строится).
	DeficitAmount *int64
}

// BalanceForecastPoint — баланс (фактический или прогнозируемый) на дату.
type BalanceForecastPoint struct {
	Date    time.Time
	Balance int64
}

// Insight — один сработавший авто-инсайт с готовым текстом.
type Insight struct {
	Type string
	Text string
}

// TagDelta — сумма по тегу за текущий месяц и среднее за предыдущие месяцы.
type TagDelta struct {
	TagID         int64
	CurrentAmount int64
	AverageAmount float64
}

// PeriodComparisonAmounts — суммы по тегу за два сравниваемых периода.
type PeriodComparisonAmounts struct {
	TagID            int64
	CurrentAmount    int64
	ComparisonAmount int64
}

// TagCategoryAmount — сумма операций одного типа по тегу за период.
type TagCategoryAmount struct {
	TagID        int64
	Category     string // income | expense
	Amount       int64
	AllMandatory bool
}

// MonthIncomeExpense — доходы и расходы за месяц года.
type MonthIncomeExpense struct {
	Month   int
	Income  int64
	Expense int64
}
