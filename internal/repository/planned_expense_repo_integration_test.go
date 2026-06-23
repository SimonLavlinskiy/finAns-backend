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

func TestPlannedExpenseRepository_CreateGetUpdateDelete(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()
	catRepo := repository.NewPlannedExpenseCategoryRepository(pool)
	repo := repository.NewPlannedExpenseRepository(pool)

	cat, err := catRepo.Create(ctx, "Электроника", "#112250")
	require.NoError(t, err)

	cost := int64(150000)
	created, err := repo.Create(ctx, domain.PlannedExpense{
		CategoryID:  cat.ID,
		Title:       "Наушники",
		CostKopecks: &cost,
		Priority:    domain.PlannedExpensePriorityMedium,
	})
	require.NoError(t, err)
	require.Equal(t, "Наушники", created.Title)
	require.Equal(t, domain.PlannedExpenseStatusActive, created.Status)

	fetched, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)

	fetched.Title = "Беспроводные наушники"
	fetched.Priority = domain.PlannedExpensePriorityHigh
	updated, err := repo.Update(ctx, fetched)
	require.NoError(t, err)
	require.Equal(t, "Беспроводные наушники", updated.Title)
	require.Equal(t, domain.PlannedExpensePriorityHigh, updated.Priority)

	require.NoError(t, repo.Delete(ctx, created.ID))
	_, err = repo.Get(ctx, created.ID)
	require.Error(t, err)
}

func TestPlannedExpenseRepository_ListByStatusAndArchive(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()
	catRepo := repository.NewPlannedExpenseCategoryRepository(pool)
	repo := repository.NewPlannedExpenseRepository(pool)

	cat, err := catRepo.Create(ctx, "Дом", "#3C5070")
	require.NoError(t, err)

	created, err := repo.Create(ctx, domain.PlannedExpense{
		CategoryID: cat.ID,
		Title:      "Пылесос",
		Priority:   domain.PlannedExpensePriorityLow,
	})
	require.NoError(t, err)

	active, err := repo.ListByStatus(ctx, domain.PlannedExpenseStatusActive)
	require.NoError(t, err)
	require.Len(t, active, 1)
	require.Equal(t, created.ID, active[0].ID)

	now := time.Now().UTC()
	archived, err := repo.Archive(ctx, created.ID, now)
	require.NoError(t, err)
	require.Equal(t, domain.PlannedExpenseStatusArchived, archived.Status)
	require.NotNil(t, archived.ArchivedAt)

	active, err = repo.ListByStatus(ctx, domain.PlannedExpenseStatusActive)
	require.NoError(t, err)
	require.Len(t, active, 0)

	archivedList, err := repo.ListByStatus(ctx, domain.PlannedExpenseStatusArchived)
	require.NoError(t, err)
	require.Len(t, archivedList, 1)
	require.Equal(t, created.ID, archivedList[0].ID)
}
