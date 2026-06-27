package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/SimonLavlinskiy/finAns-backend/docs"
)

type RouterDeps struct {
	Logger                        *slog.Logger
	CORSOrigins                   []string
	UserRepo                      domain.UserRepository
	ProjectRepo                   domain.ProjectRepository
	HealthHandler                 *HealthHandler
	UserHandler                   *UserHandler
	ProjectHandler                *ProjectHandler
	TransactionHandler            *TransactionHandler
	TagHandler                    *TagHandler
	BalanceHandler                *BalanceHandler
	FileHandler                   *FileHandler
	AnalyticsHandler              *AnalyticsHandler
	ImportHandler                 *ImportHandler
	MandatoryPaymentHandler       *MandatoryPaymentHandler
	PlannedExpenseCategoryHandler *PlannedExpenseCategoryHandler
	PlannedExpenseHandler         *PlannedExpenseHandler
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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-User-ID", "X-Project-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", deps.HealthHandler.HealthCheck)

		// Public endpoints (no auth required)
		api.Post("/auth/login", deps.UserHandler.Login)
		api.Get("/users", deps.UserHandler.List)

		// User-authenticated routes (require X-User-ID)
		api.Group(func(userRoutes chi.Router) {
			userRoutes.Use(middleware.UserContextMiddleware(deps.UserRepo))

			// Admin-only: create user
			userRoutes.Post("/users", deps.UserHandler.Create)

			// Project management (user context only, no project context needed)
			userRoutes.Get("/projects", deps.ProjectHandler.List)
			userRoutes.Post("/projects", deps.ProjectHandler.Create)

			// Data routes (require both user and project context)
			userRoutes.Group(func(projectRoutes chi.Router) {
				projectRoutes.Use(middleware.ProjectContextMiddleware(deps.ProjectRepo))

				projectRoutes.Get("/projects/{id}", deps.ProjectHandler.Get)
				projectRoutes.Get("/projects/{id}/members", deps.ProjectHandler.ListMembers)
				projectRoutes.Post("/projects/{id}/members", deps.ProjectHandler.AddMember)
				projectRoutes.Delete("/projects/{id}/members/{userID}", deps.ProjectHandler.RemoveMember)

				projectRoutes.Route("/transactions", func(tr chi.Router) {
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

				projectRoutes.Route("/tags", func(tr chi.Router) {
					tr.Get("/", deps.TagHandler.List)
					tr.Post("/", deps.TagHandler.Create)
					tr.Put("/{id}", deps.TagHandler.Update)
					tr.Delete("/{id}", deps.TagHandler.Delete)
					tr.Get("/{id}/usage", deps.TagHandler.Usage)
				})

				projectRoutes.Get("/balance", deps.BalanceHandler.Get)
				projectRoutes.Put("/balance", deps.BalanceHandler.Update)

				projectRoutes.Get("/analytics/expenses-calendar", deps.AnalyticsHandler.GetExpensesCalendar)

				projectRoutes.Route("/import", func(ir chi.Router) {
					ir.Post("/batches", deps.ImportHandler.UploadBatch)
					ir.Get("/batches/active", deps.ImportHandler.GetActiveBatch)
					ir.Post("/batches/{id}/accept", deps.ImportHandler.AcceptBatch)
					ir.Post("/batches/{id}/close", deps.ImportHandler.CloseBatch)
					ir.Patch("/rows/{id}", deps.ImportHandler.UpdateRow)
					ir.Post("/rows/{id}/accept", deps.ImportHandler.AcceptRow)
				})

				projectRoutes.Route("/mandatory-payments", func(mp chi.Router) {
					mp.Get("/", deps.MandatoryPaymentHandler.List)
					mp.Post("/", deps.MandatoryPaymentHandler.Create)
					mp.Get("/{id}", deps.MandatoryPaymentHandler.Get)
					mp.Put("/{id}", deps.MandatoryPaymentHandler.Update)
					mp.Delete("/{id}", deps.MandatoryPaymentHandler.Delete)
					mp.Post("/{id}/duplicate", deps.MandatoryPaymentHandler.Duplicate)
					mp.Post("/{id}/mark-paid", deps.MandatoryPaymentHandler.MarkPaid)
				})

				projectRoutes.Route("/planned-expense-categories", func(pc chi.Router) {
					pc.Get("/", deps.PlannedExpenseCategoryHandler.List)
					pc.Post("/", deps.PlannedExpenseCategoryHandler.Create)
					pc.Patch("/reorder", deps.PlannedExpenseCategoryHandler.Reorder)
				})

				projectRoutes.Route("/planned-expenses", func(pe chi.Router) {
					pe.Get("/", deps.PlannedExpenseHandler.List)
					pe.Post("/", deps.PlannedExpenseHandler.Create)
					pe.Patch("/{id}", deps.PlannedExpenseHandler.Update)
					pe.Delete("/{id}", deps.PlannedExpenseHandler.Delete)
					pe.Post("/{id}/complete", deps.PlannedExpenseHandler.Complete)
				})
			})
		})
	})

	r.NotFound(NotFound)
	r.MethodNotAllowed(MethodNotAllowed)

	return r
}
