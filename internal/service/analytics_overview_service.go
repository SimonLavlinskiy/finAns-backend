package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
)

const monthLayout = "2006-01"

// AnalyticsOverviewService — расчёты нового раздела «Аналитика»:
// сводка месяца, динамика по году и диапазон данных.
type AnalyticsOverviewService struct {
	repo    *repository.AnalyticsRepository
	tagRepo *repository.TagRepository
	clock   Clock
}

func NewAnalyticsOverviewService(repo *repository.AnalyticsRepository, tagRepo *repository.TagRepository, clock Clock) *AnalyticsOverviewService {
	return &AnalyticsOverviewService{repo: repo, tagRepo: tagRepo, clock: clock}
}

// ParseMonth разбирает YYYY-MM и проверяет, что месяц не в будущем.
func ParseMonth(raw string, now time.Time) (time.Time, error) {
	m, err := time.Parse(monthLayout, raw)
	if err != nil {
		return time.Time{}, &apperrors.ValidationError{
			Message: "validation failed",
			Fields:  map[string]string{"month": "формат YYYY-MM"},
		}
	}
	current := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	if m.After(current) {
		return time.Time{}, &apperrors.ValidationError{
			Message: "validation failed",
			Fields:  map[string]string{"month": "будущий месяц"},
		}
	}
	return m, nil
}

func (s *AnalyticsOverviewService) MonthSummary(ctx context.Context, rawMonth string, projectID int64) (dto.MonthSummaryResponse, error) {
	now := s.clock.Now()
	from, err := ParseMonth(rawMonth, now)
	if err != nil {
		return dto.MonthSummaryResponse{}, err
	}
	to := from.AddDate(0, 1, 0)

	amounts, err := s.repo.SumByTagAndCategoryInPeriod(ctx, from, to, projectID)
	if err != nil {
		return dto.MonthSummaryResponse{}, err
	}
	tags, err := s.tagRepo.ListWithDeleted(ctx, projectID)
	if err != nil {
		return dto.MonthSummaryResponse{}, err
	}
	return buildMonthSummary(from, now, amounts, tags), nil
}

// buildMonthSummary сворачивает суммы по тегам в категории верхнего уровня.
// Подкатегория — потомок первого уровня под корнем; операции самого корня — подкатегория nil.
func buildMonthSummary(month, now time.Time, amounts []domain.TagCategoryAmount, tags []domain.Tag) dto.MonthSummaryResponse {
	daysInMonth := month.AddDate(0, 1, -1).Day()
	resp := dto.MonthSummaryResponse{
		Month:       month.Format(monthLayout),
		DaysInMonth: daysInMonth,
		Categories:  []dto.MonthCategoryResponse{},
	}
	if month.Year() == now.Year() && month.Month() == now.Month() {
		resp.IsCurrent = true
		today := now.Day()
		resp.Today = &today
	}

	parentOf := make(map[int64]*int64, len(tags))
	for _, t := range tags {
		parentOf[t.ID] = t.ParentID
	}
	// rootAndSub возвращает корень и подкатегорию первого уровня для тега.
	rootAndSub := func(tagID int64) (int64, *int64) {
		cur := tagID
		var prev *int64
		for i := 0; i < 32; i++ { // защита от циклов
			p, ok := parentOf[cur]
			if !ok || p == nil {
				break
			}
			c := cur
			prev = &c
			cur = *p
		}
		return cur, prev
	}

	type subAgg struct {
		amount       int64
		allMandatory bool
	}
	type catAgg struct {
		amount int64
		subs   map[int64]*subAgg // ключ 0 — без подкатегории
	}
	cats := map[int64]*catAgg{}

	for _, a := range amounts {
		switch a.Category {
		case "income":
			resp.Income += a.Amount
			continue
		case "expense":
			resp.Expense += a.Amount
		default:
			continue
		}
		root, sub := rootAndSub(a.TagID)
		c, ok := cats[root]
		if !ok {
			c = &catAgg{subs: map[int64]*subAgg{}}
			cats[root] = c
		}
		c.amount += a.Amount
		key := int64(0)
		if sub != nil {
			key = *sub
		}
		sa, ok := c.subs[key]
		if !ok {
			sa = &subAgg{allMandatory: true}
			c.subs[key] = sa
		}
		sa.amount += a.Amount
		sa.allMandatory = sa.allMandatory && a.AllMandatory
	}

	for id, c := range cats {
		cat := dto.MonthCategoryResponse{CategoryID: id, Amount: c.amount, Subcategories: []dto.MonthSubcategoryResponse{}}
		for key, sa := range c.subs {
			sub := dto.MonthSubcategoryResponse{Amount: sa.amount, AllMandatory: sa.allMandatory}
			if key != 0 {
				k := key
				sub.SubcategoryID = &k
			}
			cat.Subcategories = append(cat.Subcategories, sub)
		}
		sort.Slice(cat.Subcategories, func(i, j int) bool {
			a, b := cat.Subcategories[i], cat.Subcategories[j]
			if a.Amount != b.Amount {
				return a.Amount > b.Amount
			}
			return subKey(a.SubcategoryID) < subKey(b.SubcategoryID)
		})
		resp.Categories = append(resp.Categories, cat)
	}
	sort.Slice(resp.Categories, func(i, j int) bool {
		a, b := resp.Categories[i], resp.Categories[j]
		if a.Amount != b.Amount {
			return a.Amount > b.Amount
		}
		return a.CategoryID < b.CategoryID
	})
	return resp
}

