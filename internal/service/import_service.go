package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
)

const dateLayout = "2006-01-02"

type ImportService struct {
	repo    *repository.ImportRepository
	tagRepo *repository.TagRepository
}

func NewImportService(repo *repository.ImportRepository, tagRepo *repository.TagRepository) *ImportService {
	return &ImportService{repo: repo, tagRepo: tagRepo}
}

func (s *ImportService) UploadBatch(ctx context.Context, fileName string, r io.Reader) (domain.ImportBatch, []domain.ModerationRow, error) {
	records, err := parseCSV(r)
	if err != nil {
		return domain.ImportBatch{}, nil, &apperrors.ValidationError{Message: err.Error(), Fields: map[string]string{"file": err.Error()}}
	}

	tags, err := s.tagRepo.List(ctx)
	if err != nil {
		return domain.ImportBatch{}, nil, err
	}

	batch, err := s.repo.CreateBatch(ctx, fileName, len(records))
	if err != nil {
		return domain.ImportBatch{}, nil, err
	}

	rows := make([]domain.ModerationRow, 0, len(records))
	for i, rec := range records {
		rows = append(rows, buildAndValidateRow(batch.ID, i+1, rec, tags))
	}

	inserted, err := s.repo.InsertRows(ctx, rows)
	if err != nil {
		return domain.ImportBatch{}, nil, err
	}
	return batch, inserted, nil
}

func (s *ImportService) GetActiveBatch(ctx context.Context) (domain.ImportBatch, []domain.ModerationRow, bool, error) {
	batch, ok, err := s.repo.GetActiveBatch(ctx)
	if err != nil || !ok {
		return domain.ImportBatch{}, nil, false, err
	}
	rows, err := s.repo.ListRows(ctx, batch.ID)
	if err != nil {
		return domain.ImportBatch{}, nil, false, err
	}
	return batch, rows, true, nil
}

// UpdateRowInput описывает PATCH-правку строки модерации; nil-поле = не редактировалось.
type UpdateRowInput struct {
	Title       *string
	Amount      *string
	Date        *string
	TagID       *int64 // 0 = очистить тег
	Category    *string
	Specificity *string
	Comment     *string
	URL         *string
}

func (s *ImportService) UpdateRow(ctx context.Context, id int64, in UpdateRowInput) (domain.ModerationRow, error) {
	row, err := s.repo.GetRow(ctx, id)
	if err != nil {
		return domain.ModerationRow{}, err
	}

	errs := make(map[string]string, len(row.FieldErrors))
	for k, v := range row.FieldErrors {
		errs[k] = v
	}

	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return domain.ModerationRow{}, &apperrors.ValidationError{Fields: map[string]string{"title": "не может быть пустым"}}
		}
		row.Title = &title
		delete(errs, "title")
	}

	if in.Amount != nil {
		raw := strings.TrimSpace(*in.Amount)
		if raw == "" {
			return domain.ModerationRow{}, &apperrors.ValidationError{Fields: map[string]string{"amount": "не может быть пустым"}}
		}
		if amount, err := parseAmount(raw); err != nil {
			errs["amount"] = fmt.Sprintf("не удалось распознать значение %q", raw)
			row.Amount = nil
		} else {
			row.Amount = &amount
			delete(errs, "amount")
		}
	}

	if in.Date != nil {
		raw := strings.TrimSpace(*in.Date)
		if raw == "" {
			return domain.ModerationRow{}, &apperrors.ValidationError{Fields: map[string]string{"date": "не может быть пустым"}}
		}
		if d, err := time.Parse(dateLayout, raw); err != nil {
			errs["date"] = fmt.Sprintf("значение %q не соответствует формату YYYY-MM-DD", raw)
			row.Date = nil
		} else {
			row.Date = &d
			delete(errs, "date")
		}
	}

	if in.Comment != nil {
		row.Comment = in.Comment
	}
	if in.URL != nil {
		row.URL = in.URL
	}

	if in.Category != nil {
		cat := strings.TrimSpace(*in.Category)
		switch {
		case cat == "":
			row.Category = nil
			delete(errs, "category")
		case cat != "expense" && cat != "income":
			// Не сохраняем невалидное значение — иначе INSERT/UPDATE в enum-колонку упадёт с ошибкой БД.
			errs["category"] = fmt.Sprintf("значение %q недопустимо, ожидается expense или income", cat)
			row.Category = nil
		default:
			row.Category = &cat
			delete(errs, "category")
		}
	}

	if in.Specificity != nil {
		sp := strings.TrimSpace(*in.Specificity)
		switch {
		case sp == "":
			row.Specificity = nil
			delete(errs, "specificity")
		case sp != "required" && sp != "simple":
			// Аналогично: не сохраняем, чтобы не сломать enum-cast.
			errs["specificity"] = fmt.Sprintf("значение %q недопустимо, ожидается required или simple", sp)
			row.Specificity = nil
		default:
			row.Specificity = &sp
			delete(errs, "specificity")
		}
	}

	if in.TagID != nil {
		if *in.TagID == 0 {
			row.TagID = nil
			delete(errs, "tag")
		} else {
			exists, err := s.tagRepo.Exists(ctx, *in.TagID)
			if err != nil {
				return domain.ModerationRow{}, err
			}
			if !exists {
				errs["tag"] = fmt.Sprintf("метка с id %d не найдена в системе", *in.TagID)
				row.TagID = nil
			} else {
				row.TagID = in.TagID
				delete(errs, "tag")
			}
		}
	}

	row.FieldErrors = errs
	recomputeStatus(&row)
	return s.repo.UpdateRow(ctx, row)
}

