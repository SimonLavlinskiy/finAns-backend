package dto

type SpendingLimitResponse struct {
	ID          int64          `json:"id"`
	TagID       int64          `json:"tag_id"`
	Tag         TransactionTag `json:"tag"`
	Amount      int64          `json:"amount"`
	Spent       int64          `json:"spent"`
	PeriodType  string         `json:"period_type"`
	PeriodStart *string        `json:"period_start"`
	PeriodEnd   *string        `json:"period_end"`
}

type CreateSpendingLimitRequest struct {
	TagID       int64   `json:"tag_id"`
	Amount      int64   `json:"amount"`
	PeriodType  string  `json:"period_type"`
	PeriodStart *string `json:"period_start"`
	PeriodEnd   *string `json:"period_end"`
}

type UpdateSpendingLimitRequest struct {
	TagID       int64   `json:"tag_id"`
	Amount      int64   `json:"amount"`
	PeriodType  string  `json:"period_type"`
	PeriodStart *string `json:"period_start"`
	PeriodEnd   *string `json:"period_end"`
}
