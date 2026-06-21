package repository

import (
	"context"
	"errors"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, login, password_hash, created_at FROM users WHERE login = $1`, login).
		Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, &apperrors.NotFoundError{Resource: "user"}
	}
	return u, err
}
