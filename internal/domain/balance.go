package domain

import "context"

type BalanceSnapshot struct {
	InitialAmount  int64
	TotalIncome    int64
	TotalExpense   int64
	CurrentBalance int64
}

type BalanceRepository interface {
	GetSnapshot(ctx context.Context, projectID int64) (BalanceSnapshot, error)
	UpsertInitialAmount(ctx context.Context, amount int64, projectID int64) error
	SetCurrentBalanceAtomic(ctx context.Context, target int64, projectID int64) (BalanceSnapshot, error)
}
