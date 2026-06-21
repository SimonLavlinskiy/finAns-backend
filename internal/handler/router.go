package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/SimonLavlinskiy/finAns-backend/docs"
)

type RouterDeps struct {
	Logger                   *slog.Logger
	CORSOrigins              []string
	SessionSecret            []byte
	HealthHandler            *HealthHandler
	AuthHandler              *AuthHandler
	TransactionHandler       *TransactionHandler
	TagHandler               *TagHandler
	BalanceHandler           *BalanceHandler
	FileHandler              *FileHandler
	AnalyticsHandler         *AnalyticsHandler
	ImportHandler            *ImportHandler
	MandatoryPaymentHandler  *MandatoryPaymentHandler
}

func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestLogger(deps.Logger))
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   deps.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", deps.HealthHandler.HealthCheck)

		api.Post("/auth/login", deps.AuthHandler.Login)
		api.Post("/auth/logout", deps.AuthHandler.Logout)

		api.Group(func(protected chi.Router) {
			protected.Use(middleware.RequireAuth(deps.SessionSecret))

			protected.Get("/auth/me", deps.AuthHandler.Me)

			protected.Route("/transactions", func(tr chi.Router) {
				tr.Get("/", deps.TransactionHandler.List)
				tr.Post("/", deps.TransactionHandler.Create)
				tr.Get("/suggestions", deps.TransactionHandler.Suggestions)
				tr.Get("/{id}", deps.TransactionHandler.Get)
				tr.Put("/{id}", deps.TransactionHandler.Update)
				tr.Delete("/{id}", deps.TransactionHandler.Delete)
				tr.Post("/{id}/duplicate", deps.TransactionHandler.Duplicate)
				tr.Post("/{id}/file", deps.FileHandler.Upload)
				tr.Delete("/{id}/file", deps.FileHandler.Delete)
			})

			protected.Route("/tags", func(tr chi.Router) {
				tr.Get("/", deps.TagHandler.List)
				tr.Post("/", deps.TagHandler.Create)
				tr.Put("/{id}", deps.TagHandler.Update)
				tr.Delete("/{id}", deps.TagHandler.Delete)
				tr.Get("/{id}/usage", deps.TagHandler.Usage)
			})

			protected.Get("/balance", deps.BalanceHandler.Get)
			protected.Put("/balance", deps.BalanceHandler.Update)

			protected.Get("/analytics/expenses-calendar", deps.AnalyticsHandler.GetExpensesCalendar)

			protected.Route("/import", func(ir chi.Router) {
				ir.Post("/batches", deps.ImportHandler.UploadBatch)
				ir.Get("/batches/active", deps.ImportHandler.GetActiveBatch)
				ir.Post("/batches/{id}/accept", deps.ImportHandler.AcceptBatch)
				ir.Post("/batches/{id}/close", deps.ImportHandler.CloseBatch)
				ir.Patch("/rows/{id}", deps.ImportHandler.UpdateRow)
				ir.Post("/rows/{id}/accept", deps.ImportHandler.AcceptRow)
			})

			protected.Route("/mandatory-payments", func(mp chi.Router) {
				mp.Get("/", deps.MandatoryPaymentHandler.List)
				mp.Post("/", deps.MandatoryPaymentHandler.Create)
				mp.Get("/{id}", deps.MandatoryPaymentHandler.Get)
				mp.Put("/{id}", deps.MandatoryPaymentHandler.Update)
				mp.Delete("/{id}", deps.MandatoryPaymentHandler.Delete)
				mp.Post("/{id}/duplicate", deps.MandatoryPaymentHandler.Duplicate)
				mp.Post("/{id}/mark-paid", deps.MandatoryPaymentHandler.MarkPaid)
			})
		})
	})

	r.NotFound(NotFound)
	r.MethodNotAllowed(MethodNotAllowed)

	return r
}