func subKey(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}

func (s *AnalyticsOverviewService) YearSummary(ctx context.Context, year int, projectID int64) (dto.YearSummaryResponse, error) {
	now := s.clock.Now()
	if year < 1900 || year > now.Year() {
		return dto.YearSummaryResponse{}, &apperrors.ValidationError{
			Message: "validation failed",
			Fields:  map[string]string{"year": fmt.Sprintf("допустимо до %d", now.Year())},
		}
	}
	months, err := s.repo.IncomeExpenseByMonthInYear(ctx, year, projectID)
	if err != nil {
		return dto.YearSummaryResponse{}, err
	}
	return buildYearSummary(year, now, months), nil
}

// buildYearSummary оставляет месяцы с данными до текущего месяца включительно.
func buildYearSummary(year int, now time.Time, months []domain.MonthIncomeExpense) dto.YearSummaryResponse {
	resp := dto.YearSummaryResponse{Year: year, Months: []dto.YearMonthResponse{}}
	for _, m := range months {
		if year == now.Year() && m.Month > int(now.Month()) {
			continue
		}
		resp.Months = append(resp.Months, dto.YearMonthResponse{
			Month:     fmt.Sprintf("%04d-%02d", year, m.Month),
			Income:    m.Income,
			Expense:   m.Expense,
			IsCurrent: year == now.Year() && m.Month == int(now.Month()),
		})
	}
	return resp
}

func (s *AnalyticsOverviewService) Range(ctx context.Context, projectID int64) (dto.AnalyticsRangeResponse, error) {
	now := s.clock.Now()
	first, err := s.repo.FirstTransactionDate(ctx, projectID)
	if err != nil {
		return dto.AnalyticsRangeResponse{}, err
	}
	years, err := s.repo.TransactionYears(ctx, projectID)
	if err != nil {
		return dto.AnalyticsRangeResponse{}, err
	}
	return buildRange(now, first, years), nil
}

func buildRange(now time.Time, first *time.Time, years []int) dto.AnalyticsRangeResponse {
	current := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	resp := dto.AnalyticsRangeResponse{
		CurrentMonth: current.Format(monthLayout),
		Today:        now.Format("2006-01-02"),
		Years:        []int{},
	}
	if first != nil {
		fm := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC)
		if fm.After(current) {
			fm = current
		}
		v := fm.Format(monthLayout)
		resp.FirstMonth = &v
	}
	for _, y := range years {
		if y <= now.Year() {
			resp.Years = append(resp.Years, y)
		}
	}
	return resp
}
