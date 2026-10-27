package dto

import "github.com/SimonLavlinskiy/finAns-backend/internal/domain"

type PlannerRequest struct {
	Days             *int                   `json:"days"`
	Incomes          []domain.PlannerIncome `json:"incomes"`
	PaymentOverrides map[string]int64       `json:"payment_overrides"`
	Plans            map[string]int64       `json:"plans"`
}

type PlannerResponse struct {
	Days             *int                   `json:"days"`
	Incomes          []domain.PlannerIncome `json:"incomes"`
	PaymentOverrides map[string]int64       `json:"payment_overrides"`
	Plans            map[string]int64       `json:"plans"`
	UpdatedAt        *string                `json:"updated_at"`
}
