package domain

import "time"

// PlannerIncome — строка «Планируемый доход»; Amount nil — поле суммы пустое.
type PlannerIncome struct {
	Name   string `json:"name"`
	Amount *int64 `json:"amount"`
}

// PlannerData — состояние вкладки «Планирование» (хранится JSONB, одно на проект).
type PlannerData struct {
	// Days — дней в планируемом месяце; nil — авто (дни следующего месяца).
	Days    *int            `json:"days"`
	Incomes []PlannerIncome `json:"incomes"`
	// PaymentOverrides — id регулярного платежа → своя сумма в копейках.
	PaymentOverrides map[string]int64 `json:"payment_overrides"`
	// Plans — id категории → «План» в копейках.
	Plans map[string]int64 `json:"plans"`
}

type Planner struct {
	ProjectID int64
	Data      PlannerData
	UpdatedBy *int64
	UpdatedAt *time.Time
}
