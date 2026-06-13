//go:build integration

package repository_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestHealthRepository_Ping(t *testing.T) {
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

	repo := repository.NewHealthRepository(pool)
	require.NoError(t, repo.Ping(ctx))
}
