//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/stretchr/testify/require"
)

// MarkPaid должен создавать транзакцию-расход за платёж и сдвигать next_payment_date.
func TestMandatoryPaymentService_MarkPaid_CreatesTransactionAndAdvancesDate(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	var tagID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO tags (name, color) VALUES ('Аренда', '#FF0000') RETURNING id`).Scan(&tagID))

	tagRepo := repository.NewTagRepository(pool)
	txRepo := repository.NewTransactionRepository(pool)
	mpRepo := repository.NewMandatoryPaymentRepository(pool)
	tagSvc := service.NewTagService(tagRepo)
	mpSvc := service.NewMandatoryPaymentService(mpRepo, tagRepo, tagSvc, txRepo)

	created, err := mpSvc.Create(ctx, dto.CreateMandatoryPaymentRequest{
		Title:           "Аренда квартиры",
		Amount:          3500000,
		TagID:           tagID,
		Recurrence:      "monthly",
		NextPaymentDate: "2026-06-01",
	})
	require.NoError(t, err)

	updated, err := mpSvc.MarkPaid(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "2026-07-01", updated.NextPaymentDate)

	var txCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM transactions WHERE title = $1 AND amount = $2 AND tag_id = $3`,
		"Аренда квартиры", int64(3500000), tagID).Scan(&txCount))
	require.Equal(t, 1, txCount)

	var category, specificity string
	var date time.Time
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT category::text, specificity::text, date FROM transactions WHERE title = $1`,
		"Аренда квартиры").Scan(&category, &specificity, &date))
	require.Equal(t, "expense", category)
	require.Equal(t, "required", specificity)
	require.Equal(t, "2026-06-01", date.Format("2006-01-02"))
}
