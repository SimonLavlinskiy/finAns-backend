package dto

import "github.com/SimonLavlinskiy/finAns-backend/internal/domain"

type TagBreakdownResponse struct {
	TagID   int64   `json:"tag_id"`
	Name    string  `json:"name"`
	Color   string  `json:"color"`
	Amount  int64   `json:"amount"`
	Percent float64 `json:"percent"`
}

type TransactionBriefResponse struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Amount int64  `json:"amount"`
}

type CalendarItemResponse struct {
	Key            string                     `json:"key"`
	Amount         int64                      `json:"amount"`
	HasData        bool                       `json:"has_data"`
	IsCurrent      bool                       `json:"is_current"`
	BreakdownByTag []TagBreakdownResponse     `json:"breakdown_by_tag"`
	Transactions   []TransactionBriefResponse `json:"transactions"`
}

type CalendarPeriod struct {
	Year  int  `json:"year"`
	Month *int `json:"month,omitempty"`
}

type CalendarResponse struct {
	Level       string                 `json:"level"`
	Period      CalendarPeriod         `json:"period"`
	Total       int64                  `json:"total"`
	HasPrevious bool                   `json:"has_previous"`
	Items       []CalendarItemResponse `json:"items"`
}

func CalendarResultToResponse(r domain.CalendarResult) CalendarResponse {
	period := CalendarPeriod{Year: r.Year}
	if r.Level == domain.CalendarLevelDay {
		month := r.Month
		period.Month = &month
	}

	items := make([]CalendarItemResponse, 0, len(r.Items))
	for _, it := range r.Items {
		breakdown := make([]TagBreakdownResponse, 0, len(it.BreakdownByTag))
		for _, b := range it.BreakdownByTag {
			breakdown = append(breakdown, TagBreakdownResponse{
				TagID:   b.TagID,
				Name:    b.Name,
				Color:   b.Color,
				Amount:  b.Amount,
				Percent: b.Percent,
			})
		}
		transactions := make([]TransactionBriefResponse, 0, len(it.Transactions))
		for _, t := range it.Transactions {
			transactions = append(transactions, TransactionBriefResponse{
				ID:     t.ID,
				Title:  t.Title,
				Amount: t.Amount,
			})
		}
		items = append(items, CalendarItemResponse{
			Key:            it.Key,
			Amount:         it.Amount,
			HasData:        it.HasData,
			IsCurrent:      it.IsCurrent,
			BreakdownByTag: breakdown,
			Transactions:   transactions,
		})
	}

	return CalendarResponse{
		Level:       string(r.Level),
		Period:      period,
		Total:       r.Total,
		HasPrevious: r.HasPrevious,
		Items:       items,
	}
}

type FreedomIndexPointResponse struct {
	Month        string   `json:"month"`
	SimpleAmount int64    `json:"simple_amount"`
	TotalExpense int64    `json:"total_expense"`
	Share        *float64 `json:"share"`
}

type FixedCostRatioResponse struct {
	Income         int64    `json:"income"`
	MandatoryTotal int64    `json:"mandatory_total"`
	Ratio          *float64 `json:"ratio"`
}

type SankeyNodeResponse struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type SankeyFlowResponse struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int64  `json:"amount"`
}

type SankeyResponse struct {
	Nodes         []SankeyNodeResponse `json:"nodes"`
	Flows         []SankeyFlowResponse `json:"flows"`
	DeficitAmount *int64               `json:"deficit_amount"`
}

func SankeyResultToResponse(r domain.SankeyResult) SankeyResponse {
	nodes := make([]SankeyNodeResponse, 0, len(r.Nodes))
	for _, n := range r.Nodes {
		nodes = append(nodes, SankeyNodeResponse{ID: n.ID, Label: n.Label})
	}
	flows := make([]SankeyFlowResponse, 0, len(r.Flows))
	for _, f := range r.Flows {
		flows = append(flows, SankeyFlowResponse{From: f.From, To: f.To, Amount: f.Amount})
	}
	return SankeyResponse{Nodes: nodes, Flows: flows, DeficitAmount: r.DeficitAmount}
}

type BalancePointResponse struct {
	Date    string `json:"date"`
	Balance int64  `json:"balance"`
}

type BalanceForecastResponse struct {
	CurrentBalance   int64                  `json:"current_balance"`
	History          []BalancePointResponse `json:"history"`
	Forecast         []BalancePointResponse `json:"forecast"`
	ZeroCrossingDate *string                `json:"zero_crossing_date"`
}

type SavingsGoalResponse struct {
	PlannedExpenseID  int64   `json:"planned_expense_id"`
	Title             string  `json:"title"`
	Cost              int64   `json:"cost"`
	ProjectedDate     *string `json:"projected_date"`
	AlreadyAffordable bool    `json:"already_affordable"`
}

type MoodDayResponse struct {
	Day          int      `json:"day"`
	SimpleAmount int64    `json:"simple_amount"`
	TotalExpense int64    `json:"total_expense"`
	Share        *float64 `json:"share"`
}

type InsightResponse struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type PeriodComparisonResponse struct {
	TagID            int64  `json:"tag_id"`
	Name             string `json:"name"`
	CurrentAmount    int64  `json:"current_amount"`
	ComparisonAmount int64  `json:"comparison_amount"`
}
