package repository

import (
	"context"

	"github.com/SimonLavlinskiy/finAns-backend/internal/repository/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewHealthRepository(pool *pgxpool.Pool) *HealthRepository {
	return &HealthRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *HealthRepository) Ping(ctx context.Context) error {
	_, err := r.queries.Ping(ctx)
	return err
}
