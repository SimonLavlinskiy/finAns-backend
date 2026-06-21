package domain

import (
	"context"
	"time"
)

const (
	ImportBatchStatusOpen   = "open"
	ImportBatchStatusClosed = "closed"

	ModerationRowStatusPending = "pending"
	ModerationRowStatusReady   = "ready"
	ModerationRowStatusError   = "error"
)

type ImportBatch struct {
	ID        int64
	FileName  string
	TotalRows int
	Status    string
	CreatedAt time.Time
	ClosedAt  *time.Time
}

type ModerationRow struct {
	ID          int64
	BatchID     int64
	RowNumber   int
	Title       *string
	Amount      *int64
	Date        *time.Time
	TagID       *int64
	Category    *string
	Specificity *string
	Comment     *string
	URL         *string
	Status      string
	FieldErrors map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ImportRepository interface {
	CreateBatch(ctx context.Context, fileName string, totalRows int) (ImportBatch, error)
	InsertRows(ctx context.Context, rows []ModerationRow) ([]ModerationRow, error)
	GetActiveBatch(ctx context.Context) (ImportBatch, bool, error)
	GetBatch(ctx context.Context, id int64) (ImportBatch, error)
	ListRows(ctx context.Context, batchID int64) ([]ModerationRow, error)
	GetRow(ctx context.Context, id int64) (ModerationRow, error)
	UpdateRow(ctx context.Context, row ModerationRow) (ModerationRow, error)
	AcceptRows(ctx context.Context, batchID int64, rowIDs []int64) ([]Transaction, error)
	CloseBatch(ctx context.Context, batchID int64) error
}
