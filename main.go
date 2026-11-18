// Package main adalah entry point aplikasi Monetry.
//
// Monetry - REST API pencatatan keuangan pribadi.
// Dibangun dengan Go, Fiber, PostgreSQL, dan Clean Architecture.
package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monetry-api/app/repository"
	"monetry-api/app/service"
	"monetry-api/config"
	"monetry-api/database"
	_ "monetry-api/docs"
	"monetry-api/helper"
	"monetry-api/route"
)

// @title           Monetry API
// @version         1.0
// @description     REST API aplikasi pencatatan keuangan pribadi (Monetry). Mendukung autentikasi JWT dengan rotasi refresh token, manajemen kategori & transaksi, budget bulanan, serta ringkasan dashboard.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Monetry Team
// @contact.email  admin@monetry.local

// @BasePath  /api/v1
// @schemes   http
//
// Catatan: @host sengaja tidak diisi agar Swagger UI memakai origin yang sedang
// dipakai (mendukung port host apa pun, mis. 8080 atau 8081).

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                Ketik "Bearer {access_token}" untuk mengakses endpoint yang dilindungi.

//go:embed migrations/*.sql
var migrationFiles embed.FS

func main() {
	if err := run(); err != nil {
		slog.Error("aplikasi berhenti karena error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := config.NewLogger(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("gagal menutup koneksi database", "error", err)
		}
	}()

	if err := database.Migrate(ctx, db, migrationFiles, "migrations", logger); err != nil {
		return err
	}
	if err := database.SeedAdmin(ctx, db, cfg, logger); err != nil {
		return err
	}

	jwtManager := helper.NewJWTManager(helper.JWTConfig{
		Secret:        cfg.JWTSecret,
		AccessExpiry:  cfg.JWTAccessExpiry,
		RefreshExpiry: cfg.JWTRefreshExpiry,
	})

	// Repository
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	budgetRepo := repository.NewBudgetRepository(db)

	// Service (sekaligus controller)
	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, db, logger)
	userService := service.NewUserService(userRepo)
	categoryService := service.NewCategoryService(categoryRepo, logger)
	transactionService := service.NewTransactionService(transactionRepo, categoryRepo, db, logger)
	budgetService := service.NewBudgetService(budgetRepo, categoryRepo, logger)
	dashboardService := service.NewDashboardService(transactionRepo)

	app := config.NewFiberApp(cfg, logger)

	route.Setup(app, route.Handlers{
		Auth:        authService,
		User:        userService,
		Category:    categoryService,
		Transaction: transactionService,
		Budget:      budgetService,
		Dashboard:   dashboardService,
		JWT:         jwtManager,
	})

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server Monetry berjalan",
			"port", cfg.AppPort,
			"swagger", "http://localhost:"+cfg.AppPort+"/swagger/index.html",
		)
		serverErrors <- app.Listen(":" + cfg.AppPort)
	}()

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		logger.Info("sinyal shutdown diterima, menghentikan server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			return err
		}
		logger.Info("server berhenti dengan rapi")
	}

	return nil
}
