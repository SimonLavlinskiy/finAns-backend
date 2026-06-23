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

type PlannedExpenseRepository struct {
	pool *pgxpool.Pool
}

func NewPlannedExpenseRepository(pool *pgxpool.Pool) *PlannedExpenseRepository {
	return &PlannedExpenseRepository{pool: pool}
}

const peColumns = `id, category_id, title, cost_kopecks, due_date, url, priority::text, status::text, created_at, archived_at`

func scanPlannedExpense(row pgx.Row) (domain.PlannedExpense, error) {
	var e domain.PlannedExpense
	err := row.Scan(
		&e.ID, &e.CategoryID, &e.Title, &e.CostKopecks, &e.DueDate, &e.URL,
		&e.Priority, &e.Status, &e.CreatedAt, &e.ArchivedAt,
	)
	return e, err
}

func (r *PlannedExpenseRepository) ListByStatus(ctx context.Context, status string) ([]domain.PlannedExpense, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+peColumns+`
		FROM planned_expenses
		WHERE status = $1::planned_expense_status
		ORDER BY created_at ASC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.PlannedExpense
	for rows.Next() {
		e, err := scanPlannedExpense(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *PlannedExpenseRepository) Get(ctx context.Context, id int64) (domain.PlannedExpense, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+peColumns+` FROM planned_expenses WHERE id = $1`, id)
	e, err := scanPlannedExpense(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PlannedExpense{}, &apperrors.NotFoundError{Resource: "planned_expense"}
	}
	return e, err
}

func (r *PlannedExpenseRepository) Create(ctx context.Context, e domain.PlannedExpense) (domain.PlannedExpense, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO planned_expenses (category_id, title, cost_kopecks, due_date, url, priority)
		VALUES ($1, $2, $3, $4, $5, $6::planned_expense_priority)
		RETURNING `+peColumns,
		e.CategoryID, e.Title, e.CostKopecks, e.DueDate, e.URL, e.Priority)
	return scanPlannedExpense(row)
}

func (r *PlannedExpenseRepository) Update(ctx context.Context, e domain.PlannedExpense) (domain.PlannedExpense, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE planned_expenses SET
			category_id=$2, title=$3, cost_kopecks=$4, due_date=$5, url=$6, priority=$7::planned_expense_priority
		WHERE id=$1
		RETURNING `+peColumns,
		e.ID, e.CategoryID, e.Title, e.CostKopecks, e.DueDate, e.URL, e.Priority)
	updated, err := scanPlannedExpense(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PlannedExpense{}, &apperrors.NotFoundError{Resource: "planned_expense"}
	}
	return updated, err
}

func (r *PlannedExpenseRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM planned_expenses WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return &apperrors.NotFoundError{Resource: "planned_expense"}
	}
	return nil
}

func (r *PlannedExpenseRepository) Archive(ctx context.Context, id int64, archivedAt time.Time) (domain.PlannedExpense, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE planned_expenses
		SET status='archived'::planned_expense_status, archived_at=$2
		WHERE id=$1
		RETURNING `+peColumns,
		id, archivedAt)
	updated, err := scanPlannedExpense(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PlannedExpense{}, &apperrors.NotFoundError{Resource: "planned_expense"}
	}
	return updated, err
}
