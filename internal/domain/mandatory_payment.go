package domain

import "time"

type MandatoryPayment struct {
	ID                int64
	Title             string
	Amount            int64
	TagID             int64
	Recurrence        string // daily|weekly|monthly|quarterly|semi_annual|yearly
	NextPaymentDate   time.Time
	LastPaidAt        *time.Time
	LastTransactionID *int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
