package domain

import (
	"context"
	"time"
)

type Transaction struct {
	ID          int64
	Title       string
	Amount      int64
	Date        time.Time
	TagID       int64
	Category    string
	Specificity string
	Comment     *string
	URL         *string
	FilePath    *string
	FileName    *string
	FileMIME    *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TransactionFilters struct {
	Search      string
	AmountMin   *int64
	AmountMax   *int64
	DateFrom    *time.Time
	DateTo      *time.Time
	TagIDs      []int64
	Category    string
	Specificity string
	Page        int
	PerPage     int
	SortBy      string
	SortOrder   string
}

type ListResult struct {
	Items []Transaction
	Total int64
}

type TransactionRepository interface {
	List(ctx context.Context, f TransactionFilters) (ListResult, error)
	GetByID(ctx context.Context, id int64) (Transaction, error)
	Create(ctx context.Context, t Transaction) (Transaction, error)
	Update(ctx context.Context, t Transaction) (Transaction, error)
	Delete(ctx context.Context, id int64) error
	Suggestions(ctx context.Context, q string, limit int) ([]string, error)
}
