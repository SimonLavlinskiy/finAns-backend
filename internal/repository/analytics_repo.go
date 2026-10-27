package repository

import (
	"context"
	"errors"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsRepository struct {
	pool *pgxpool.Pool
}

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

func (r *AnalyticsRepository) SumByDayInMonth(ctx context.Context, year, month int, projectID int64) ([]domain.DailySum, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT EXTRACT(DAY FROM date)::int AS day, SUM(amount)
		FROM transactions
		WHERE category = 'expense'::transaction_category
		  AND project_id = $3
		  AND date >= make_date($1, $2, 1)
		  AND date < (make_date($1, $2, 1) + INTERVAL '1 month')
		GROUP BY day`, year, month, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sums []domain.DailySum
	for rows.Next() {
		var s domain.DailySum
		if err := rows.Scan(&s.Day, &s.Amount); err != nil {
			return nil, err
		}
		sums = append(sums, s)
	}
	return sums, rows.Err()
}

func (r *AnalyticsRepository) SumByMonthInYear(ctx context.Context, year int, projectID int64) ([]domain.MonthlySum, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT EXTRACT(MONTH FROM date)::int AS month, SUM(amount)
		FROM transactions
		WHERE category = 'expense'::transaction_category
		  AND project_id = $2
		  AND date >= make_date($1, 1, 1)
		  AND date < make_date($1 + 1, 1, 1)
		GROUP BY month`, year, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sums []domain.MonthlySum
	for rows.Next() {
		var s domain.MonthlySum
		if err := rows.Scan(&s.Month, &s.Amount); err != nil {
			return nil, err
		}
		sums = append(sums, s)
	}
	return sums, rows.Err()
}

func (r *AnalyticsRepository) ListExpenseTransactionsInMonth(ctx context.Context, year, month int, projectID int64) ([]domain.TransactionBrief, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, amount, EXTRACT(DAY FROM date)::int AS day, tag_id
		FROM transactions
		WHERE category = 'expense'::transaction_category
		  AND project_id = $3
		  AND date >= make_date($1, $2, 1)
		  AND date < (make_date($1, $2, 1) + INTERVAL '1 month')
		ORDER BY date, id`, year, month, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.TransactionBrief
	for rows.Next() {
		var t domain.TransactionBrief
		if err := rows.Scan(&t.ID, &t.Title, &t.Amount, &t.Day, &t.TagID); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (r *AnalyticsRepository) ExistsExpenseBefore(ctx context.Context, before time.Time, projectID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM transactions
			WHERE category = 'expense'::transaction_category AND project_id = $2 AND date < $1
		)`, before, projectID).Scan(&exists)
	return exists, err
}

// FreedomIndexByMonth возвращает сумму «простых» и общих расходов по месяцам,
// начиная с since. Месяцы без расходов не попадают в результат — вызывающий
// код (сервис) заполняет пропуски.
func (r *AnalyticsRepository) FreedomIndexByMonth(ctx context.Context, since time.Time, projectID int64) ([]domain.FreedomIndexPoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT date_trunc('month', date) AS month,
		       COALESCE(SUM(amount) FILTER (WHERE specificity = 'simple'), 0) AS simple_sum,
		       COALESCE(SUM(amount), 0) AS total_expense
		FROM transactions
		WHERE category = 'expense'::transaction_category AND project_id = $1 AND date >= $2
		GROUP BY 1 ORDER BY 1`, projectID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []domain.FreedomIndexPoint
	for rows.Next() {
		var p domain.FreedomIndexPoint
		if err := rows.Scan(&p.Month, &p.SimpleAmount, &p.TotalExpense); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// SpendingMoodByDay — сумма «простых» и общих расходов по дням месяца (для
// календаря, окрашенного по доле impulse-трат за день).
func (r *AnalyticsRepository) SpendingMoodByDay(ctx context.Context, year, month int, projectID int64) ([]domain.DailyMoodPoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT EXTRACT(DAY FROM date)::int AS day,
		       COALESCE(SUM(amount) FILTER (WHERE specificity = 'simple'), 0) AS simple_sum,
		       COALESCE(SUM(amount), 0) AS total_expense
		FROM transactions
		WHERE category = 'expense'::transaction_category
		  AND project_id = $3
		  AND date >= make_date($1, $2, 1)
		  AND date < (make_date($1, $2, 1) + INTERVAL '1 month')
		GROUP BY day`, year, month, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []domain.DailyMoodPoint
	for rows.Next() {
		var p domain.DailyMoodPoint
		if err := rows.Scan(&p.Day, &p.SimpleAmount, &p.TotalExpense); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// MonthlyIncome — сумма дохода за один календарный месяц.
func (r *AnalyticsRepository) MonthlyIncome(ctx context.Context, year, month int, projectID int64) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM transactions
		WHERE category = 'income'::transaction_category
		  AND project_id = $3
		  AND date >= make_date($1, $2, 1)
		  AND date < (make_date($1, $2, 1) + INTERVAL '1 month')`, year, month, projectID).Scan(&total)
	return total, err
}

// SumByTagInPeriod — сумма транзакций по тегу за период [from, to) для заданной категории.
func (r *AnalyticsRepository) SumByTagInPeriod(ctx context.Context, from, to time.Time, category string, projectID int64) ([]domain.TagAmount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT tag_id, SUM(amount) FROM transactions
		WHERE category = $1::transaction_category AND project_id = $2 AND date >= $3 AND date < $4
		GROUP BY tag_id`, category, projectID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.TagAmount
	for rows.Next() {
		var t domain.TagAmount
		if err := rows.Scan(&t.TagID, &t.Amount); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

// DailyIncomeExpenseTrend — суммы дохода и «простых» расходов за последние sinceDays дней
// (для расчёта среднедневной ставки трендом делением на sinceDays).
func (r *AnalyticsRepository) DailyIncomeExpenseTrend(ctx context.Context, sinceDays int, projectID int64) (income, discretionaryExpense int64, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE category = 'income'), 0),
			COALESCE(SUM(amount) FILTER (WHERE category = 'expense' AND specificity = 'simple'), 0)
		FROM transactions
		WHERE project_id = $1 AND date >= (CURRENT_DATE - $2::int)`, projectID, sinceDays).Scan(&income, &discretionaryExpense)
	return income, discretionaryExpense, err
}

// MandatoryDailyRate — средняя дневная сумма обязательных регулярных платежей,
// оценённая по уже совершённым (specificity='required') расходным транзакциям
// за последние sinceDays дней.
func (r *AnalyticsRepository) MandatoryDailyAverage(ctx context.Context, sinceDays int, projectID int64) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM transactions
		WHERE category = 'expense'::transaction_category AND specificity = 'required'
		  AND project_id = $1 AND date >= (CURRENT_DATE - $2::int)`, projectID, sinceDays).Scan(&total)
	return total, err
}

