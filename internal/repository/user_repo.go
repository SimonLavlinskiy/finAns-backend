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

func (r *UserRepository) Create(ctx context.Context, username, displayName, passwordHash string) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (username, display_name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, username, display_name, password_hash, is_admin, created_at`,
		username, displayName, passwordHash).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
	return u, err
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, display_name, password_hash, is_admin, created_at
		FROM users WHERE lower(username) = lower($1)`, username).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, &apperrors.NotFoundError{Resource: "user"}
	}
	return u, err
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, display_name, password_hash, is_admin, created_at
		FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, &apperrors.NotFoundError{Resource: "user"}
	}
	return u, err
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, username, display_name, password_hash, is_admin, created_at
		FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Exists(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE lower(username) = lower($1))`, username).Scan(&exists)
	return exists, err
}
