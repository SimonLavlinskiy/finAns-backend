package repository

import (
	"context"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsRepository struct {
	pool *pgxpool.Pool
}

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

func (r *AnalyticsRepository) SumByDayInMonth(ctx context.Context, year, month int) ([]domain.DailySum, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT EXTRACT(DAY FROM date)::int AS day, SUM(amount)
		FROM transactions
		WHERE category = 'expense'::transaction_category
		  AND date >= make_date($1, $2, 1)
		  AND date < (make_date($1, $2, 1) + INTERVAL '1 month')
		GROUP BY day`, year, month)
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

func (r *AnalyticsRepository) SumByMonthInYear(ctx context.Context, year int) ([]domain.MonthlySum, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT EXTRACT(MONTH FROM date)::int AS month, SUM(amount)
		FROM transactions
		WHERE category = 'expense'::transaction_category
		  AND date >= make_date($1, 1, 1)
		  AND date < make_date($1 + 1, 1, 1)
		GROUP BY month`, year)
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

func (r *AnalyticsRepository) ListExpenseTransactionsInMonth(ctx context.Context, year, month int) ([]domain.TransactionBrief, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, amount, EXTRACT(DAY FROM date)::int AS day, tag_id
		FROM transactions
		WHERE category = 'expense'::transaction_category
		  AND date >= make_date($1, $2, 1)
		  AND date < (make_date($1, $2, 1) + INTERVAL '1 month')
		ORDER BY date, id`, year, month)
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

func (r *AnalyticsRepository) ExistsExpenseBefore(ctx context.Context, before time.Time) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM transactions
			WHERE category = 'expense'::transaction_category AND date < $1
		)`, before).Scan(&exists)
	return exists, err
}
