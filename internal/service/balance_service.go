package service

import (
	"context"
	"fmt"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
)

type BalanceService struct {
	repo *repository.BalanceRepository
}

func NewBalanceService(repo *repository.BalanceRepository) *BalanceService {
	return &BalanceService{repo: repo}
}

func (s *BalanceService) Get(ctx context.Context) (dto.BalanceResponse, error) {
	snap, err := s.repo.GetSnapshot(ctx)
	if err != nil {
		return dto.BalanceResponse{}, err
	}
	return dto.BalanceResponse{
		Balance:       snap.CurrentBalance,
		InitialAmount: snap.InitialAmount,
		TotalIncome:   snap.TotalIncome,
		TotalExpense:  snap.TotalExpense,
	}, nil
}

// UpdateFromRequest задаёт текущий баланс (balance) или базовую сумму (initial_amount).
func (s *BalanceService) UpdateFromRequest(ctx context.Context, req dto.UpdateBalanceRequest) (dto.BalanceResponse, error) {
	switch {
	case req.Balance != nil:
		// Атомарная операция: всё в одной DB-транзакции
		return s.setCurrentBalance(ctx, *req.Balance)
	case req.InitialAmount != nil:
		if err := s.repo.UpsertBalance(ctx, *req.InitialAmount); err != nil {
			return dto.BalanceResponse{}, err
		}
		return s.Get(ctx)
	default:
		return dto.BalanceResponse{}, fmt.Errorf("balance or initial_amount required")
	}
}

// setCurrentBalance атомарно перечисляет initial_amount так, чтобы текущий баланс стал target.
func (s *BalanceService) setCurrentBalance(ctx context.Context, target int64) (dto.BalanceResponse, error) {
	snap, err := s.repo.SetCurrentBalanceAtomic(ctx, target)
	if err != nil {
		return dto.BalanceResponse{}, err
	}
	return dto.BalanceResponse{
		Balance:       snap.CurrentBalance,
		InitialAmount: snap.InitialAmount,
		TotalIncome:   snap.TotalIncome,
		TotalExpense:  snap.TotalExpense,
	}, nil
}
