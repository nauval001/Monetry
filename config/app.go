package config

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"monetry-api/helper"
	"monetry-api/middleware"
)

// NewFiberApp merakit instance Fiber beserta middleware global.
func NewFiberApp(cfg *Config, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               cfg.AppName,
		DisableStartupMessage: false,
		ReadTimeout:           15 * time.Second,
		WriteTimeout:          15 * time.Second,
		IdleTimeout:           60 * time.Second,
		BodyLimit:             4 * 1024 * 1024,
		ErrorHandler:          middleware.ErrorHandler(logger),
	})

	app.Use(recover.New())
	app.Use(middleware.RequestLogger(logger))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))
	app.Use(limiter.New(limiter.Config{
		Max:        cfg.RateLimitMax,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return helper.FailWithDetails(c, fiber.StatusTooManyRequests, "terlalu banyak permintaan, coba lagi nanti", nil)
		},
	}))

	// Catatan: handler fallback 404 didaftarkan di route.Setup setelah seluruh
	// route terdaftar, karena Fiber mengeksekusi middleware sesuai urutan.

	return app
}
