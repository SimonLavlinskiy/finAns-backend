package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
)

type AnalyticsService struct {
	repo    *repository.AnalyticsRepository
	tagRepo *repository.TagRepository
	mpRepo  *repository.MandatoryPaymentRepository
	balRepo *repository.BalanceRepository
	peRepo  *repository.PlannedExpenseRepository
}

func NewAnalyticsService(
	repo *repository.AnalyticsRepository,
	tagRepo *repository.TagRepository,
	mpRepo *repository.MandatoryPaymentRepository,
	balRepo *repository.BalanceRepository,
	peRepo *repository.PlannedExpenseRepository,
) *AnalyticsService {
	return &AnalyticsService{repo: repo, tagRepo: tagRepo, mpRepo: mpRepo, balRepo: balRepo, peRepo: peRepo}
}

func (s *AnalyticsService) GetExpensesCalendar(ctx context.Context, level string, year, month int, projectID int64) (domain.CalendarResult, error) {
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
		return s.getYearLevel(ctx, year, now, projectID)
	}
	return s.getMonthLevel(ctx, year, month, now, projectID)
}

func (s *AnalyticsService) getYearLevel(ctx context.Context, year int, now time.Time, projectID int64) (domain.CalendarResult, error) {
	sums, err := s.repo.SumByMonthInYear(ctx, year, projectID)
	if err != nil {
		return domain.CalendarResult{}, err
	}
	byMonth := make(map[int]int64, len(sums))
	for _, ms := range sums {
		byMonth[ms.Month] = ms.Amount
	}

	periodStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	hasPrevious, err := s.repo.ExistsExpenseBefore(ctx, periodStart, projectID)
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

func (s *AnalyticsService) getMonthLevel(ctx context.Context, year, month int, now time.Time, projectID int64) (domain.CalendarResult, error) {
	sums, err := s.repo.SumByDayInMonth(ctx, year, month, projectID)
	if err != nil {
		return domain.CalendarResult{}, err
	}
	byDay := make(map[int]int64, len(sums))
	for _, ds := range sums {
		byDay[ds.Day] = ds.Amount
	}

	transactions, err := s.repo.ListExpenseTransactionsInMonth(ctx, year, month, projectID)
	if err != nil {
		return domain.CalendarResult{}, err
	}
	byDayTxs := make(map[int][]domain.TransactionBrief)
	for _, t := range transactions {
		byDayTxs[t.Day] = append(byDayTxs[t.Day], t)
	}

	rootOf, err := s.tagRootMap(ctx, projectID)
	if err != nil {
		return domain.CalendarResult{}, err
	}

	periodStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	hasPrevious, err := s.repo.ExistsExpenseBefore(ctx, periodStart, projectID)
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

func (s *AnalyticsService) tagRootMap(ctx context.Context, projectID int64) (map[int64]tagInfo, error) {
	tags, err := s.tagRepo.List(ctx, projectID)
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

// GetFreedomIndex возвращает долю «простых» (по желанию) расходов от общих
// расходов по месяцам за последние months месяцев (пропуски заполняются
// нулями, share=nil для месяцев без расходов).
func (s *AnalyticsService) GetFreedomIndex(ctx context.Context, months int, projectID int64) ([]dto.FreedomIndexPointResponse, error) {
	if months <= 0 {
		months = 12
	}
	now := time.Now().UTC()
	since := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -(months - 1), 0)

	points, err := s.repo.FreedomIndexByMonth(ctx, since, projectID)
	if err != nil {
		return nil, err
	}
	return buildFreedomIndexResponse(points, since, months), nil
}

// buildFreedomIndexResponse заполняет пропуски в месяцах без расходов
// (repo возвращает точки только для месяцев, где были расходы) и вычисляет
// долю «простых» трат (nil, если расходов за месяц не было).
func buildFreedomIndexResponse(points []domain.FreedomIndexPoint, since time.Time, months int) []dto.FreedomIndexPointResponse {
	byMonth := make(map[string]domain.FreedomIndexPoint, len(points))
	for _, p := range points {
		byMonth[p.Month.Format("2006-01")] = p
	}

	resp := make([]dto.FreedomIndexPointResponse, 0, months)
	for i := 0; i < months; i++ {
		m := since.AddDate(0, i, 0)
		key := m.Format("2006-01")
		item := dto.FreedomIndexPointResponse{Month: key}
		if p, ok := byMonth[key]; ok {
			item.SimpleAmount = p.SimpleAmount
			item.TotalExpense = p.TotalExpense
		}
		if item.TotalExpense > 0 {
			share := float64(item.SimpleAmount) / float64(item.TotalExpense)
			item.Share = &share
		}
		resp = append(resp, item)
	}
	return resp
}

// GetFixedCostRatio возвращает долю дохода месяца, уходящую на обязательные
// регулярные платежи.
func (s *AnalyticsService) GetFixedCostRatio(ctx context.Context, year, month int, projectID int64) (dto.FixedCostRatioResponse, error) {
	income, err := s.repo.MonthlyIncome(ctx, year, month, projectID)
	if err != nil {
		return dto.FixedCostRatioResponse{}, err
	}
	payments, err := s.mpRepo.List(ctx, projectID)
	if err != nil {
		return dto.FixedCostRatioResponse{}, err
	}
	var mandatoryTotal int64
	for _, p := range payments {
		mandatoryTotal += p.Amount
	}

	resp := dto.FixedCostRatioResponse{Income: income, MandatoryTotal: mandatoryTotal}
	if income > 0 {
		ratio := float64(mandatoryTotal) / float64(income)
		resp.Ratio = &ratio
	}
	return resp, nil
}

// monthOrYearRange вычисляет полуоткрытый интервал [from, to) для level=day
// (месяц) или level=month (год), аналогично границам в GetExpensesCalendar.
func monthOrYearRange(level string, year, month int) (time.Time, time.Time, error) {
	switch level {
	case string(domain.CalendarLevelDay):
		if month < 1 || month > 12 {
			return time.Time{}, time.Time{}, &apperrors.ValidationError{
				Fields: map[string]string{"month": "required and must be 1..12 for level=day"}, Message: "validation failed",
			}
		}
		from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		return from, from.AddDate(0, 1, 0), nil
	case string(domain.CalendarLevelMonth):
		from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
		return from, from.AddDate(1, 0, 0), nil
	default:
		return time.Time{}, time.Time{}, &apperrors.ValidationError{
			Fields: map[string]string{"level": "must be day or month"}, Message: "validation failed",
		}
	}
}

// GetIncomeExpenseSankey строит узлы и потоки Sankey-диаграммы: источники
// дохода → «Доходы» → категории расходов верхнего уровня → подкатегории →
// «Остаток» (только если доход за период превысил расход).
func (s *AnalyticsService) GetIncomeExpenseSankey(ctx context.Context, level string, year, month int, projectID int64) (domain.SankeyResult, error) {
	from, to, err := monthOrYearRange(level, year, month)
	if err != nil {
		return domain.SankeyResult{}, err
	}

	incomeSums, err := s.repo.SumByTagInPeriod(ctx, from, to, "income", projectID)
	if err != nil {
		return domain.SankeyResult{}, err
	}
	expenseSums, err := s.repo.SumByTagInPeriod(ctx, from, to, "expense", projectID)
	if err != nil {
		return domain.SankeyResult{}, err
	}
	rootOf, err := s.tagRootMap(ctx, projectID)
	if err != nil {
		return domain.SankeyResult{}, err
	}
	tags, err := s.tagRepo.List(ctx, projectID)
	if err != nil {
		return domain.SankeyResult{}, err
	}
	nameByID := make(map[int64]string, len(tags))
	for _, t := range tags {
		nameByID[t.ID] = t.Name
	}

	return buildSankeyResult(incomeSums, expenseSums, rootOf, nameByID), nil
}

// buildSankeyResult строит узлы и потоки из уже загруженных сумм по тегам:
// источники дохода → «Доходы» → категории расходов верхнего уровня →
// подкатегории → «Остаток» (только если доход превысил расход за период).
func buildSankeyResult(incomeSums, expenseSums []domain.TagAmount, rootOf map[int64]tagInfo, nameByID map[int64]string) domain.SankeyResult {
	const hubID = "income_hub"
	result := domain.SankeyResult{}
	nodeSeen := map[string]bool{}
	addNode := func(id, label string) {
		if !nodeSeen[id] {
			nodeSeen[id] = true
			result.Nodes = append(result.Nodes, domain.SankeyNode{ID: id, Label: label})
		}
	}

	var totalIncome, totalExpense int64
	for _, inc := range incomeSums {
		if inc.Amount <= 0 {
			continue
		}
		totalIncome += inc.Amount
		info, ok := rootOf[inc.TagID]
		rootID := inc.TagID
		label := nameByID[inc.TagID]
		if ok {
			rootID = info.rootID
			label = info.name
		}
		nodeID := "income:" + strconv.FormatInt(rootID, 10)
		addNode(nodeID, label)
		addNode(hubID, "Доходы")
		result.Flows = append(result.Flows, domain.SankeyFlow{From: nodeID, To: hubID, Amount: inc.Amount})
	}

	// Свернуть расходы: сумма по корневому тегу (включая собственные
	// транзакции и все дочерние), плюс отдельная сумма по каждому
	// непосредственно дочернему тегу для потока root -> child.
	rootExpenseTotal := map[int64]int64{}
	rootLabel := map[int64]string{}
	childExpense := map[int64]map[int64]int64{} // rootID -> childTagID -> amount
	for _, exp := range expenseSums {
		if exp.Amount <= 0 {
			continue
		}
		totalExpense += exp.Amount
		info, ok := rootOf[exp.TagID]
		rootID := exp.TagID
		label := nameByID[exp.TagID]
		if ok {
			rootID = info.rootID
			label = info.name
		}
		rootExpenseTotal[rootID] += exp.Amount
		rootLabel[rootID] = label
		if exp.TagID != rootID {
			if childExpense[rootID] == nil {
				childExpense[rootID] = map[int64]int64{}
			}
			childExpense[rootID][exp.TagID] += exp.Amount
		}
	}
	if totalIncome > 0 {
		addNode(hubID, "Доходы")
	}
	for rootID, total := range rootExpenseTotal {
		nodeID := "expense:" + strconv.FormatInt(rootID, 10)
		addNode(nodeID, rootLabel[rootID])
		if totalIncome > 0 {
			result.Flows = append(result.Flows, domain.SankeyFlow{From: hubID, To: nodeID, Amount: total})
		}
		for childID, amount := range childExpense[rootID] {
			childNodeID := "expense:" + strconv.FormatInt(childID, 10)
			addNode(childNodeID, nameByID[childID])
			result.Flows = append(result.Flows, domain.SankeyFlow{From: nodeID, To: childNodeID, Amount: amount})
		}
	}

	if totalIncome > totalExpense {
		remainder := totalIncome - totalExpense
		addNode("remainder", "Остаток")
		result.Flows = append(result.Flows, domain.SankeyFlow{From: hubID, To: "remainder", Amount: remainder})
	} else if totalExpense > totalIncome {
		deficit := totalExpense - totalIncome
		result.DeficitAmount = &deficit
	}

	return result
}

// dailySavingsRate — среднедневная ставка накопления за последние 90 дней:
// доход минус «простые» (impulse) расходы минус усреднённая ставка
// обязательных платежей. Используется и прогнозом баланса (для не-mandatory
// части), и прогнозом накопления на цели вишлиста.
func (s *AnalyticsService) dailySavingsRate(ctx context.Context, projectID int64) (float64, error) {
	const trendDays = 90
	income, discretionary, err := s.repo.DailyIncomeExpenseTrend(ctx, trendDays, projectID)
	if err != nil {
		return 0, err
	}
	mandatory, err := s.repo.MandatoryDailyAverage(ctx, trendDays, projectID)
	if err != nil {
		return 0, err
	}
	return float64(income-discretionary-mandatory) / float64(trendDays), nil
}

// projectMandatoryOccurrences раскладывает регулярные платежи на конкретные
// даты наступления в горизонте [from, from+days], используя advanceDate —
// платёж может наступить несколько раз в горизонте (например, ежедневный).
func projectMandatoryOccurrences(payments []domain.MandatoryPayment, from time.Time, days int) map[string]int64 {
	horizonEnd := from.AddDate(0, 0, days)
	occurrences := map[string]int64{}
	for _, p := range payments {
		date := p.NextPaymentDate
		for i := 0; i < 1000 && !date.After(horizonEnd); i++ {
			if !date.Before(from) {
				occurrences[date.Format("2006-01-02")] += p.Amount
			}
			date = advanceDate(date, p.Recurrence)
		}
	}
	return occurrences
}

// GetBalanceForecast строит факт за последние 90 дней и прогноз на days дней
// вперёд: текущий баланс + дневной тренд дохода/расхода (за вычетом
// обязательных платежей, которые проецируются явно по датам) - регулярные
// платежи, наступающие в горизонте.
func (s *AnalyticsService) GetBalanceForecast(ctx context.Context, days int, projectID int64) (dto.BalanceForecastResponse, error) {
	if days != 30 && days != 60 && days != 90 {
		days = 30
	}
	const historyDays = 90

	snap, err := s.balRepo.GetSnapshot(ctx, projectID)
	if err != nil {
		return dto.BalanceForecastResponse{}, err
	}
	historyPoints, err := s.repo.HistoricalDailyBalance(ctx, historyDays, snap.InitialAmount, projectID)
	if err != nil {
		return dto.BalanceForecastResponse{}, err
	}
	income, discretionary, err := s.repo.DailyIncomeExpenseTrend(ctx, historyDays, projectID)
	if err != nil {
		return dto.BalanceForecastResponse{}, err
	}
	dailyRate := float64(income-discretionary) / float64(historyDays)

	payments, err := s.mpRepo.List(ctx, projectID)
	if err != nil {
		return dto.BalanceForecastResponse{}, err
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	occurrences := projectMandatoryOccurrences(payments, today, days)

	history := make([]dto.BalancePointResponse, 0, len(historyPoints))
	for _, p := range historyPoints {
		history = append(history, dto.BalancePointResponse{Date: p.Date.Format("2006-01-02"), Balance: p.Balance})
	}

	balance := float64(snap.CurrentBalance)
	forecast := make([]dto.BalancePointResponse, 0, days)
	var zeroCrossing *string
	for i := 1; i <= days; i++ {
		date := today.AddDate(0, 0, i)
		balance += dailyRate
		if amt, ok := occurrences[date.Format("2006-01-02")]; ok {
			balance -= float64(amt)
		}
		rounded := int64(balance)
		if rounded < 0 && zeroCrossing == nil {
			d := date.Format("2006-01-02")
			zeroCrossing = &d
		}
		forecast = append(forecast, dto.BalancePointResponse{Date: date.Format("2006-01-02"), Balance: rounded})
	}

	return dto.BalanceForecastResponse{
		CurrentBalance:   snap.CurrentBalance,
		History:          history,
		Forecast:         forecast,
		ZeroCrossingDate: zeroCrossing,
	}, nil
}

// GetSavingsGoals прогнозирует дату, когда баланс дорастёт до цены каждого
// активного товара вишлиста с указанной ценой, по dailySavingsRate.
func (s *AnalyticsService) GetSavingsGoals(ctx context.Context, projectID int64) ([]dto.SavingsGoalResponse, error) {
	items, err := s.peRepo.ListByStatus(ctx, domain.PlannedExpenseStatusActive, projectID)
	if err != nil {
		return nil, err
	}
	snap, err := s.balRepo.GetSnapshot(ctx, projectID)
	if err != nil {
		return nil, err
	}
	rate, err := s.dailySavingsRate(ctx, projectID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	resp := make([]dto.SavingsGoalResponse, 0, len(items))
	for _, item := range items {
		if item.CostKopecks == nil {
			continue
		}
		resp = append(resp, projectSavingsGoal(item, snap.CurrentBalance, rate, now))
	}
	return resp, nil
}

// projectSavingsGoal вычисляет прогноз для одного товара вишлиста: уже
// доступно, дата достижения (при rate > 0), либо неизвестно (rate <= 0).
// Вызывающий код гарантирует item.CostKopecks != nil.
func projectSavingsGoal(item domain.PlannedExpense, currentBalance int64, rate float64, now time.Time) dto.SavingsGoalResponse {
	cost := *item.CostKopecks
	g := dto.SavingsGoalResponse{PlannedExpenseID: item.ID, Title: item.Title, Cost: cost}
	if cost <= currentBalance {
		g.AlreadyAffordable = true
	} else if rate > 0 {
		daysNeeded := int(math.Ceil(float64(cost-currentBalance) / rate))
		date := now.AddDate(0, 0, daysNeeded).Format("2006-01-02")
		g.ProjectedDate = &date
	}
	return g
}

// GetSpendingMoodCalendar возвращает долю «простых» расходов по дням месяца
// для календаря, окрашенного по «настроению» трат.
func (s *AnalyticsService) GetSpendingMoodCalendar(ctx context.Context, year, month int, projectID int64) ([]dto.MoodDayResponse, error) {
	points, err := s.repo.SpendingMoodByDay(ctx, year, month, projectID)
	if err != nil {
		return nil, err
	}
	byDay := make(map[int]domain.DailyMoodPoint, len(points))
	for _, p := range points {
		byDay[p.Day] = p
	}

	periodStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	daysInMonth := periodStart.AddDate(0, 1, -1).Day()

	resp := make([]dto.MoodDayResponse, 0, daysInMonth)
	for d := 1; d <= daysInMonth; d++ {
		item := dto.MoodDayResponse{Day: d}
		if p, ok := byDay[d]; ok {
			item.SimpleAmount = p.SimpleAmount
			item.TotalExpense = p.TotalExpense
		}
		if item.TotalExpense > 0 {
			share := float64(item.SimpleAmount) / float64(item.TotalExpense)
			item.Share = &share
		}
		resp = append(resp, item)
	}
	return resp, nil
}

const (
	insightTypeTopCategoryGrowth = "top_category_growth"
	insightTypeBiggestTx         = "biggest_transaction"
	insightTypeNewTag            = "new_tag"
)

// insightTopCategoryGrowth — тег с ростом расходов >20% и >1000₽ относительно
// среднего за предыдущие 3 месяца.
// biggestSignificantGrowth находит тег с наибольшим ростом расходов
// относительно среднего за предыдущие месяцы, среди тех, где рост >20% и
// >1000₽ (в копейках — 1000_00). Возвращает nil, если ни один тег не проходит порог.
func biggestSignificantGrowth(deltas []domain.TagDelta) (*domain.TagDelta, float64) {
	var best *domain.TagDelta
	var bestGrowthPct float64
	for i, d := range deltas {
		if d.AverageAmount <= 0 {
			continue
		}
		deltaAmount := float64(d.CurrentAmount) - d.AverageAmount
		growthPct := deltaAmount / d.AverageAmount
		if growthPct > 0.2 && deltaAmount > 1000_00 && growthPct > bestGrowthPct {
			bestGrowthPct = growthPct
			best = &deltas[i]
		}
	}
	return best, bestGrowthPct
}

func (s *AnalyticsService) insightTopCategoryGrowth(ctx context.Context, projectID int64) (*domain.Insight, error) {
	deltas, err := s.repo.TagDeltaVsAverage(ctx, projectID)
	if err != nil {
		return nil, err
	}
	tags, err := s.tagRepo.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	nameByID := make(map[int64]string, len(tags))
	for _, t := range tags {
		nameByID[t.ID] = t.Name
	}

	best, bestGrowthPct := biggestSignificantGrowth(deltas)
	if best == nil {
		return nil, nil
	}
	name := nameByID[best.TagID]
	text := fmt.Sprintf("В этом месяце расходы на «%s» на %.0f%% больше, чем в среднем за последние 3 месяца", name, bestGrowthPct*100)
	return &domain.Insight{Type: insightTypeTopCategoryGrowth, Text: text}, nil
}

// insightBiggestTransaction — самая крупная расходная транзакция месяца.
func (s *AnalyticsService) insightBiggestTransaction(ctx context.Context, projectID int64) (*domain.Insight, error) {
	tx, err := s.repo.BiggestExpenseThisMonth(ctx, projectID)
	if err != nil || tx == nil {
		return nil, err
	}
	text := fmt.Sprintf("Самая крупная покупка месяца — «%s», %.0f ₽", tx.Title, float64(tx.Amount)/100)
	return &domain.Insight{Type: insightTypeBiggestTx, Text: text}, nil
}

// insightNewTag — тег, впервые встретившийся в этом месяце.
func (s *AnalyticsService) insightNewTag(ctx context.Context, projectID int64) (*domain.Insight, error) {
	newTagIDs, err := s.repo.NewTagsThisMonth(ctx, projectID)
	if err != nil || len(newTagIDs) == 0 {
		return nil, err
	}
	tags, err := s.tagRepo.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	nameByID := make(map[int64]string, len(tags))
	for _, t := range tags {
		nameByID[t.ID] = t.Name
	}
	text := fmt.Sprintf("Впервые в этом месяце появились расходы по категории «%s»", nameByID[newTagIDs[0]])
	return &domain.Insight{Type: insightTypeNewTag, Text: text}, nil
}

// GetInsights собирает результаты всех правил инсайтов (0-3 элемента).
func (s *AnalyticsService) GetInsights(ctx context.Context, projectID int64) ([]dto.InsightResponse, error) {
	rules := []func(context.Context, int64) (*domain.Insight, error){
		s.insightTopCategoryGrowth,
		s.insightBiggestTransaction,
		s.insightNewTag,
	}
	resp := make([]dto.InsightResponse, 0, len(rules))
	for _, rule := range rules {
		insight, err := rule(ctx, projectID)
		if err != nil {
			return nil, err
		}
		if insight != nil {
			resp = append(resp, dto.InsightResponse{Type: insight.Type, Text: insight.Text})
		}
	}
	return resp, nil
}

// GetPeriodComparison сравнивает топ-5 тегов текущего месяца с предыдущим
// месяцем (mode=prev_month) или тем же месяцем год назад (mode=prev_year).
func (s *AnalyticsService) GetPeriodComparison(ctx context.Context, mode string, projectID int64) ([]dto.PeriodComparisonResponse, error) {
	now := time.Now().UTC()
	currentFrom := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	currentTo := currentFrom.AddDate(0, 1, 0)

	var comparisonFrom time.Time
	switch mode {
	case "prev_year":
		comparisonFrom = currentFrom.AddDate(-1, 0, 0)
	default:
		comparisonFrom = currentFrom.AddDate(0, -1, 0)
	}
	comparisonTo := comparisonFrom.AddDate(0, 1, 0)

	amounts, err := s.repo.SumByTagForTwoPeriods(ctx, currentFrom, currentTo, comparisonFrom, comparisonTo, projectID)
	if err != nil {
		return nil, err
	}
	tags, err := s.tagRepo.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	nameByID := make(map[int64]string, len(tags))
	for _, t := range tags {
		nameByID[t.ID] = t.Name
	}

	amounts = topNByMax(amounts, 5)

	resp := make([]dto.PeriodComparisonResponse, 0, len(amounts))
	for _, a := range amounts {
		resp = append(resp, dto.PeriodComparisonResponse{
			TagID:            a.TagID,
			Name:             nameByID[a.TagID],
			CurrentAmount:    a.CurrentAmount,
			ComparisonAmount: a.ComparisonAmount,
		})
	}
	return resp, nil
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// topNByMax сортирует по убыванию max(CurrentAmount, ComparisonAmount) и
// возвращает первые n элементов.
func topNByMax(amounts []domain.PeriodComparisonAmounts, n int) []domain.PeriodComparisonAmounts {
	sort.Slice(amounts, func(i, j int) bool {
		return max64(amounts[i].CurrentAmount, amounts[i].ComparisonAmount) > max64(amounts[j].CurrentAmount, amounts[j].ComparisonAmount)
	})
	if len(amounts) > n {
		amounts = amounts[:n]
	}
	return amounts
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