// HistoricalDailyBalance — фактический баланс по дням за последние sinceDays дней,
// как initialAmount плюс кумулятивная сумма всех транзакций до этого дня включительно.
func (r *AnalyticsRepository) HistoricalDailyBalance(ctx context.Context, sinceDays int, initialAmount int64, projectID int64) ([]domain.BalanceForecastPoint, error) {
	rows, err := r.pool.Query(ctx, `
		WITH daily AS (
			SELECT date::date AS day,
			       SUM(CASE WHEN category = 'income' THEN amount ELSE -amount END) AS delta
			FROM transactions
			WHERE project_id = $1
			GROUP BY date
		),
		running AS (
			SELECT day, SUM(delta) OVER (ORDER BY day) AS cumulative
			FROM daily
		)
		SELECT day, $2::bigint + cumulative AS balance
		FROM running
		WHERE day >= (CURRENT_DATE - $3::int)
		ORDER BY day`, projectID, initialAmount, sinceDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []domain.BalanceForecastPoint
	for rows.Next() {
		var p domain.BalanceForecastPoint
		if err := rows.Scan(&p.Date, &p.Balance); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// TagDeltaVsAverage — сумма по тегу за текущий месяц и среднее за предыдущие 3 месяца.
func (r *AnalyticsRepository) TagDeltaVsAverage(ctx context.Context, projectID int64) ([]domain.TagDelta, error) {
	rows, err := r.pool.Query(ctx, `
		WITH current_month AS (
			SELECT tag_id, SUM(amount) AS amount
			FROM transactions
			WHERE category = 'expense'::transaction_category AND project_id = $1
			  AND date >= date_trunc('month', CURRENT_DATE)
			GROUP BY tag_id
		),
		previous_months AS (
			SELECT tag_id, SUM(amount) / 3.0 AS avg_amount
			FROM transactions
			WHERE category = 'expense'::transaction_category AND project_id = $1
			  AND date >= date_trunc('month', CURRENT_DATE) - INTERVAL '3 months'
			  AND date < date_trunc('month', CURRENT_DATE)
			GROUP BY tag_id
		)
		SELECT c.tag_id, c.amount, COALESCE(p.avg_amount, 0)
		FROM current_month c
		LEFT JOIN previous_months p ON p.tag_id = c.tag_id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deltas []domain.TagDelta
	for rows.Next() {
		var d domain.TagDelta
		if err := rows.Scan(&d.TagID, &d.CurrentAmount, &d.AverageAmount); err != nil {
			return nil, err
		}
		deltas = append(deltas, d)
	}
	return deltas, rows.Err()
}

// BiggestExpenseThisMonth — крупнейшая расходная транзакция текущего месяца (nil, если расходов не было).
func (r *AnalyticsRepository) BiggestExpenseThisMonth(ctx context.Context, projectID int64) (*domain.TransactionBrief, error) {
	var t domain.TransactionBrief
	err := r.pool.QueryRow(ctx, `
		SELECT id, title, amount, EXTRACT(DAY FROM date)::int, tag_id
		FROM transactions
		WHERE category = 'expense'::transaction_category AND project_id = $1
		  AND date >= date_trunc('month', CURRENT_DATE)
		ORDER BY amount DESC LIMIT 1`, projectID).Scan(&t.ID, &t.Title, &t.Amount, &t.Day, &t.TagID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// NewTagsThisMonth — теги, по которым в текущем месяце впервые появились расходы
// (не было расходов по ним ни разу до начала месяца).
func (r *AnalyticsRepository) NewTagsThisMonth(ctx context.Context, projectID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT tag_id FROM transactions cur
		WHERE cur.category = 'expense'::transaction_category AND cur.project_id = $1
		  AND cur.date >= date_trunc('month', CURRENT_DATE)
		  AND NOT EXISTS (
		    SELECT 1 FROM transactions prev
		    WHERE prev.category = 'expense'::transaction_category AND prev.project_id = $1
		      AND prev.tag_id = cur.tag_id
		      AND prev.date < date_trunc('month', CURRENT_DATE)
		  )`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SumByTagForTwoPeriods — суммы по тегу за два периода одним запросом.
func (r *AnalyticsRepository) SumByTagForTwoPeriods(ctx context.Context, currentFrom, currentTo, comparisonFrom, comparisonTo time.Time, projectID int64) ([]domain.PeriodComparisonAmounts, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT tag_id,
		       COALESCE(SUM(amount) FILTER (WHERE date >= $2 AND date < $3), 0) AS current_amount,
		       COALESCE(SUM(amount) FILTER (WHERE date >= $4 AND date < $5), 0) AS comparison_amount
		FROM transactions
		WHERE category = 'expense'::transaction_category AND project_id = $1
		  AND ((date >= $2 AND date < $3) OR (date >= $4 AND date < $5))
		GROUP BY tag_id`, projectID, currentFrom, currentTo, comparisonFrom, comparisonTo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.PeriodComparisonAmounts
	for rows.Next() {
		var p domain.PeriodComparisonAmounts
		if err := rows.Scan(&p.TagID, &p.CurrentAmount, &p.ComparisonAmount); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

// SumByTagAndCategoryInPeriod — суммы операций в [from, to) по тегу и типу операции
// с признаком «все операции обязательные».
func (r *AnalyticsRepository) SumByTagAndCategoryInPeriod(ctx context.Context, from, to time.Time, projectID int64) ([]domain.TagCategoryAmount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT tag_id, category::text, SUM(amount), BOOL_AND(specificity = 'required'::transaction_specificity)
		FROM transactions
		WHERE project_id = $1 AND date >= $2 AND date < $3
		GROUP BY tag_id, category`, projectID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.TagCategoryAmount
	for rows.Next() {
		var a domain.TagCategoryAmount
		if err := rows.Scan(&a.TagID, &a.Category, &a.Amount, &a.AllMandatory); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// IncomeExpenseByMonthInYear — доходы и расходы по месяцам года (только месяцы с операциями).
func (r *AnalyticsRepository) IncomeExpenseByMonthInYear(ctx context.Context, year int, projectID int64) ([]domain.MonthIncomeExpense, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT EXTRACT(MONTH FROM date)::int AS month,
		       COALESCE(SUM(amount) FILTER (WHERE category = 'income'::transaction_category), 0),
		       COALESCE(SUM(amount) FILTER (WHERE category = 'expense'::transaction_category), 0)
		FROM transactions
		WHERE project_id = $2
		  AND date >= make_date($1, 1, 1)
		  AND date < make_date($1 + 1, 1, 1)
		GROUP BY month
		ORDER BY month`, year, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.MonthIncomeExpense
	for rows.Next() {
		var m domain.MonthIncomeExpense
		if err := rows.Scan(&m.Month, &m.Income, &m.Expense); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// TransactionYears — годы, в которых есть операции, по возрастанию.
func (r *AnalyticsRepository) TransactionYears(ctx context.Context, projectID int64) ([]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT EXTRACT(YEAR FROM date)::int AS y
		FROM transactions WHERE project_id = $1
		ORDER BY y`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var y int
		if err := rows.Scan(&y); err != nil {
			return nil, err
		}
		years = append(years, y)
	}
	return years, rows.Err()
}

// FirstTransactionDate — дата самой ранней операции проекта (nil, если операций нет).
func (r *AnalyticsRepository) FirstTransactionDate(ctx context.Context, projectID int64) (*time.Time, error) {
	var d *time.Time
	err := r.pool.QueryRow(ctx, `SELECT MIN(date) FROM transactions WHERE project_id = $1`, projectID).Scan(&d)
	return d, err
}
