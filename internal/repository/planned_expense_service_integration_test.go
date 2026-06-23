//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPlannedExpenseService_Create_WithInlineNewCategory(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	catRepo := repository.NewPlannedExpenseCategoryRepository(pool)
	catSvc := service.NewPlannedExpenseCategoryService(catRepo)
	repo := repository.NewPlannedExpenseRepository(pool)
	svc := service.NewPlannedExpenseService(repo, catRepo, catSvc)

	created, err := svc.Create(ctx, dto.CreatePlannedExpenseRequest{
		Title:    "Наушники",
		Priority: "medium",
		NewCategory: &dto.CreatePlannedExpenseCategoryRequest{
			Name:  "Электроника",
			Color: "#112250",
		},
	})
	require.NoError(t, err)
	require.NotZero(t, created.CategoryID)

	cats, err := catSvc.List(ctx)
	require.NoError(t, err)
	require.Len(t, cats, 1)
	require.Equal(t, "Электроника", cats[0].Name)
	require.Equal(t, created.CategoryID, cats[0].ID)
}

func TestPlannedExpenseService_Create_InvalidCategoryColor(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	catRepo := repository.NewPlannedExpenseCategoryRepository(pool)
	catSvc := service.NewPlannedExpenseCategoryService(catRepo)
	repo := repository.NewPlannedExpenseRepository(pool)
	svc := service.NewPlannedExpenseService(repo, catRepo, catSvc)

	_, err := svc.Create(ctx, dto.CreatePlannedExpenseRequest{
		Title:    "Наушники",
		Priority: "medium",
		NewCategory: &dto.CreatePlannedExpenseCategoryRequest{
			Name:  "Электроника",
			Color: "#ABCDEF",
		},
	})
	require.Error(t, err)
}

func TestPlannedExpenseService_Complete_PreservesAllFields(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	catRepo := repository.NewPlannedExpenseCategoryRepository(pool)
	catSvc := service.NewPlannedExpenseCategoryService(catRepo)
	repo := repository.NewPlannedExpenseRepository(pool)
	svc := service.NewPlannedExpenseService(repo, catRepo, catSvc)

	cost := int64(250000)
	due := "2026-07-01"
	url := "https://example.com/item"
	created, err := svc.Create(ctx, dto.CreatePlannedExpenseRequest{
		Title:       "Кофемашина",
		CostKopecks: &cost,
		DueDate:     &due,
		URL:         &url,
		Priority:    "low",
		NewCategory: &dto.CreatePlannedExpenseCategoryRequest{Name: "Дом", Color: "#3C5070"},
	})
	require.NoError(t, err)

	archived, err := svc.Complete(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.PlannedExpenseStatusArchived, archived.Status)
	require.Equal(t, "Кофемашина", archived.Title)
	require.Equal(t, cost, *archived.CostKopecks)
	require.Equal(t, due, *archived.DueDate)
	require.Equal(t, url, *archived.URL)
	require.Equal(t, created.CategoryID, archived.CategoryID)
	require.Equal(t, "low", archived.Priority)
	require.NotNil(t, archived.ArchivedAt)

	archivedList, err := svc.ListArchived(ctx)
	require.NoError(t, err)
	require.Len(t, archivedList, 1)
	require.Equal(t, created.ID, archivedList[0].ID)
}

func TestPlannedExpenseService_ListActive_SortsByEffectivePriority(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	catRepo := repository.NewPlannedExpenseCategoryRepository(pool)
	catSvc := service.NewPlannedExpenseCategoryService(catRepo)
	repo := repository.NewPlannedExpenseRepository(pool)
	svc := service.NewPlannedExpenseService(repo, catRepo, catSvc)

	cat, err := catSvc.Create(ctx, dto.CreatePlannedExpenseCategoryRequest{Name: "Покупки", Color: "#112250"})
	require.NoError(t, err)
	catID := cat.ID

	_, err = svc.Create(ctx, dto.CreatePlannedExpenseRequest{Title: "Низкий", Priority: "low", CategoryID: &catID})
	require.NoError(t, err)
	_, err = svc.Create(ctx, dto.CreatePlannedExpenseRequest{Title: "Средний", Priority: "medium", CategoryID: &catID})
	require.NoError(t, err)
	soonDue := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	_, err = svc.Create(ctx, dto.CreatePlannedExpenseRequest{Title: "Скоро горит", Priority: "low", DueDate: &soonDue, CategoryID: &catID})
	require.NoError(t, err)

	active, err := svc.ListActive(ctx)
	require.NoError(t, err)
	require.Len(t, active, 1)
	items := active[0].Items
	require.Len(t, items, 3)

	require.Equal(t, "Скоро горит", items[0].Title)
	require.Equal(t, "high", items[0].EffectivePriority)
	require.True(t, items[0].IsDueSoon)

	require.Equal(t, "Средний", items[1].Title)
	require.Equal(t, "Низкий", items[2].Title)
}

func TestPlannedExpenseCategoryService_Reorder_RejectsInvalidPermutation(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	catRepo := repository.NewPlannedExpenseCategoryRepository(pool)
	catSvc := service.NewPlannedExpenseCategoryService(catRepo)

	a, err := catSvc.Create(ctx, dto.CreatePlannedExpenseCategoryRequest{Name: "A", Color: "#112250"})
	require.NoError(t, err)
	b, err := catSvc.Create(ctx, dto.CreatePlannedExpenseCategoryRequest{Name: "B", Color: "#3C5070"})
	require.NoError(t, err)

	err = catSvc.Reorder(ctx, []int64{a.ID, a.ID})
	require.Error(t, err)

	require.NoError(t, catSvc.Reorder(ctx, []int64{b.ID, a.ID}))
	list, err := catSvc.List(ctx)
	require.NoError(t, err)
	require.Equal(t, b.ID, list[0].ID)
	require.Equal(t, a.ID, list[1].ID)
}
