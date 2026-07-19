package dto

type MandatoryPaymentResponse struct {
	ID                  int64          `json:"id"`
	Title               string         `json:"title"`
	Amount              int64          `json:"amount"`
	Tag                 TransactionTag `json:"tag"`
	Recurrence          string         `json:"recurrence"`
	NextPaymentDate     string         `json:"next_payment_date"` // "YYYY-MM-DD"
	LastPaidAt          *string        `json:"last_paid_at"`      // RFC3339 or null
	IsPaidCurrentPeriod bool           `json:"is_paid_current_period"`
	CreatedAt           string         `json:"created_at"`
	UpdatedAt           string         `json:"updated_at"`
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

// MarkPaidRequest — необязательное тело запроса при оплате регулярного платежа.
// Amount позволяет отредактировать сумму создаваемой транзакции перед подтверждением;
// если не передан (nil) или <= 0, используется сумма самого регулярного платежа.
type MarkPaidRequest struct {
	Amount *int64 `json:"amount"`
}
