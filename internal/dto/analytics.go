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
