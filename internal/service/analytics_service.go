package service

import (
	"context"
	"fmt"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
)

type AnalyticsService struct {
	repo    *repository.AnalyticsRepository
	tagRepo *repository.TagRepository
}

func NewAnalyticsService(repo *repository.AnalyticsRepository, tagRepo *repository.TagRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo, tagRepo: tagRepo}
}

func (s *AnalyticsService) GetExpensesCalendar(ctx context.Context, level string, year, month int) (domain.CalendarResult, error) {
	fields := map[string]string{}
	if level != string(domain.CalendarLevelDay) && level != string(domain.CalendarLevelMonth) {
		fields["level"] = "must be day or month"
	}
	if year < 1 {
		fields["year"] = "required"
	}
	if level == string(domain.CalendarLevelDay) && (month < 1 || month > 12) {
		fields["month"] = "required and must be 1..12 for level=day"
	}
	if len(fields) > 0 {
		return domain.CalendarResult{}, &apperrors.ValidationError{Fields: fields, Message: "validation failed"}
	}

	now := time.Now().UTC()

	if level == string(domain.CalendarLevelMonth) {
		return s.getYearLevel(ctx, year, now)
	}
	return s.getMonthLevel(ctx, year, month, now)
}

func (s *AnalyticsService) getYearLevel(ctx context.Context, year int, now time.Time) (domain.CalendarResult, error) {
	sums, err := s.repo.SumByMonthInYear(ctx, year)
	if err != nil {
		return domain.CalendarResult{}, err
	}
	byMonth := make(map[int]int64, len(sums))
	for _, ms := range sums {
		byMonth[ms.Month] = ms.Amount
	}

	periodStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	hasPrevious, err := s.repo.ExistsExpenseBefore(ctx, periodStart)
	if err != nil {
		return domain.CalendarResult{}, err
	}

	var total int64
	items := make([]domain.CalendarItem, 0, 12)
	for m := 1; m <= 12; m++ {
		amount := byMonth[m]
		total += amount
		items = append(items, domain.CalendarItem{
			Key:       fmt.Sprintf("%04d-%02d", year, m),
			Amount:    amount,
			HasData:   amount > 0,
			IsCurrent: now.Year() == year && int(now.Month()) == m,
		})
	}

	return domain.CalendarResult{
		Level:       domain.CalendarLevelMonth,
		Year:        year,
		Total:       total,
		HasPrevious: hasPrevious,
		Items:       items,
	}, nil
}

func (s *AnalyticsService) getMonthLevel(ctx context.Context, year, month int, now time.Time) (domain.CalendarResult, error) {
	sums, err := s.repo.SumByDayInMonth(ctx, year, month)
	if err != nil {
		return domain.CalendarResult{}, err
	}
	byDay := make(map[int]int64, len(sums))
	for _, ds := range sums {
		byDay[ds.Day] = ds.Amount
	}

	transactions, err := s.repo.ListExpenseTransactionsInMonth(ctx, year, month)
	if err != nil {
		return domain.CalendarResult{}, err
	}
	byDayTxs := make(map[int][]domain.TransactionBrief)
	for _, t := range transactions {
		byDayTxs[t.Day] = append(byDayTxs[t.Day], t)
	}

	rootOf, err := s.tagRootMap(ctx)
	if err != nil {
		return domain.CalendarResult{}, err
	}

	periodStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	hasPrevious, err := s.repo.ExistsExpenseBefore(ctx, periodStart)
	if err != nil {
		return domain.CalendarResult{}, err
	}

	daysInMonth := periodStart.AddDate(0, 1, -1).Day()

	var total int64
	items := make([]domain.CalendarItem, 0, daysInMonth)
	for d := 1; d <= daysInMonth; d++ {
		amount := byDay[d]
		total += amount

		item := domain.CalendarItem{
			Key:       fmt.Sprintf("%04d-%02d-%02d", year, month, d),
			Amount:    amount,
			HasData:   amount > 0,
			IsCurrent: now.Year() == year && int(now.Month()) == month && now.Day() == d,
		}

		if dayTxs, ok := byDayTxs[d]; ok {
			item.BreakdownByTag = buildTagBreakdown(dayTxs, rootOf, amount)
			item.Transactions = make([]domain.CalendarTransactionBrief, 0, len(dayTxs))
			for _, t := range dayTxs {
				item.Transactions = append(item.Transactions, domain.CalendarTransactionBrief{
					ID:     t.ID,
					Title:  t.Title,
					Amount: t.Amount,
				})
			}
		}

		items = append(items, item)
	}

	return domain.CalendarResult{
		Level:       domain.CalendarLevelDay,
		Year:        year,
		Month:       month,
		Total:       total,
		HasPrevious: hasPrevious,
		Items:       items,
	}, nil
}

type tagInfo struct {
	rootID int64
	name   string
	color  string
}

// tagRootMap строит map tagID -> корневой тег (name/color корня) для свёртки подтегов в breakdown.
func (s *AnalyticsService) tagRootMap(ctx context.Context) (map[int64]tagInfo, error) {
	tags, err := s.tagRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]domain.Tag, len(tags))
	for _, t := range tags {
		byID[t.ID] = t
	}

	result := make(map[int64]tagInfo, len(tags))
	for _, t := range tags {
		root := t
		for root.ParentID != nil {
			parent, ok := byID[*root.ParentID]
			if !ok {
				break
			}
			root = parent
		}
		result[t.ID] = tagInfo{rootID: root.ID, name: root.Name, color: root.Color}
	}
	return result, nil
}

func buildTagBreakdown(txs []domain.TransactionBrief, rootOf map[int64]tagInfo, dayTotal int64) []domain.CalendarTagBreakdown {
	sums := make(map[int64]int64)
	infoByRoot := make(map[int64]tagInfo)
	order := make([]int64, 0)
	for _, t := range txs {
		info, ok := rootOf[t.TagID]
		if !ok {
			info = tagInfo{rootID: t.TagID}
		}
		if _, seen := sums[info.rootID]; !seen {
			order = append(order, info.rootID)
			infoByRoot[info.rootID] = info
		}
		sums[info.rootID] += t.Amount
	}

	breakdown := make([]domain.CalendarTagBreakdown, 0, len(order))
	for _, rootID := range order {
		info := infoByRoot[rootID]
		var percent float64
		if dayTotal > 0 {
			percent = float64(sums[rootID]) / float64(dayTotal) * 100
		}
		breakdown = append(breakdown, domain.CalendarTagBreakdown{
			TagID:   rootID,
			Name:    info.name,
			Color:   info.color,
			Amount:  sums[rootID],
			Percent: percent,
		})
	}
	return breakdown
}
