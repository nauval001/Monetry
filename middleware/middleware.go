package middleware

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"monetry-api/helper"
)

// LocalsUserID & kawan-kawan adalah key context yang diset oleh AuthMiddleware
// setelah access token berhasil diverifikasi.
const (
	LocalsUserID   = "user_id"
	LocalsUsername = "username"
	LocalsRole     = "role"
)

// RequestLogger mencatat setiap request menggunakan slog.
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		logger.Info("http_request",
			"method", c.Method(),
			"path", c.Path(),
			"status", resolveStatus(c, err),
			"ip", c.IP(),
			"duration_ms", time.Since(start).Milliseconds(),
			"error", errString(err),
		)

		return err
	}
}

// resolveStatus mengembalikan status HTTP final. Ketika handler mengembalikan
// error, status belum diterapkan ke response (baru di-set oleh ErrorHandler),
// sehingga status diambil dari error tersebut.
func resolveStatus(c *fiber.Ctx, err error) int {
	if err == nil {
		return c.Response().StatusCode()
	}

	if appErr, ok := helper.AsAppError(err); ok {
		return appErr.Status
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return fiberErr.Code
	}

	return fiber.StatusInternalServerError
}

// ErrorHandler adalah error handler global Fiber. Semua error yang dikembalikan
// handler akan dikonversi ke format response standar.
func ErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		if appErr, ok := helper.AsAppError(err); ok {
			if appErr.Status >= fiber.StatusInternalServerError {
				logger.Error("request_failed", "path", c.Path(), "error", appErr.Error())
			}
			return helper.FailWithDetails(c, appErr.Status, appErr.Message, appErr.Details)
		}

		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return helper.FailWithDetails(c, fiberErr.Code, messageForStatus(fiberErr.Code), nil)
		}

		logger.Error("unhandled_error", "path", c.Path(), "error", err.Error())
		return helper.FailWithDetails(c, fiber.StatusInternalServerError, "terjadi kesalahan pada server", nil)
	}
}

func messageForStatus(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "permintaan tidak valid"
	case fiber.StatusUnauthorized:
		return "autentikasi diperlukan"
	case fiber.StatusForbidden:
		return "anda tidak memiliki akses ke resource ini"
	case fiber.StatusNotFound:
		return "endpoint tidak ditemukan"
	case fiber.StatusMethodNotAllowed:
		return "method tidak diizinkan"
	case fiber.StatusUnprocessableEntity:
		return "validasi data gagal"
	case fiber.StatusTooManyRequests:
		return "terlalu banyak permintaan"
	default:
		return "terjadi kesalahan pada server"
	}
}

// BearerToken mengambil token dari header Authorization.
func BearerToken(c *fiber.Ctx) (string, bool) {
	header := c.Get(fiber.HeaderAuthorization)
	if header == "" {
		return "", false
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
