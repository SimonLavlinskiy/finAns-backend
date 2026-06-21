//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/stretchr/testify/require"
)

// Regression test: patching a moderation row with an invalid category/specificity
// used to write the raw invalid string straight into an `::transaction_category` /
// `::transaction_specificity` cast, which Postgres rejects with SQLSTATE 22P02.
// UpdateRow must instead record a FieldErrors entry and leave the enum column unset.
func TestImportService_UpdateRow_InvalidCategory_DoesNotHitEnumError(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	var tagID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO tags (name, color) VALUES ('Еда', '#FF0000') RETURNING id`).Scan(&tagID))

	importRepo := repository.NewImportRepository(pool)
	tagRepo := repository.NewTagRepository(pool)
	svc := service.NewImportService(importRepo, tagRepo)

	batch, err := importRepo.CreateBatch(ctx, "transactions.csv", 1)
	require.NoError(t, err)

	title := "Покупка"
	amount := int64(40000)
	date := mustParseDate(t, "2026-06-13")

	inserted, err := importRepo.InsertRows(ctx, []domain.ModerationRow{{
		BatchID:     batch.ID,
		RowNumber:   1,
		Title:       &title,
		Amount:      &amount,
		Date:        &date,
		TagID:       &tagID,
		Status:      domain.ModerationRowStatusPending,
		FieldErrors: map[string]string{},
	}})
	require.NoError(t, err)
	require.Len(t, inserted, 1)

	invalidCategory := "покупка"
	updated, err := svc.UpdateRow(ctx, inserted[0].ID, service.UpdateRowInput{Category: &invalidCategory})
	require.NoError(t, err, "must not surface a raw Postgres enum error")

	require.Nil(t, updated.Category)
	require.Equal(t, domain.ModerationRowStatusError, updated.Status)
	require.Contains(t, updated.FieldErrors["category"], "покупка")

	invalidSpecificity := "разовая"
	updated2, err := svc.UpdateRow(ctx, inserted[0].ID, service.UpdateRowInput{Specificity: &invalidSpecificity})
	require.NoError(t, err, "must not surface a raw Postgres enum error")

	require.Nil(t, updated2.Specificity)
	require.Equal(t, domain.ModerationRowStatusError, updated2.Status)
	require.Contains(t, updated2.FieldErrors["specificity"], "разовая")
}
