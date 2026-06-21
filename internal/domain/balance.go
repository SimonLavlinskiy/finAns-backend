package domain

import "context"

type BalanceSnapshot struct {
	InitialAmount  int64
	TotalIncome    int64
	TotalExpense   int64
	CurrentBalance int64
}

type BalanceRepository interface {
	GetSnapshot(ctx context.Context) (BalanceSnapshot, error)
	UpsertInitialAmount(ctx context.Context, amount int64) error
}
