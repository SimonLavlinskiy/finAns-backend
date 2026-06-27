package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
)

var validPlannedExpensePriorities = map[string]bool{
	domain.PlannedExpensePriorityLow:    true,
	domain.PlannedExpensePriorityMedium: true,
	domain.PlannedExpensePriorityHigh:   true,
}

const defaultPlannedExpensePriority = domain.PlannedExpensePriorityMedium

type PlannedExpenseService struct {
	repo       *repository.PlannedExpenseRepository
	catRepo    *repository.PlannedExpenseCategoryRepository
	catService *PlannedExpenseCategoryService
}

func NewPlannedExpenseService(
	repo *repository.PlannedExpenseRepository,
	catRepo *repository.PlannedExpenseCategoryRepository,
	catService *PlannedExpenseCategoryService,
) *PlannedExpenseService {
	return &PlannedExpenseService{repo: repo, catRepo: catRepo, catService: catService}
}

func (s *PlannedExpenseService) Create(ctx context.Context, req dto.CreatePlannedExpenseRequest, projectID int64) (dto.PlannedExpenseResponse, error) {
	e, err := s.validateAndBuild(ctx, 0, req.Title, req.CostKopecks, req.DueDate, req.URL, req.Priority, req.CategoryID, req.NewCategory, projectID)
	if err != nil {
		return dto.PlannedExpenseResponse{}, err
	}
	created, err := s.repo.Create(ctx, e, projectID)
	if err != nil {
		return dto.PlannedExpenseResponse{}, err
	}
	return s.toResponse(created, time.Now()), nil
}

func (s *PlannedExpenseService) Update(ctx context.Context, id int64, req dto.UpdatePlannedExpenseRequest, projectID int64) (dto.PlannedExpenseResponse, error) {
	e, err := s.validateAndBuild(ctx, id, req.Title, req.CostKopecks, req.DueDate, req.URL, req.Priority, req.CategoryID, req.NewCategory, projectID)
	if err != nil {
		return dto.PlannedExpenseResponse{}, err
	}
	updated, err := s.repo.Update(ctx, e)
	if err != nil {
		return dto.PlannedExpenseResponse{}, err
	}
	return s.toResponse(updated, time.Now()), nil
}

func (s *PlannedExpenseService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *PlannedExpenseService) Complete(ctx context.Context, id int64) (dto.PlannedExpenseResponse, error) {
	archived, err := s.repo.Archive(ctx, id, time.Now())
	if err != nil {
		return dto.PlannedExpenseResponse{}, err
	}
	return s.toResponse(archived, time.Now()), nil
}

