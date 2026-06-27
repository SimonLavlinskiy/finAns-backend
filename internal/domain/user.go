package domain

import (
	"context"
	"time"
)

type User struct {
	ID          int64
	Username    string
	DisplayName string
	CreatedAt   time.Time
}

type UserRepository interface {
	Create(ctx context.Context, username, displayName string) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	GetByID(ctx context.Context, id int64) (User, error)
	List(ctx context.Context) ([]User, error)
	Exists(ctx context.Context, username string) (bool, error)
}
