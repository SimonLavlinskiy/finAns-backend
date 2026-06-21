//go:build integration

package repository_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupAnalyticsTestPool(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()

	pg, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("finans"),
		postgres.WithUsername("finans"),
		postgres.WithPassword("finans"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, pg.Terminate(ctx))
	})

	connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	migrationsPath, err := filepath.Abs(filepath.Join("..", "..", "db", "migrations"))
	require.NoError(t, err)

	m, err := migrate.New("file://"+migrationsPath, connStr)
	require.NoError(t, err)
	require.NoError(t, m.Up())
	m.Close()

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

func TestAnalyticsRepository_SumByDayInMonth(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	var tagID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO tags (name, color) VALUES ('Еда', '#FF0000') RETURNING id`).Scan(&tagID))

	insertTx := func(date string, amount int64, category string) {
		_, err := pool.Exec(ctx, `
			INSERT INTO transactions (title, amount, date, tag_id, category, specificity)
			VALUES ('test', $1, $2, $3, $4::transaction_category, 'simple'::transaction_specificity)`,
			amount, date, tagID, category)
		require.NoError(t, err)
	}
	insertTx("2026-06-11", 50000, "expense")
	insertTx("2026-06-11", 30000, "expense")
	insertTx("2026-06-12", 10000, "expense")
	insertTx("2026-06-11", 99999, "income") // income must not count

	repo := repository.NewAnalyticsRepository(pool)
	sums, err := repo.SumByDayInMonth(ctx, 2026, 6)
	require.NoError(t, err)

	byDay := make(map[int]int64)
	for _, s := range sums {
		byDay[s.Day] = s.Amount
	}
	require.Equal(t, int64(80000), byDay[11])
	require.Equal(t, int64(10000), byDay[12])
}

func TestAnalyticsRepository_ExistsExpenseBefore(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	var tagID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO tags (name, color) VALUES ('Еда', '#FF0000') RETURNING id`).Scan(&tagID))

	_, err := pool.Exec(ctx, `
		INSERT INTO transactions (title, amount, date, tag_id, category, specificity)
		VALUES ('test', 10000, '2026-05-15', $1, 'expense'::transaction_category, 'simple'::transaction_specificity)`,
		tagID)
	require.NoError(t, err)

	repo := repository.NewAnalyticsRepository(pool)

	// 2026-05-15 is before the start of June 2026.
	exists, err := repo.ExistsExpenseBefore(ctx, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.True(t, exists)

	// 2026-05-15 is not before the start of 2026.
	exists, err = repo.ExistsExpenseBefore(ctx, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.False(t, exists)
}

func TestAnalyticsService_GetExpensesCalendar_DayLevelWithSubtagRollup(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	var foodTagID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO tags (name, color) VALUES ('Еда', '#FF0000') RETURNING id`).Scan(&foodTagID))
	var cafeTagID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO tags (name, color, parent_id) VALUES ('Кафе', '#FF8888', $1) RETURNING id`, foodTagID).Scan(&cafeTagID))

	_, err := pool.Exec(ctx, `
		INSERT INTO transactions (title, amount, date, tag_id, category, specificity) VALUES
			('Обед', 50000, '2026-06-11', $1, 'expense'::transaction_category, 'simple'::transaction_specificity),
			('Кофе', 30000, '2026-06-11', $2, 'expense'::transaction_category, 'simple'::transaction_specificity)`,
		foodTagID, cafeTagID)
	require.NoError(t, err)

	analyticsRepo := repository.NewAnalyticsRepository(pool)
	tagRepo := repository.NewTagRepository(pool)
	svc := service.NewAnalyticsService(analyticsRepo, tagRepo)

	result, err := svc.GetExpensesCalendar(ctx, "day", 2026, 6)
	require.NoError(t, err)
	require.Equal(t, int64(80000), result.Total)

	var day11 *domain.CalendarItem
	for i, item := range result.Items {
		if item.Key == "2026-06-11" {
			day11 = &result.Items[i]
		}
	}
	require.NotNil(t, day11)
	require.Equal(t, int64(80000), day11.Amount)
	require.Len(t, day11.BreakdownByTag, 1)
	require.Equal(t, foodTagID, day11.BreakdownByTag[0].TagID)
	require.Equal(t, int64(80000), day11.BreakdownByTag[0].Amount)
}

func TestAnalyticsService_GetExpensesCalendar_MonthLevel(t *testing.T) {
	pool := setupAnalyticsTestPool(t)
	ctx := context.Background()

	var tagID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO tags (name, color) VALUES ('Еда', '#FF0000') RETURNING id`).Scan(&tagID))

	_, err := pool.Exec(ctx, `
		INSERT INTO transactions (title, amount, date, tag_id, category, specificity) VALUES
			('test', 50000, '2026-06-11', $1, 'expense'::transaction_category, 'simple'::transaction_specificity)`,
		tagID)
	require.NoError(t, err)

	analyticsRepo := repository.NewAnalyticsRepository(pool)
	tagRepo := repository.NewTagRepository(pool)
	svc := service.NewAnalyticsService(analyticsRepo, tagRepo)

	result, err := svc.GetExpensesCalendar(ctx, "month", 2026, 0)
	require.NoError(t, err)
	require.Len(t, result.Items, 12)
	require.Equal(t, int64(50000), result.Total)
	for _, item := range result.Items {
		require.Empty(t, item.BreakdownByTag)
		require.Empty(t, item.Transactions)
	}
}
