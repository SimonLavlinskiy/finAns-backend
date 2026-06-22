package service

import (
	"context"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
)

var validRecurrences = map[string]bool{
	"daily": true, "weekly": true, "monthly": true,
	"quarterly": true, "semi_annual": true, "yearly": true,
}

type MandatoryPaymentService struct {
	repo    *repository.MandatoryPaymentRepository
	tagRepo *repository.TagRepository
	tagSvc  *TagService
	txRepo  *repository.TransactionRepository
}

func NewMandatoryPaymentService(
	repo *repository.MandatoryPaymentRepository,
	tagRepo *repository.TagRepository,
	tagSvc *TagService,
	txRepo *repository.TransactionRepository,
) *MandatoryPaymentService {
	return &MandatoryPaymentService{repo: repo, tagRepo: tagRepo, tagSvc: tagSvc, txRepo: txRepo}
}

func (s *MandatoryPaymentService) List(ctx context.Context) ([]dto.MandatoryPaymentResponse, error) {
	payments, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	responses := make([]dto.MandatoryPaymentResponse, 0, len(payments))
	for _, p := range payments {
		resp, err := s.toResponse(ctx, p)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}
	return responses, nil
}

func (s *MandatoryPaymentService) Get(ctx context.Context, id int64) (dto.MandatoryPaymentResponse, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.MandatoryPaymentResponse{}, err
	}
	return s.toResponse(ctx, p)
}

func (s *MandatoryPaymentService) Create(ctx context.Context, req dto.CreateMandatoryPaymentRequest) (dto.MandatoryPaymentResponse, error) {
	p, err := s.validateAndBuild(ctx, req.Title, req.Amount, req.TagID, req.Recurrence, req.NextPaymentDate)
	if err != nil {
		return dto.MandatoryPaymentResponse{}, err
	}
	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return dto.MandatoryPaymentResponse{}, err
	}
	return s.toResponse(ctx, created)
}

func (s *MandatoryPaymentService) Update(ctx context.Context, id int64, req dto.UpdateMandatoryPaymentRequest) (dto.MandatoryPaymentResponse, error) {
	p, err := s.validateAndBuild(ctx, req.Title, req.Amount, req.TagID, req.Recurrence, req.NextPaymentDate)
	if err != nil {
		return dto.MandatoryPaymentResponse{}, err
	}
	p.ID = id
	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		return dto.MandatoryPaymentResponse{}, err
	}
	return s.toResponse(ctx, updated)
}

func (s *MandatoryPaymentService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *MandatoryPaymentService) Duplicate(ctx context.Context, id int64) (dto.MandatoryPaymentResponse, error) {
	orig, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.MandatoryPaymentResponse{}, err
	}
	copy := domain.MandatoryPayment{
		Title:           orig.Title,
		Amount:          orig.Amount,
		TagID:           orig.TagID,
		Recurrence:      orig.Recurrence,
		NextPaymentDate: time.Now().Truncate(24 * time.Hour),
	}
	created, err := s.repo.Create(ctx, copy)
	if err != nil {
		return dto.MandatoryPaymentResponse{}, err
	}
	return s.toResponse(ctx, created)
}

// MarkPaid фиксирует факт оплаты: создаёт транзакцию-расход за платёж
// и сдвигает next_payment_date на следующий период.
func (s *MandatoryPaymentService) MarkPaid(ctx context.Context, id int64) (dto.MandatoryPaymentResponse, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.MandatoryPaymentResponse{}, err
	}

	_, err = s.txRepo.Create(ctx, domain.Transaction{
		Title:       p.Title,
		Amount:      p.Amount,
		Date:        p.NextPaymentDate,
		TagID:       p.TagID,
		Category:    "expense",
		Specificity: "required",
	})
	if err != nil {
		return dto.MandatoryPaymentResponse{}, err
	}

	newDate := advanceDate(p.NextPaymentDate, p.Recurrence)
	updated, err := s.repo.AdvanceDate(ctx, id, newDate)
	if err != nil {
		return dto.MandatoryPaymentResponse{}, err
	}
	return s.toResponse(ctx, updated)
}

// advanceDate вычисляет следующую дату платежа по типу периодичности.
// Для месячных интервалов используем addMonthsClamped чтобы избежать
// переполнения (например Jan 31 + 1 month = Feb 28, не Mar 3).
func advanceDate(t time.Time, recurrence string) time.Time {
	switch recurrence {
	case "daily":
		return t.AddDate(0, 0, 1)
	case "weekly":
		return t.AddDate(0, 0, 7)
	case "monthly":
		return addMonthsClamped(t, 1)
	case "quarterly":
		return addMonthsClamped(t, 3)
	case "semi_annual":
		return addMonthsClamped(t, 6)
	case "yearly":
		return addMonthsClamped(t, 12)
	default:
		return addMonthsClamped(t, 1)
	}
}

// addMonthsClamped добавляет n месяцев, ограничивая день последним днём целевого месяца.
func addMonthsClamped(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	targetMonth := time.Month(int(m) + months)
	targetYear := y
	for targetMonth > 12 {
		targetMonth -= 12
		targetYear++
	}
	// Последний день целевого месяца
	lastDay := time.Date(targetYear, targetMonth+1, 0, 0, 0, 0, 0, t.Location()).Day()
	if d > lastDay {
		d = lastDay
	}
	return time.Date(targetYear, targetMonth, d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

func (s *MandatoryPaymentService) validateAndBuild(
	ctx context.Context,
	title string, amount int64, tagID int64, recurrence string, nextPaymentDate string,
) (domain.MandatoryPayment, error) {
	fields := map[string]string{}

	if title == "" {
		fields["title"] = "обязательное поле"
	}
	if amount <= 0 {
		fields["amount"] = "должно быть > 0"
	}
	if !validRecurrences[recurrence] {
		fields["recurrence"] = "недопустимое значение"
	}

	var nextDate time.Time
	if nextPaymentDate == "" {
		fields["next_payment_date"] = "обязательное поле"
	} else {
		var err error
		nextDate, err = time.Parse("2006-01-02", nextPaymentDate)
		if err != nil {
			fields["next_payment_date"] = "не соответствует формату YYYY-MM-DD"
		}
	}

	if tagID <= 0 {
		fields["tag_id"] = "обязательное поле"
	} else if err := ValidateTagExists(ctx, s.tagRepo, tagID); err != nil {
		fields["tag_id"] = "тег не найден"
	}

	if len(fields) > 0 {
		return domain.MandatoryPayment{}, &apperrors.ValidationError{
			Message: "validation failed",
			Fields:  fields,
		}
	}

	return domain.MandatoryPayment{
		Title:           title,
		Amount:          amount,
		TagID:           tagID,
		Recurrence:      recurrence,
		NextPaymentDate: nextDate,
	}, nil
}

func (s *MandatoryPaymentService) toResponse(ctx context.Context, p domain.MandatoryPayment) (dto.MandatoryPaymentResponse, error) {
	tag, err := s.tagSvc.TagWithParent(ctx, p.TagID)
	if err != nil {
		return dto.MandatoryPaymentResponse{}, err
	}
	return dto.MandatoryPaymentResponse{
		ID:              p.ID,
		Title:           p.Title,
		Amount:          p.Amount,
		Tag:             tag,
		Recurrence:      p.Recurrence,
		NextPaymentDate: p.NextPaymentDate.Format("2006-01-02"),
		CreatedAt:       p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       p.UpdatedAt.Format(time.RFC3339),
	}, nil
}
