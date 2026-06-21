package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/config"
	"github.com/SimonLavlinskiy/finAns-backend/internal/handler"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

//	@title			finAns API
//	@version		1.0
//	@description	REST API for finAns personal finance admin
//	@BasePath		/

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Warn("database ping failed on startup", "error", err)
	}

	healthRepo := repository.NewHealthRepository(pool)
	healthSvc := service.NewHealthService(healthRepo, cfg.AppVersion)
	healthHandler := handler.NewHealthHandler(healthSvc)

	tagRepo := repository.NewTagRepository(pool)
	txRepo := repository.NewTransactionRepository(pool)
	balRepo := repository.NewBalanceRepository(pool)
	userRepo := repository.NewUserRepository(pool)

	tagSvc := service.NewTagService(tagRepo)
	fileSvc := service.NewFileService(cfg.UploadDir, txRepo)
	txSvc := service.NewTransactionService(txRepo, tagRepo, tagSvc, fileSvc)
	balSvc := service.NewBalanceService(balRepo)
	authSvc := service.NewAuthService(userRepo, []byte(cfg.SessionSecret))

	router := handler.NewRouter(handler.RouterDeps{
		Logger:             logger,
		CORSOrigins:        cfg.CORSOrigins,
		SessionSecret:      []byte(cfg.SessionSecret),
		HealthHandler:      healthHandler,
		AuthHandler:        handler.NewAuthHandler(authSvc, cfg.SecureCookies),
		TransactionHandler: handler.NewTransactionHandler(txSvc),
		TagHandler:         handler.NewTagHandler(tagSvc),
		BalanceHandler:     handler.NewBalanceHandler(balSvc),
		FileHandler:        handler.NewFileHandler(fileSvc),
	})

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("server shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
}

func newLogger(level, format string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler
	if format == "json" {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(h)
}
