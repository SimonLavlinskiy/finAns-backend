package service

import (
	"context"
	"sort"
	"strings"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/colorutil"
)

type PlannedExpenseCategoryService struct {
	repo *repository.PlannedExpenseCategoryRepository
}

func NewPlannedExpenseCategoryService(repo *repository.PlannedExpenseCategoryRepository) *PlannedExpenseCategoryService {
	return &PlannedExpenseCategoryService{repo: repo}
}

func (s *PlannedExpenseCategoryService) List(ctx context.Context, projectID int64) ([]domain.PlannedExpenseCategory, error) {
	return s.repo.List(ctx, projectID)
}

func (s *PlannedExpenseCategoryService) Create(ctx context.Context, req dto.CreatePlannedExpenseCategoryRequest, projectID int64) (domain.PlannedExpenseCategory, error) {
	return s.validateAndCreate(ctx, req.Name, req.Color, projectID)
}

func (s *PlannedExpenseCategoryService) validateAndCreate(ctx context.Context, name, color string, projectID int64) (domain.PlannedExpenseCategory, error) {
	fields := map[string]string{}
	if strings.TrimSpace(name) == "" {
		fields["name"] = "обязательное поле"
	}
	if !colorutil.IsValidCategoryColor(color) {
		fields["color"] = "недопустимый цвет"
	}
	if len(fields) > 0 {
		return domain.PlannedExpenseCategory{}, &apperrors.ValidationError{Message: "validation failed", Fields: fields}
	}
	return s.repo.Create(ctx, name, color, projectID)
}

func (s *PlannedExpenseCategoryService) Reorder(ctx context.Context, ids []int64, projectID int64) error {
	existing, err := s.repo.List(ctx, projectID)
	if err != nil {
		return err
	}

	if len(ids) != len(existing) {
		return &apperrors.ValidationError{Message: "validation failed", Fields: map[string]string{"ids": "должен содержать все категории без пропусков и дублей"}}
	}

	existingIDs := make([]int64, 0, len(existing))
	for _, c := range existing {
		existingIDs = append(existingIDs, c.ID)
	}
	sort.Slice(existingIDs, func(i, j int) bool { return existingIDs[i] < existingIDs[j] })

	gotIDs := make([]int64, len(ids))
	copy(gotIDs, ids)
	sort.Slice(gotIDs, func(i, j int) bool { return gotIDs[i] < gotIDs[j] })

	for i := range existingIDs {
		if existingIDs[i] != gotIDs[i] {
			return &apperrors.ValidationError{Message: "validation failed", Fields: map[string]string{"ids": "должен содержать все категории без пропусков и дублей"}}
		}
	}

	return s.repo.Reorder(ctx, ids)
}
