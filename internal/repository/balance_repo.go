package repository

import (
	"context"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BalanceRepository struct {
	pool *pgxpool.Pool
}

func NewBalanceRepository(pool *pgxpool.Pool) *BalanceRepository {
	return &BalanceRepository{pool: pool}
}

func (r *BalanceRepository) GetSnapshot(ctx context.Context, projectID int64) (domain.BalanceSnapshot, error) {
	var snap domain.BalanceSnapshot
	err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT initial_amount FROM user_balance WHERE project_id = $1 ORDER BY id LIMIT 1), 0),
			COALESCE((SELECT SUM(amount) FROM transactions WHERE category = 'income' AND project_id = $1), 0),
			COALESCE((SELECT SUM(amount) FROM transactions WHERE category = 'expense' AND project_id = $1), 0)
	`, projectID).Scan(&snap.InitialAmount, &snap.TotalIncome, &snap.TotalExpense)
	if err != nil {
		return domain.BalanceSnapshot{}, err
	}
	snap.CurrentBalance = snap.InitialAmount + snap.TotalIncome - snap.TotalExpense
	return snap, nil
}

func (r *BalanceRepository) UpsertInitialAmount(ctx context.Context, amount int64, projectID int64) error {
	var count int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_balance WHERE project_id = $1`, projectID).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		_, err := r.pool.Exec(ctx, `INSERT INTO user_balance (initial_amount, project_id) VALUES ($1, $2)`, amount, projectID)
		return err
	}
	_, err := r.pool.Exec(ctx, `UPDATE user_balance SET initial_amount = $1, updated_at = NOW() WHERE project_id = $2`, amount, projectID)
	return err
}

func (r *BalanceRepository) SetCurrentBalanceAtomic(ctx context.Context, target int64, projectID int64) (domain.BalanceSnapshot, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.BalanceSnapshot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var snap domain.BalanceSnapshot
	if err := tx.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT initial_amount FROM user_balance WHERE project_id = $1 ORDER BY id LIMIT 1), 0),
			COALESCE((SELECT SUM(amount) FROM transactions WHERE category = 'income' AND project_id = $1), 0),
			COALESCE((SELECT SUM(amount) FROM transactions WHERE category = 'expense' AND project_id = $1), 0)
	`, projectID).Scan(&snap.InitialAmount, &snap.TotalIncome, &snap.TotalExpense); err != nil {
		return domain.BalanceSnapshot{}, err
	}

	newInitial := target - snap.TotalIncome + snap.TotalExpense

	var count int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM user_balance WHERE project_id = $1`, projectID).Scan(&count); err != nil {
		return domain.BalanceSnapshot{}, err
	}
	if count == 0 {
		if _, err := tx.Exec(ctx, `INSERT INTO user_balance (initial_amount, project_id) VALUES ($1, $2)`, newInitial, projectID); err != nil {
			return domain.BalanceSnapshot{}, err
		}
	} else {
		if _, err := tx.Exec(ctx, `UPDATE user_balance SET initial_amount = $1, updated_at = NOW() WHERE project_id = $2`, newInitial, projectID); err != nil {
			return domain.BalanceSnapshot{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.BalanceSnapshot{}, err
	}

	snap.InitialAmount = newInitial
	snap.CurrentBalance = target
	return snap, nil
}
