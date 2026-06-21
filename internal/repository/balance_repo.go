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

func (r *BalanceRepository) GetSnapshot(ctx context.Context) (domain.BalanceSnapshot, error) {
	var snap domain.BalanceSnapshot
	err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT initial_amount FROM user_balance ORDER BY id LIMIT 1), 0),
			COALESCE((SELECT SUM(amount) FROM transactions WHERE category = 'income'), 0),
			COALESCE((SELECT SUM(amount) FROM transactions WHERE category = 'expense'), 0)
	`).Scan(&snap.InitialAmount, &snap.TotalIncome, &snap.TotalExpense)
	if err != nil {
		return domain.BalanceSnapshot{}, err
	}
	snap.CurrentBalance = snap.InitialAmount + snap.TotalIncome - snap.TotalExpense
	return snap, nil
}

func (r *BalanceRepository) UpsertBalance(ctx context.Context, amount int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_balance (initial_amount) VALUES ($1)
		ON CONFLICT ((id <= 2147483647)) DO UPDATE SET initial_amount = $1, updated_at = NOW()`,
		amount)
	if err == nil {
		return nil
	}
	// Fallback: классический UPSERT через COUNT
	return r.upsertFallback(ctx, amount)
}

func (r *BalanceRepository) upsertFallback(ctx context.Context, amount int64) error {
	var count int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_balance`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		_, err := r.pool.Exec(ctx, `INSERT INTO user_balance (initial_amount) VALUES ($1)`, amount)
		return err
	}
	_, err := r.pool.Exec(ctx, `UPDATE user_balance SET initial_amount = $1, updated_at = NOW()`, amount)
	return err
}

// SetCurrentBalanceAtomic атомарно вычисляет initial_amount из target balance в одной транзакции БД.
// target = initial_amount + total_income - total_expense → initial_amount = target - total_income + total_expense
func (r *BalanceRepository) SetCurrentBalanceAtomic(ctx context.Context, target int64) (domain.BalanceSnapshot, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.BalanceSnapshot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var snap domain.BalanceSnapshot
	if err := tx.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT initial_amount FROM user_balance ORDER BY id LIMIT 1), 0),
			COALESCE((SELECT SUM(amount) FROM transactions WHERE category = 'income'), 0),
			COALESCE((SELECT SUM(amount) FROM transactions WHERE category = 'expense'), 0)
	`).Scan(&snap.InitialAmount, &snap.TotalIncome, &snap.TotalExpense); err != nil {
		return domain.BalanceSnapshot{}, err
	}

	newInitial := target - snap.TotalIncome + snap.TotalExpense

	var count int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM user_balance`).Scan(&count); err != nil {
		return domain.BalanceSnapshot{}, err
	}
	if count == 0 {
		if _, err := tx.Exec(ctx, `INSERT INTO user_balance (initial_amount) VALUES ($1)`, newInitial); err != nil {
			return domain.BalanceSnapshot{}, err
		}
	} else {
		if _, err := tx.Exec(ctx, `UPDATE user_balance SET initial_amount = $1, updated_at = NOW()`, newInitial); err != nil {
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
