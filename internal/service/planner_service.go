package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
)

const (
	plannerMaxIncomes  = 50
	plannerMaxNameLen  = 255
	plannerMaxMapItems = 500
)

// PlannerService — общее для проекта планирование (последняя запись побеждает).
type PlannerService struct {
	repo *repository.PlannerRepository
}

func NewPlannerService(repo *repository.PlannerRepository) *PlannerService {
	return &PlannerService{repo: repo}
}

func (s *PlannerService) Get(ctx context.Context, projectID int64) (dto.PlannerResponse, error) {
	p, found, err := s.repo.Get(ctx, projectID)
	if err != nil {
		return dto.PlannerResponse{}, err
	}
	if !found {
		return plannerToResponse(domain.Planner{Data: DefaultPlannerData()}), nil
	}
	return plannerToResponse(p), nil
}

func (s *PlannerService) Save(ctx context.Context, req dto.PlannerRequest, projectID int64, userID *int64) (dto.PlannerResponse, error) {
	data, err := ValidatePlanner(req)
	if err != nil {
		return dto.PlannerResponse{}, err
	}
	p, err := s.repo.Upsert(ctx, projectID, data, userID)
	if err != nil {
		return dto.PlannerResponse{}, err
	}
	return plannerToResponse(p), nil
}

// DefaultPlannerData — значения по умолчанию: одна пустая строка дохода, дни — авто.
func DefaultPlannerData() domain.PlannerData {
	return domain.PlannerData{
		Incomes:          []domain.PlannerIncome{{Name: "", Amount: nil}},
		PaymentOverrides: map[string]int64{},
		Plans:            map[string]int64{},
	}
}

// ValidatePlanner проверяет и нормализует данные планирования.
func ValidatePlanner(req dto.PlannerRequest) (domain.PlannerData, error) {
	fields := map[string]string{}

	if req.Days != nil && (*req.Days < 1 || *req.Days > 31) {
		fields["days"] = "от 1 до 31 или null"
	}
	if len(req.Incomes) > plannerMaxIncomes {
		fields["incomes"] = fmt.Sprintf("не больше %d строк", plannerMaxIncomes)
	}
	incomes := make([]domain.PlannerIncome, 0, len(req.Incomes))
	for i, in := range req.Incomes {
		name := strings.TrimSpace(in.Name)
		if utf8.RuneCountInString(name) > plannerMaxNameLen {
			fields[fmt.Sprintf("incomes[%d].name", i)] = fmt.Sprintf("не длиннее %d символов", plannerMaxNameLen)
		}
		if in.Amount != nil && *in.Amount < 0 {
			fields[fmt.Sprintf("incomes[%d].amount", i)] = "не может быть отрицательной"
		}
		incomes = append(incomes, domain.PlannerIncome{Name: name, Amount: in.Amount})
	}
	if len(incomes) == 0 {
		incomes = DefaultPlannerData().Incomes
	}

	overrides, ok := validateAmountMap(req.PaymentOverrides)
	if !ok {
		fields["payment_overrides"] = "ключи — id платежей, суммы ≥ 0"
	}
	plans, ok := validateAmountMap(req.Plans)
	if !ok {
		fields["plans"] = "ключи — id категорий, суммы ≥ 0"
	}

	if len(fields) > 0 {
		return domain.PlannerData{}, &apperrors.ValidationError{Message: "validation failed", Fields: fields}
	}
	return domain.PlannerData{Days: req.Days, Incomes: incomes, PaymentOverrides: overrides, Plans: plans}, nil
}

func validateAmountMap(m map[string]int64) (map[string]int64, bool) {
	out := make(map[string]int64, len(m))
	if len(m) > plannerMaxMapItems {
		return nil, false
	}
	for k, v := range m {
		id, err := strconv.ParseInt(k, 10, 64)
		if err != nil || id <= 0 || v < 0 {
			return nil, false
		}
		out[k] = v
	}
	return out, true
}

func plannerToResponse(p domain.Planner) dto.PlannerResponse {
	resp := dto.PlannerResponse{
		Days:             p.Data.Days,
		Incomes:          p.Data.Incomes,
		PaymentOverrides: p.Data.PaymentOverrides,
		Plans:            p.Data.Plans,
	}
	if resp.Incomes == nil {
		resp.Incomes = DefaultPlannerData().Incomes
	}
	if resp.PaymentOverrides == nil {
		resp.PaymentOverrides = map[string]int64{}
	}
	if resp.Plans == nil {
		resp.Plans = map[string]int64{}
	}
	if p.UpdatedAt != nil {
		v := p.UpdatedAt.UTC().Format(time.RFC3339)
		resp.UpdatedAt = &v
	}
	return resp
}
