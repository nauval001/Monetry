package route

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"

	"monetry-api/app/model"
	"monetry-api/app/service"
	"monetry-api/helper"
	"monetry-api/middleware"
)

// Handlers menampung seluruh service yang dibutuhkan oleh route.
type Handlers struct {
	Auth        *service.AuthService
	User        *service.UserService
	Category    *service.CategoryService
	Transaction *service.TransactionService
	Budget      *service.BudgetService
	Dashboard   *service.DashboardService
	JWT         *helper.JWTManager
}

// Setup mendaftarkan seluruh endpoint API v1 beserta middleware-nya.
//
// Endpoint publik: register, login, refresh.
// Endpoint user : CRUD kategori, transaksi, budget, dashboard, me, logout.
// Endpoint admin: daftar seluruh user.
func Setup(app *fiber.App, handlers Handlers) {
	app.Get("/health", healthCheck)
	app.Get("/swagger/*", swagger.HandlerDefault)

	api := app.Group("/api/v1")

	// Publik
	api.Post("/auth/register", handlers.Auth.Register)
	api.Post("/auth/login", handlers.Auth.Login)
	api.Post("/auth/refresh", handlers.Auth.Refresh)

	// Butuh access token
	protected := api.Group("", middleware.Auth(handlers.JWT))

	protected.Post("/auth/logout", handlers.Auth.Logout)
	protected.Get("/auth/me", handlers.User.Me)

	protected.Post("/categories", handlers.Category.Create)
	protected.Get("/categories", handlers.Category.List)
	protected.Put("/categories/:id", handlers.Category.Update)
	protected.Delete("/categories/:id", handlers.Category.Delete)

	protected.Post("/transactions", handlers.Transaction.Create)
	protected.Get("/transactions", handlers.Transaction.List)
	protected.Put("/transactions/:id", handlers.Transaction.Update)
	protected.Delete("/transactions/:id", handlers.Transaction.Delete)

	protected.Post("/budgets", handlers.Budget.Set)
	protected.Get("/budgets", handlers.Budget.List)

	protected.Get("/dashboard/summary", handlers.Dashboard.Summary)

	// Khusus admin (RBAC)
	admin := protected.Group("", middleware.RequireRole(model.RoleAdmin))
	admin.Get("/users", handlers.User.ListUsers)

	// Fallback untuk endpoint yang tidak terdaftar (didaftarkan paling akhir
	// agar tidak menutupi route di atasnya).
	app.Use(func(c *fiber.Ctx) error {
		return helper.FailWithDetails(c, fiber.StatusNotFound, "endpoint tidak ditemukan", nil)
	})
}

// healthCheck godoc
//
// @Summary      Health check
// @Description  Memastikan service berjalan.
// @Tags         Health
// @Produce      json
// @Success      200 {object} helper.Response "Service sehat"
// @Router       /health [get]
func healthCheck(c *fiber.Ctx) error {
	return helper.Success(c, fiber.StatusOK, "service sehat", fiber.Map{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}
