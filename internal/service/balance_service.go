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

func (s *BalanceService) Get(ctx context.Context, projectID int64) (dto.BalanceResponse, error) {
	snap, err := s.repo.GetSnapshot(ctx, projectID)
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

func (s *BalanceService) UpdateFromRequest(ctx context.Context, req dto.UpdateBalanceRequest, projectID int64) (dto.BalanceResponse, error) {
	switch {
	case req.Balance != nil:
		return s.setCurrentBalance(ctx, *req.Balance, projectID)
	case req.InitialAmount != nil:
		if err := s.repo.UpsertInitialAmount(ctx, *req.InitialAmount, projectID); err != nil {
			return dto.BalanceResponse{}, err
		}
		return s.Get(ctx, projectID)
	default:
		return dto.BalanceResponse{}, fmt.Errorf("balance or initial_amount required")
	}
}

func (s *BalanceService) setCurrentBalance(ctx context.Context, target int64, projectID int64) (dto.BalanceResponse, error) {
	snap, err := s.repo.SetCurrentBalanceAtomic(ctx, target, projectID)
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
