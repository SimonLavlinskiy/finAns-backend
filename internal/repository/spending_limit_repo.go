package repository

import (
	"context"
	"errors"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SpendingLimitRepository struct {
	pool *pgxpool.Pool
}

func NewSpendingLimitRepository(pool *pgxpool.Pool) *SpendingLimitRepository {
	return &SpendingLimitRepository{pool: pool}
}

const slColumns = `id, tag_id, amount, period_type::text, period_start, period_end, project_id, created_at`

func scanSpendingLimit(row pgx.Row) (domain.SpendingLimit, error) {
	var l domain.SpendingLimit
	err := row.Scan(&l.ID, &l.TagID, &l.Amount, &l.PeriodType, &l.PeriodStart, &l.PeriodEnd, &l.ProjectID, &l.CreatedAt)
	return l, err
}

func (r *SpendingLimitRepository) List(ctx context.Context, projectID int64) ([]domain.SpendingLimit, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+slColumns+`
		FROM spending_limits
		WHERE project_id = $1
		ORDER BY created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.SpendingLimit
	for rows.Next() {
		l, err := scanSpendingLimit(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, rows.Err()
}

func (r *SpendingLimitRepository) GetByID(ctx context.Context, id int64) (domain.SpendingLimit, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+slColumns+` FROM spending_limits WHERE id = $1`, id)
	l, err := scanSpendingLimit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SpendingLimit{}, &apperrors.NotFoundError{Resource: "spending_limit"}
	}
	return l, err
}

func (r *SpendingLimitRepository) Create(ctx context.Context, l domain.SpendingLimit, projectID int64) (domain.SpendingLimit, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO spending_limits (tag_id, amount, period_type, period_start, period_end, project_id)
		VALUES ($1, $2, $3::limit_period_type, $4, $5, $6)
		RETURNING `+slColumns,
		l.TagID, l.Amount, l.PeriodType, l.PeriodStart, l.PeriodEnd, projectID)
	return scanSpendingLimit(row)
}

func (r *SpendingLimitRepository) Update(ctx context.Context, l domain.SpendingLimit) (domain.SpendingLimit, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE spending_limits SET
			tag_id=$2, amount=$3, period_type=$4::limit_period_type, period_start=$5, period_end=$6
		WHERE id=$1
		RETURNING `+slColumns,
		l.ID, l.TagID, l.Amount, l.PeriodType, l.PeriodStart, l.PeriodEnd)
	updated, err := scanSpendingLimit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SpendingLimit{}, &apperrors.NotFoundError{Resource: "spending_limit"}
	}
	return updated, err
}

func (r *SpendingLimitRepository) Delete(ctx context.Context, id int64) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM spending_limits WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return &apperrors.NotFoundError{Resource: "spending_limit"}
	}
	return nil
}

// UpsertMonthlyForTag создаёт или заменяет единственный месячный лимит тега в проекте.
func (r *SpendingLimitRepository) UpsertMonthlyForTag(ctx context.Context, tagID, amount, projectID int64) (domain.SpendingLimit, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO spending_limits (tag_id, amount, period_type, project_id)
		VALUES ($1, $2, 'month'::limit_period_type, $3)
		ON CONFLICT (project_id, tag_id) WHERE period_type = 'month'
		DO UPDATE SET amount = EXCLUDED.amount
		RETURNING `+slColumns,
		tagID, amount, projectID)
	return scanSpendingLimit(row)
}

// DeleteMonthlyForTag удаляет месячный лимит тега; отсутствие лимита — не ошибка.
func (r *SpendingLimitRepository) DeleteMonthlyForTag(ctx context.Context, tagID, projectID int64) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM spending_limits
		WHERE tag_id = $1 AND project_id = $2 AND period_type = 'month'`, tagID, projectID)
	return err
}

// SumSpentInPeriod — сумма расходов по тегу (включая непосредственные подтеги)
// в пределах [from, to).
func (r *SpendingLimitRepository) SumSpentInPeriod(ctx context.Context, tagID int64, from, to time.Time, projectID int64) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(t.amount), 0)
		FROM transactions t
		WHERE t.category = 'expense'::transaction_category
		  AND t.project_id = $1
		  AND (t.tag_id = $2 OR t.tag_id IN (SELECT id FROM tags WHERE parent_id = $2))
		  AND t.date >= $3 AND t.date < $4`, projectID, tagID, from, to).Scan(&total)
	return total, err
}
