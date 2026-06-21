package dto

type MandatoryPaymentResponse struct {
	ID              int64          `json:"id"`
	Title           string         `json:"title"`
	Amount          int64          `json:"amount"`
	Tag             TransactionTag `json:"tag"`
	Recurrence      string         `json:"recurrence"`
	NextPaymentDate string         `json:"next_payment_date"` // "YYYY-MM-DD"
	CreatedAt       string         `json:"created_at"`
	UpdatedAt       string         `json:"updated_at"`
}

type CreateMandatoryPaymentRequest struct {
	Title           string `json:"title"`
	Amount          int64  `json:"amount"`
	TagID           int64  `json:"tag_id"`
	Recurrence      string `json:"recurrence"`
	NextPaymentDate string `json:"next_payment_date"` // "YYYY-MM-DD"
}

type UpdateMandatoryPaymentRequest struct {
	Title           string `json:"title"`
	Amount          int64  `json:"amount"`
	TagID           int64  `json:"tag_id"`
	Recurrence      string `json:"recurrence"`
	NextPaymentDate string `json:"next_payment_date"`
}