func (s *ImportService) AcceptRow(ctx context.Context, id int64) (domain.Transaction, error) {
	row, err := s.repo.GetRow(ctx, id)
	if err != nil {
		return domain.Transaction{}, err
	}
	if row.Status != domain.ModerationRowStatusReady {
		return domain.Transaction{}, &apperrors.ValidationError{Message: "строка не готова к принятию"}
	}
	created, err := s.repo.AcceptRows(ctx, row.BatchID, []int64{id})
	if err != nil {
		return domain.Transaction{}, err
	}
	if len(created) == 0 {
		return domain.Transaction{}, &apperrors.ValidationError{Message: "строка не готова к принятию"}
	}
	return created[0], nil
}

// AcceptBatch принимает только те строки, чей статус ready.
// repo.AcceptRows тоже фильтрует по status='ready', но мы проверяем заранее
// чтобы вернуть понятную ошибку вместо молчаливого пропуска.
func (s *ImportService) AcceptBatch(ctx context.Context, batchID int64, rowIDs []int64) ([]domain.Transaction, error) {
	if len(rowIDs) == 0 {
		return nil, &apperrors.ValidationError{Message: "не указаны строки для принятия"}
	}
	return s.repo.AcceptRows(ctx, batchID, rowIDs)
}

func (s *ImportService) CloseBatch(ctx context.Context, batchID int64) error {
	return s.repo.CloseBatch(ctx, batchID)
}

type rawRecord struct {
	Title       string
	Amount      string
	Date        string
	Tag         string
	Category    string
	Specificity string
	Comment     string
	URL         string
}

func parseCSV(r io.Reader) ([]rawRecord, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать заголовок CSV: %w", err)
	}
	colIndex := make(map[string]int, len(header))
	for i, h := range header {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, required := range []string{"title", "amount", "date"} {
		if _, ok := colIndex[required]; !ok {
			return nil, fmt.Errorf("в файле отсутствует обязательная колонка %q", required)
		}
	}

	get := func(rec []string, col string) string {
		idx, ok := colIndex[col]
		if !ok || idx >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[idx])
	}

	var records []rawRecord
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("ошибка разбора CSV: %w", err)
		}
		records = append(records, rawRecord{
			Title:       get(rec, "title"),
			Amount:      get(rec, "amount"),
			Date:        get(rec, "date"),
			Tag:         get(rec, "tag"),
			Category:    get(rec, "category"),
			Specificity: get(rec, "specificity"),
			Comment:     get(rec, "comment"),
			URL:         get(rec, "url"),
		})
	}
	return records, nil
}

