//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestImportRepository_AcceptRows_SingleRow(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	var tagID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO tags (name, color) VALUES ('Еда', '#FF0000') RETURNING id`).Scan(&tagID))

	repo := repository.NewImportRepository(pool)
	batch, err := repo.CreateBatch(ctx, "transactions.csv", 1)
	require.NoError(t, err)

	title := "Продукты"
	amount := int64(125000)
	date := mustParseDate(t, "2026-06-15")
	category := "expense"
	specificity := "simple"

	inserted, err := repo.InsertRows(ctx, []domain.ModerationRow{{
		BatchID:     batch.ID,
		RowNumber:   1,
		Title:       &title,
		Amount:      &amount,
		Date:        &date,
		TagID:       &tagID,
		Category:    &category,
		Specificity: &specificity,
		Status:      domain.ModerationRowStatusReady,
		FieldErrors: map[string]string{},
	}})
	require.NoError(t, err)
	require.Len(t, inserted, 1)

	created, err := repo.AcceptRows(ctx, batch.ID, []int64{inserted[0].ID})
	require.NoError(t, err)
	require.Len(t, created, 1)
	require.Equal(t, title, created[0].Title)
	require.Equal(t, amount, created[0].Amount)
	require.Equal(t, tagID, created[0].TagID)

	var txCount int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE id = $1`, created[0].ID).Scan(&txCount))
	require.Equal(t, int64(1), txCount)

	rows, err := repo.ListRows(ctx, batch.ID)
	require.NoError(t, err)
	require.Empty(t, rows, "accepted row must be deleted from moderation_transactions")
}

func TestImportRepository_AcceptRows_BatchOnlyAcceptsReady(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	var tagID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO tags (name, color) VALUES ('Еда', '#FF0000') RETURNING id`).Scan(&tagID))

	repo := repository.NewImportRepository(pool)
	batch, err := repo.CreateBatch(ctx, "transactions.csv", 2)
	require.NoError(t, err)

	titleReady := "Готово"
	titlePending := "Ожидает"
	amount := int64(50000)
	date := mustParseDate(t, "2026-06-15")
	category := "expense"
	specificity := "simple"

	inserted, err := repo.InsertRows(ctx, []domain.ModerationRow{
		{
			BatchID: batch.ID, RowNumber: 1, Title: &titleReady, Amount: &amount, Date: &date,
			TagID: &tagID, Category: &category, Specificity: &specificity,
			Status: domain.ModerationRowStatusReady, FieldErrors: map[string]string{},
		},
		{
			BatchID: batch.ID, RowNumber: 2, Title: &titlePending, Amount: &amount, Date: &date,
			Status: domain.ModerationRowStatusPending, FieldErrors: map[string]string{},
		},
	})
	require.NoError(t, err)
	require.Len(t, inserted, 2)

	ids := []int64{inserted[0].ID, inserted[1].ID}
	created, err := repo.AcceptRows(ctx, batch.ID, ids)
	require.NoError(t, err)
	require.Len(t, created, 1, "only the ready row should be accepted")
	require.Equal(t, titleReady, created[0].Title)

	rows, err := repo.ListRows(ctx, batch.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1, "pending row must remain in moderation_transactions")
	require.Equal(t, titlePending, *rows[0].Title)
}

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return parsed
}
