package service

import (
	"context"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
)

var validLimitPeriods = map[string]bool{
	domain.LimitPeriodMonth:  true,
	domain.LimitPeriodWeek:   true,
	domain.LimitPeriodCustom: true,
}

type SpendingLimitService struct {
	repo    *repository.SpendingLimitRepository
	tagRepo *repository.TagRepository
	tagSvc  *TagService
}

func NewSpendingLimitService(repo *repository.SpendingLimitRepository, tagRepo *repository.TagRepository, tagSvc *TagService) *SpendingLimitService {
	return &SpendingLimitService{repo: repo, tagRepo: tagRepo, tagSvc: tagSvc}
}

func (s *SpendingLimitService) List(ctx context.Context, projectID int64) ([]dto.SpendingLimitResponse, error) {
	limits, err := s.repo.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.SpendingLimitResponse, 0, len(limits))
	for _, l := range limits {
		r, err := s.toResponse(ctx, l, projectID)
		if err != nil {
			return nil, err
		}
		resp = append(resp, r)
	}
	return resp, nil
}

func (s *SpendingLimitService) Create(ctx context.Context, req dto.CreateSpendingLimitRequest, projectID int64) (dto.SpendingLimitResponse, error) {
	l, err := s.validateAndBuild(ctx, req.TagID, req.Amount, req.PeriodType, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return dto.SpendingLimitResponse{}, err
	}
	created, err := s.repo.Create(ctx, l, projectID)
	if err != nil {
		return dto.SpendingLimitResponse{}, err
	}
	return s.toResponse(ctx, created, projectID)
}

func (s *SpendingLimitService) Update(ctx context.Context, id int64, req dto.UpdateSpendingLimitRequest, projectID int64) (dto.SpendingLimitResponse, error) {
	l, err := s.validateAndBuild(ctx, req.TagID, req.Amount, req.PeriodType, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return dto.SpendingLimitResponse{}, err
	}
	l.ID = id
	updated, err := s.repo.Update(ctx, l)
	if err != nil {
		return dto.SpendingLimitResponse{}, err
	}
	return s.toResponse(ctx, updated, projectID)
}

func (s *SpendingLimitService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *SpendingLimitService) validateAndBuild(
	ctx context.Context,
	tagID int64, amount int64, periodType string, periodStart, periodEnd *string,
) (domain.SpendingLimit, error) {
	fields := map[string]string{}

	if tagID <= 0 {
		fields["tag_id"] = "обязательное поле"
	} else if err := ValidateTagExists(ctx, s.tagRepo, tagID); err != nil {
		fields["tag_id"] = "тег не найден"
	}
	if amount <= 0 {
		fields["amount"] = "должно быть > 0"
	}
	if !validLimitPeriods[periodType] {
		fields["period_type"] = "недопустимое значение"
	}

	var start, end *time.Time
	if periodType == domain.LimitPeriodCustom {
		if periodStart == nil || periodEnd == nil {
			fields["period_start"] = "обязательно для period_type=custom"
		} else {
			s, errS := time.Parse("2006-01-02", *periodStart)
			e, errE := time.Parse("2006-01-02", *periodEnd)
			if errS != nil || errE != nil {
				fields["period_start"] = "не соответствует формату YYYY-MM-DD"
			} else {
				start, end = &s, &e
			}
		}
	}

	if len(fields) > 0 {
		return domain.SpendingLimit{}, &apperrors.ValidationError{Message: "validation failed", Fields: fields}
	}

	return domain.SpendingLimit{
		TagID:       tagID,
		Amount:      amount,
		PeriodType:  periodType,
		PeriodStart: start,
		PeriodEnd:   end,
	}, nil
}

// periodBounds вычисляет [from, to) для лимита: текущий календарный месяц/неделя
// (ISO, понедельник-воскресенье) для month/week, либо явный period_start/period_end
// для custom (period_end включительно — to = period_end + 1 день).
func periodBounds(l domain.SpendingLimit) (time.Time, time.Time) {
	now := time.Now().UTC()
	switch l.PeriodType {
	case domain.LimitPeriodWeek:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(weekday - 1))
		return from, from.AddDate(0, 0, 7)
	case domain.LimitPeriodCustom:
		if l.PeriodStart != nil && l.PeriodEnd != nil {
			return *l.PeriodStart, l.PeriodEnd.AddDate(0, 0, 1)
		}
		fallthrough
	default: // month
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return from, from.AddDate(0, 1, 0)
	}
}

func (s *SpendingLimitService) toResponse(ctx context.Context, l domain.SpendingLimit, projectID int64) (dto.SpendingLimitResponse, error) {
	tag, err := s.tagSvc.TagWithParent(ctx, l.TagID)
	if err != nil {
		return dto.SpendingLimitResponse{}, err
	}
	from, to := periodBounds(l)
	spent, err := s.repo.SumSpentInPeriod(ctx, l.TagID, from, to, projectID)
	if err != nil {
		return dto.SpendingLimitResponse{}, err
	}

	resp := dto.SpendingLimitResponse{
		ID:         l.ID,
		TagID:      l.TagID,
		Tag:        tag,
		Amount:     l.Amount,
		Spent:      spent,
		PeriodType: l.PeriodType,
	}
	if l.PeriodStart != nil {
		v := l.PeriodStart.Format("2006-01-02")
		resp.PeriodStart = &v
	}
	if l.PeriodEnd != nil {
		v := l.PeriodEnd.Format("2006-01-02")
		resp.PeriodEnd = &v
	}
	return resp, nil
}
