package repository

import (
	"context"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlannedExpenseCategoryRepository struct {
	pool *pgxpool.Pool
}

func NewPlannedExpenseCategoryRepository(pool *pgxpool.Pool) *PlannedExpenseCategoryRepository {
	return &PlannedExpenseCategoryRepository{pool: pool}
}

const pecColumns = `id, name, color, sort_order, created_at`

func scanPlannedExpenseCategory(row pgx.Row) (domain.PlannedExpenseCategory, error) {
	var c domain.PlannedExpenseCategory
	err := row.Scan(&c.ID, &c.Name, &c.Color, &c.SortOrder, &c.CreatedAt)
	return c, err
}

func (r *PlannedExpenseCategoryRepository) List(ctx context.Context, projectID int64) ([]domain.PlannedExpenseCategory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+pecColumns+`
		FROM planned_expense_categories
		WHERE project_id = $1
		ORDER BY sort_order ASC, id ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.PlannedExpenseCategory
	for rows.Next() {
		c, err := scanPlannedExpenseCategory(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *PlannedExpenseCategoryRepository) Create(ctx context.Context, name, color string, projectID int64) (domain.PlannedExpenseCategory, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO planned_expense_categories (name, color, sort_order, project_id)
		VALUES ($1, $2, COALESCE((SELECT MAX(sort_order) + 1 FROM planned_expense_categories WHERE project_id = $3), 0), $3)
		RETURNING `+pecColumns,
		name, color, projectID)
	return scanPlannedExpenseCategory(row)
}

func (r *PlannedExpenseCategoryRepository) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM planned_expense_categories WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

func (r *PlannedExpenseCategoryRepository) Reorder(ctx context.Context, ids []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for position, id := range ids {
		tag, err := tx.Exec(ctx, `UPDATE planned_expense_categories SET sort_order = $2 WHERE id = $1`, id, position)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return &apperrors.NotFoundError{Resource: "planned_expense_category"}
		}
	}

	return tx.Commit(ctx)
}
