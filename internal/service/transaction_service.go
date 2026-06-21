package service

import (
	"context"
	"strings"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
)

type TransactionService struct {
	txRepo  *repository.TransactionRepository
	tagRepo *repository.TagRepository
	tagSvc  *TagService
	fileSvc *FileService
}

func NewTransactionService(txRepo *repository.TransactionRepository, tagRepo *repository.TagRepository, tagSvc *TagService, fileSvc *FileService) *TransactionService {
	return &TransactionService{txRepo: txRepo, tagRepo: tagRepo, tagSvc: tagSvc, fileSvc: fileSvc}
}

func (s *TransactionService) List(ctx context.Context, f domain.TransactionFilters) ([]dto.TransactionResponse, domain.ListResult, error) {
	normalizeFilters(&f)
	result, err := s.txRepo.List(ctx, f)
	if err != nil {
		return nil, domain.ListResult{}, err
	}

	// Батч-загрузка тегов: один запрос вместо N запросов (устраняет N+1)
	tagIDs := make([]int64, len(result.Items))
	for i, t := range result.Items {
		tagIDs[i] = t.TagID
	}
	tagMap, err := s.tagSvc.TagsWithParentBatch(ctx, tagIDs)
	if err != nil {
		return nil, domain.ListResult{}, err
	}

	responses := make([]dto.TransactionResponse, 0, len(result.Items))
	for _, t := range result.Items {
		tag := tagMap[t.TagID]
		resp := dto.TransactionResponse{
			ID:          t.ID,
			Title:       t.Title,
			Amount:      t.Amount,
			Date:        t.Date.Format("2006-01-02"),
			Tag:         tag,
			Category:    t.Category,
			Specificity: t.Specificity,
			Comment:     t.Comment,
			URL:         t.URL,
			CreatedAt:   t.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
		}
		if t.FilePath != nil && *t.FilePath != "" {
			resp.File = &dto.FileAttachment{
				Path:     *t.FilePath,
				Name:     deref(t.FileName),
				MimeType: deref(t.FileMIME),
			}
		}
		responses = append(responses, resp)
	}
	return responses, result, nil
}

func (s *TransactionService) Get(ctx context.Context, id int64) (dto.TransactionResponse, error) {
	t, err := s.txRepo.GetByID(ctx, id)
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	return s.toResponse(ctx, t)
}

func (s *TransactionService) Create(ctx context.Context, req dto.CreateTransactionRequest) (dto.TransactionResponse, error) {
	t, err := s.validateAndBuild(ctx, req.Title, req.Amount, req.Date, req.TagID, req.Category, req.Specificity, req.Comment, req.URL)
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	created, err := s.txRepo.Create(ctx, t)
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	return s.toResponse(ctx, created)
}

func (s *TransactionService) Update(ctx context.Context, id int64, req dto.UpdateTransactionRequest) (dto.TransactionResponse, error) {
	existing, err := s.txRepo.GetByID(ctx, id)
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	t, err := s.validateAndBuild(ctx, req.Title, req.Amount, req.Date, req.TagID, req.Category, req.Specificity, req.Comment, req.URL)
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	t.ID = id
	t.FilePath = existing.FilePath
	t.FileName = existing.FileName
	t.FileMIME = existing.FileMIME
	updated, err := s.txRepo.Update(ctx, t)
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	return s.toResponse(ctx, updated)
}

// Delete удаляет транзакцию и связанный файл с диска.
func (s *TransactionService) Delete(ctx context.Context, id int64) error {
	tx, err := s.txRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	// Сначала удаляем файл, если есть
	if tx.FilePath != nil && *tx.FilePath != "" {
		if err := s.fileSvc.Delete(ctx, id); err != nil {
			// Логируем, но не блокируем удаление транзакции
			_ = err
		}
	}
	return s.txRepo.Delete(ctx, id)
}

// Duplicate создаёт копию без file_path — каждый файл должен принадлежать одной записи.
func (s *TransactionService) Duplicate(ctx context.Context, id int64) (dto.TransactionResponse, error) {
	existing, err := s.txRepo.GetByID(ctx, id)
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	copy := existing
	copy.ID = 0
	copy.Date = time.Now().UTC().Truncate(24 * time.Hour)
	copy.CreatedAt = time.Time{}
	copy.UpdatedAt = time.Time{}
	// Файл не копируем — у каждой транзакции свой файл
	copy.FilePath = nil
	copy.FileName = nil
	copy.FileMIME = nil

	created, err := s.txRepo.Create(ctx, copy)
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	return s.toResponse(ctx, created)
}

func (s *TransactionService) Suggestions(ctx context.Context, q string) ([]string, error) {
	return s.txRepo.Suggestions(ctx, q, 10)
}

func (s *TransactionService) toResponse(ctx context.Context, t domain.Transaction) (dto.TransactionResponse, error) {
	tag, err := s.tagSvc.TagWithParent(ctx, t.TagID)
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	resp := dto.TransactionResponse{
		ID:          t.ID,
		Title:       t.Title,
		Amount:      t.Amount,
		Date:        t.Date.Format("2006-01-02"),
		Tag:         tag,
		Category:    t.Category,
		Specificity: t.Specificity,
		Comment:     t.Comment,
		URL:         t.URL,
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
	}
	if t.FilePath != nil && *t.FilePath != "" {
		resp.File = &dto.FileAttachment{
			Path:     *t.FilePath,
			Name:     deref(t.FileName),
			MimeType: deref(t.FileMIME),
		}
	}
	return resp, nil
}

func (s *TransactionService) validateAndBuild(ctx context.Context, title string, amount int64, dateStr string, tagID int64, category, specificity string, comment, rawURL *string) (domain.Transaction, error) {
	fields := map[string]string{}
	if strings.TrimSpace(title) == "" {
		fields["title"] = "required"
	}
	if amount <= 0 {
		fields["amount"] = "must be positive"
	}
	if category != "expense" && category != "income" {
		fields["category"] = "must be expense or income"
	}
	if specificity != "required" && specificity != "simple" {
		fields["specificity"] = "must be required or simple"
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		fields["date"] = "invalid format, use YYYY-MM-DD"
	}
	if rawURL != nil && *rawURL != "" && !isValidURL(*rawURL) {
		fields["url"] = "invalid URL"
	}
	if tagID <= 0 {
		fields["tag_id"] = "required"
	}
	if len(fields) > 0 {
		return domain.Transaction{}, &apperrors.ValidationError{Fields: fields, Message: "validation failed"}
	}
	if err := ValidateTagExists(ctx, s.tagRepo, tagID); err != nil {
		return domain.Transaction{}, err
	}

	return domain.Transaction{
		Title:       title,
		Amount:      amount,
		Date:        date,
		TagID:       tagID,
		Category:    category,
		Specificity: specificity,
		Comment:     comment,
		URL:         rawURL,
	}, nil
}

func normalizeFilters(f *domain.TransactionFilters) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 25
	}
	if f.PerPage > 100 {
		f.PerPage = 100
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
