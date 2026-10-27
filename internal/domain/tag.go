package domain

import (
	"context"
	"time"
)

const (
	TagKindExpense = "expense"
	TagKindIncome  = "income"
)

type Tag struct {
	ID        int64
	ParentID  *int64
	Name      string
	Color     string
	Kind      string
	Order     int
	CreatedAt time.Time
	DeletedAt *time.Time
}

type TagRepository interface {
	List(ctx context.Context) ([]Tag, error)
	GetByID(ctx context.Context, id int64) (Tag, error)
	Create(ctx context.Context, name string, color string, parentID *int64) (Tag, error)
	Update(ctx context.Context, id int64, name string, color string) (Tag, error)
	CountUsage(ctx context.Context, id int64) (int64, error)
	ListDescendantIDs(ctx context.Context, id int64) ([]int64, error)
}
