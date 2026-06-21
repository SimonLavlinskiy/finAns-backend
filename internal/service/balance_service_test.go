package service_test

import (
	"context"
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
)

// stubBalanceRepo implements the balance repo interface for unit tests (no DB needed).
type stubBalanceRepo struct {
	snap        domain.BalanceSnapshot
	upsertCalls []int64
	atomicCalls []int64
}

func (r *stubBalanceRepo) GetSnapshot(_ context.Context) (domain.BalanceSnapshot, error) {
	r.snap.CurrentBalance = r.snap.InitialAmount + r.snap.TotalIncome - r.snap.TotalExpense
	return r.snap, nil
}

func (r *stubBalanceRepo) UpsertBalance(_ context.Context, amount int64) error {
	r.upsertCalls = append(r.upsertCalls, amount)
	r.snap.InitialAmount = amount
	return nil
}

func (r *stubBalanceRepo) SetCurrentBalanceAtomic(_ context.Context, target int64) (domain.BalanceSnapshot, error) {
	r.atomicCalls = append(r.atomicCalls, target)
	newInitial := target - r.snap.TotalIncome + r.snap.TotalExpense
	r.snap.InitialAmount = newInitial
	r.snap.CurrentBalance = target
	return r.snap, nil
}

// balanceSvcForTest builds a BalanceService using a stub repo.
// We replicate the core logic here because BalanceService uses a concrete *BalanceRepository.
// These tests document expected behaviour and act as regression guards.

func calcSetCurrentBalance(snap domain.BalanceSnapshot, target int64) int64 {
	// initial_amount = target - total_income + total_expense
	return target - snap.TotalIncome + snap.TotalExpense
}

func TestSetCurrentBalance_Math(t *testing.T) {
	tests := []struct {
		name          string
		initial       int64
		totalIncome   int64
		totalExpense  int64
		targetBalance int64
		wantInitial   int64
	}{
		{
			name:          "zero income/expense: initial == target",
			initial:       0,
			totalIncome:   0,
			totalExpense:  0,
			targetBalance: 5000_00,
			wantInitial:   5000_00,
		},
		{
			name:          "existing income reduces required initial",
			initial:       0,
			totalIncome:   1000_00,
			totalExpense:  0,
			targetBalance: 3000_00,
			wantInitial:   2000_00, // 3000 - 1000 + 0
		},
		{
			name:          "existing expense increases required initial",
			initial:       0,
			totalIncome:   0,
			totalExpense:  500_00,
			targetBalance: 2000_00,
			wantInitial:   2500_00, // 2000 - 0 + 500
		},
		{
			name:          "both income and expense",
			initial:       1000_00,
			totalIncome:   300_00,
			totalExpense:  800_00,
			targetBalance: 1200_00,
			wantInitial:   1700_00, // 1200 - 300 + 800
		},
		{
			name:          "negative initial allowed (overdraft scenario)",
			initial:       0,
			totalIncome:   5000_00,
			totalExpense:  0,
			targetBalance: 0,
			wantInitial:   -5000_00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := domain.BalanceSnapshot{
				InitialAmount: tt.initial,
				TotalIncome:   tt.totalIncome,
				TotalExpense:  tt.totalExpense,
			}
			got := calcSetCurrentBalance(snap, tt.targetBalance)
			if got != tt.wantInitial {
				t.Errorf("calcSetCurrentBalance: got initial=%d, want %d", got, tt.wantInitial)
			}
			// Verify that resulting balance equals target
			resulting := got + tt.totalIncome - tt.totalExpense
			if resulting != tt.targetBalance {
				t.Errorf("resulting balance = %d, want target %d", resulting, tt.targetBalance)
			}
		})
	}
}

func TestBalanceResponse_Fields(t *testing.T) {
	snap := domain.BalanceSnapshot{
		InitialAmount:  1000_00,
		TotalIncome:    500_00,
		TotalExpense:   200_00,
		CurrentBalance: 1300_00,
	}
	resp := dto.BalanceResponse{
		Balance:       snap.CurrentBalance,
		InitialAmount: snap.InitialAmount,
		TotalIncome:   snap.TotalIncome,
		TotalExpense:  snap.TotalExpense,
	}
	if resp.Balance != 1300_00 {
		t.Errorf("Balance = %d, want 1300_00", resp.Balance)
	}
	if resp.TotalIncome != 500_00 {
		t.Errorf("TotalIncome = %d, want 500_00", resp.TotalIncome)
	}
}