func buildAndValidateRow(batchID int64, rowNumber int, rec rawRecord, tags []domain.Tag) domain.ModerationRow {
	row := domain.ModerationRow{
		BatchID:     batchID,
		RowNumber:   rowNumber,
		FieldErrors: map[string]string{},
	}

	if rec.Title == "" {
		row.FieldErrors["title"] = "обязательное поле не заполнено"
	} else {
		title := rec.Title
		row.Title = &title
	}

	if rec.Amount == "" {
		row.FieldErrors["amount"] = "обязательное поле не заполнено"
	} else if amount, err := parseAmount(rec.Amount); err != nil {
		row.FieldErrors["amount"] = fmt.Sprintf("не удалось распознать значение %q", rec.Amount)
	} else {
		row.Amount = &amount
	}

	if rec.Date == "" {
		row.FieldErrors["date"] = "обязательное поле не заполнено"
	} else if d, err := time.Parse(dateLayout, rec.Date); err != nil {
		row.FieldErrors["date"] = fmt.Sprintf("значение %q не соответствует формату YYYY-MM-DD", rec.Date)
	} else {
		row.Date = &d
	}

	if rec.Category != "" {
		cat := rec.Category
		if cat != "expense" && cat != "income" {
			// Не сохраняем невалидное значение — иначе INSERT в enum-колонку упадёт с ошибкой БД.
			row.FieldErrors["category"] = fmt.Sprintf("значение %q недопустимо, ожидается expense или income", cat)
		} else {
			row.Category = &cat
		}
	}

	if rec.Specificity != "" {
		sp := rec.Specificity
		if sp != "required" && sp != "simple" {
			// Аналогично: не сохраняем, чтобы не сломать enum-cast.
			row.FieldErrors["specificity"] = fmt.Sprintf("значение %q недопустимо, ожидается required или simple", sp)
		} else {
			row.Specificity = &sp
		}
	}

	if rec.Tag != "" {
		if id, ok := resolveTagPath(tags, rec.Tag); ok {
			row.TagID = &id
		} else {
			row.FieldErrors["tag"] = fmt.Sprintf("тег %q не найден в системе", rec.Tag)
		}
	}

	if rec.Comment != "" {
		comment := rec.Comment
		row.Comment = &comment
	}
	if rec.URL != "" {
		url := rec.URL
		row.URL = &url
	}

	recomputeStatus(&row)
	return row
}

// resolveTagPath ищет тег по пути "родитель/потомок" с точным совпадением имён
// (без учёта регистра и пробелов) на каждом уровне иерархии.
func resolveTagPath(tags []domain.Tag, path string) (int64, bool) {
	parts := strings.Split(path, "/")
	var parentID *int64
	var currentID int64
	matched := false

	for _, part := range parts {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			return 0, false
		}
		found := false
		for _, t := range tags {
			if strings.ToLower(strings.TrimSpace(t.Name)) != name {
				continue
			}
			if !sameParentID(t.ParentID, parentID) {
				continue
			}
			id := t.ID
			currentID = id
			parentID = &id
			found = true
			matched = true
			break
		}
		if !found {
			return 0, false
		}
	}
	return currentID, matched
}

func sameParentID(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func recomputeStatus(row *domain.ModerationRow) {
	if len(row.FieldErrors) > 0 {
		row.Status = domain.ModerationRowStatusError
		return
	}
	if row.TagID == nil || row.Category == nil || row.Specificity == nil {
		row.Status = domain.ModerationRowStatusPending
		return
	}
	row.Status = domain.ModerationRowStatusReady
}

func parseAmount(raw string) (int64, error) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(raw), " ", "")
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	f, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, err
	}
	return int64(math.Round(f * 100)), nil
}
