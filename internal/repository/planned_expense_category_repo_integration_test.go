//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestPlannedExpenseCategoryRepository_CreateAndList(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()
	repo := repository.NewPlannedExpenseCategoryRepository(pool)

	first, err := repo.Create(ctx, "Электроника", "#112250")
	require.NoError(t, err)
	require.Equal(t, "Электроника", first.Name)
	require.Equal(t, "#112250", first.Color)
	require.Equal(t, 0, first.SortOrder)

	second, err := repo.Create(ctx, "Одежда", "#3C5070")
	require.NoError(t, err)
	require.Equal(t, 1, second.SortOrder)

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, first.ID, list[0].ID)
	require.Equal(t, second.ID, list[1].ID)
}

func TestPlannedExpenseCategoryRepository_Exists(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()
	repo := repository.NewPlannedExpenseCategoryRepository(pool)

	created, err := repo.Create(ctx, "Дом", "#6B4226")
	require.NoError(t, err)

	exists, err := repo.Exists(ctx, created.ID)
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = repo.Exists(ctx, created.ID+999)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestPlannedExpenseCategoryRepository_Reorder(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()
	repo := repository.NewPlannedExpenseCategoryRepository(pool)

	a, err := repo.Create(ctx, "A", "#112250")
	require.NoError(t, err)
	b, err := repo.Create(ctx, "B", "#3C5070")
	require.NoError(t, err)
	c, err := repo.Create(ctx, "C", "#6B4226")
	require.NoError(t, err)

	require.NoError(t, repo.Reorder(ctx, []int64{c.ID, a.ID, b.ID}))

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.Equal(t, c.ID, list[0].ID)
	require.Equal(t, a.ID, list[1].ID)
	require.Equal(t, b.ID, list[2].ID)
}

func TestPlannedExpenseCategoryRepository_Reorder_UnknownID(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()
	repo := repository.NewPlannedExpenseCategoryRepository(pool)

	a, err := repo.Create(ctx, "A", "#112250")
	require.NoError(t, err)

	err = repo.Reorder(ctx, []int64{a.ID, 999999})
	require.Error(t, err)
	var notFound *apperrors.NotFoundError
	require.ErrorAs(t, err, &notFound)
}