func (s *PlannedExpenseService) ListActive(ctx context.Context, projectID int64) ([]dto.PlannedExpenseCategoryResponse, error) {
	categories, err := s.catRepo.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	expenses, err := s.repo.ListByStatus(ctx, domain.PlannedExpenseStatusActive, projectID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	byCategory := make(map[int64][]domain.PlannedExpense)
	for _, e := range expenses {
		byCategory[e.CategoryID] = append(byCategory[e.CategoryID], e)
	}

	result := make([]dto.PlannedExpenseCategoryResponse, 0, len(categories))
	for _, cat := range categories {
		items := byCategory[cat.ID]
		sort.SliceStable(items, func(i, j int) bool {
			pi, _ := computeEffectivePriority(items[i].Priority, items[i].DueDate, now)
			pj, _ := computeEffectivePriority(items[j].Priority, items[j].DueDate, now)
			ri, rj := priorityRank(pi), priorityRank(pj)
			if ri != rj {
				return ri > rj
			}
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		})

		responses := make([]dto.PlannedExpenseResponse, 0, len(items))
		for _, e := range items {
			responses = append(responses, s.toResponse(e, now))
		}

		result = append(result, dto.PlannedExpenseCategoryResponse{
			ID:        cat.ID,
			Name:      cat.Name,
			Color:     cat.Color,
			SortOrder: cat.SortOrder,
			Items:     responses,
		})
	}
	return result, nil
}

func (s *PlannedExpenseService) ListArchived(ctx context.Context, projectID int64) ([]dto.PlannedExpenseResponse, error) {
	expenses, err := s.repo.ListByStatus(ctx, domain.PlannedExpenseStatusArchived, projectID)
	if err != nil {
		return nil, err
	}

	sort.SliceStable(expenses, func(i, j int) bool {
		ai, aj := expenses[i].ArchivedAt, expenses[j].ArchivedAt
		if ai == nil || aj == nil {
			return false
		}
		return ai.After(*aj)
	})

	now := time.Now()
	result := make([]dto.PlannedExpenseResponse, 0, len(expenses))
	for _, e := range expenses {
		result = append(result, s.toResponse(e, now))
	}
	return result, nil
}

func priorityRank(priority string) int {
	switch priority {
	case domain.PlannedExpensePriorityHigh:
		return 3
	case domain.PlannedExpensePriorityMedium:
		return 2
	default:
		return 1
	}
}

func (s *PlannedExpenseService) resolveCategoryID(ctx context.Context, categoryID *int64, newCategory *dto.CreatePlannedExpenseCategoryRequest, projectID int64) (int64, string) {
	if categoryID != nil {
		exists, err := s.catRepo.Exists(ctx, *categoryID)
		if err != nil || !exists {
			return 0, "категория не найдена"
		}
		return *categoryID, ""
	}
	if newCategory != nil {
		cat, err := s.catService.Create(ctx, *newCategory, projectID)
		if err != nil {
			return 0, "не удалось создать категорию"
		}
		return cat.ID, ""
	}
	return 0, "обязательное поле"
}

func (s *PlannedExpenseService) validateAndBuild(
	ctx context.Context,
	id int64,
	title string,
	costKopecks *int64,
	dueDateStr *string,
	url *string,
	priority string,
	categoryID *int64,
	newCategory *dto.CreatePlannedExpenseCategoryRequest,
	projectID int64,
) (domain.PlannedExpense, error) {
	fields := map[string]string{}

	if strings.TrimSpace(title) == "" {
		fields["title"] = "обязательное поле"
	}

	if priority == "" {
		priority = defaultPlannedExpensePriority
	} else if !validPlannedExpensePriorities[priority] {
		fields["priority"] = "недопустимое значение"
	}

	if costKopecks != nil && *costKopecks < 0 {
		fields["cost_kopecks"] = "должно быть >= 0"
	}

	var dueDate *time.Time
	if dueDateStr != nil && *dueDateStr != "" {
		parsed, err := time.Parse(dateLayout, *dueDateStr)
		if err != nil {
			fields["due_date"] = "не соответствует формату YYYY-MM-DD"
		} else {
			dueDate = &parsed
		}
	}

	resolvedCategoryID, categoryErr := s.resolveCategoryID(ctx, categoryID, newCategory, projectID)
	if categoryErr != "" {
		fields["category_id"] = categoryErr
	}

	if len(fields) > 0 {
		return domain.PlannedExpense{}, &apperrors.ValidationError{Message: "validation failed", Fields: fields}
	}

	return domain.PlannedExpense{
		ID:          id,
		CategoryID:  resolvedCategoryID,
		Title:       title,
		CostKopecks: costKopecks,
		DueDate:     dueDate,
		URL:         url,
		Priority:    priority,
	}, nil
}

func (s *PlannedExpenseService) toResponse(e domain.PlannedExpense, now time.Time) dto.PlannedExpenseResponse {
	effectivePriority, isDueSoon := computeEffectivePriority(e.Priority, e.DueDate, now)

	resp := dto.PlannedExpenseResponse{
		ID:                e.ID,
		CategoryID:        e.CategoryID,
		Title:             e.Title,
		CostKopecks:       e.CostKopecks,
		URL:               e.URL,
		Priority:          e.Priority,
		EffectivePriority: effectivePriority,
		IsDueSoon:         isDueSoon,
		Status:            e.Status,
		CreatedAt:         e.CreatedAt.Format(time.RFC3339),
	}
	if e.DueDate != nil {
		s := e.DueDate.Format(dateLayout)
		resp.DueDate = &s
	}
	if e.ArchivedAt != nil {
		s := e.ArchivedAt.Format(time.RFC3339)
		resp.ArchivedAt = &s
	}
	return resp
}
