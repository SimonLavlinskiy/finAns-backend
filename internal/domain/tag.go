package domain

import (
	"context"
	"time"
)

type Tag struct {
	ID        int64
	ParentID  *int64
	Name      string
	Color     string
	CreatedAt time.Time
}

type TagRepository interface {
	List(ctx context.Context) ([]Tag, error)
	GetByID(ctx context.Context, id int64) (Tag, error)
	Create(ctx context.Context, name string, color string, parentID *int64) (Tag, error)
	Update(ctx context.Context, id int64, name string, color string) (Tag, error)
	Delete(ctx context.Context, id int64) error
	CountUsage(ctx context.Context, id int64) (int64, error)
	ListDescendantIDs(ctx context.Context, id int64) ([]int64, error)
	ReassignTransactions(ctx context.Context, fromTagID int64, toTagID *int64) error
}
