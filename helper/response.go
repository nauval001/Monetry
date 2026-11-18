package helper

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// Meta berisi metadata paginasi.
type Meta struct {
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Total      int64  `json:"total,omitempty"`
	TotalPages int    `json:"total_pages,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasNext    bool   `json:"has_next"`
}

// Response adalah format standar seluruh response API Monetry.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

// JSON mengirim response dengan format standar.
func JSON(c *fiber.Ctx, status int, payload Response) error {
	return c.Status(status).JSON(payload)
}

// Success mengirim response sukses tanpa metadata paginasi.
func Success(c *fiber.Ctx, status int, message string, data any) error {
	return JSON(c, status, Response{Success: true, Message: message, Data: data})
}

// SuccessWithMeta mengirim response sukses beserta metadata paginasi.
func SuccessWithMeta(c *fiber.Ctx, status int, message string, data any, meta *Meta) error {
	return JSON(c, status, Response{Success: true, Message: message, Data: data, Meta: meta})
}

// Fail mengirim response gagal berdasarkan error yang diterima.
// AppError dipetakan ke status & pesan aslinya, error lain menjadi 500.
func Fail(c *fiber.Ctx, err error) error {
	appErr, ok := AsAppError(err)
	if !ok {
		return FailWithDetails(c, http.StatusInternalServerError, "terjadi kesalahan pada server", nil)
	}
	return FailWithDetails(c, appErr.Status, appErr.Message, appErr.Details)
}

// FailWithDetails mengirim response gagal beserta detail opsional.
func FailWithDetails(c *fiber.Ctx, status int, message string, details any) error {
	return JSON(c, status, Response{Success: false, Message: message, Errors: details})
}

// PaginationMeta membangun metadata paginasi berbasis page/limit.
func PaginationMeta(page, limit int, total int64) *Meta {
	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	return &Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
	}
}

// CursorMeta membangun metadata paginasi berbasis cursor.
func CursorMeta(limit int, nextCursor string, hasNext bool) *Meta {
	return &Meta{Limit: limit, NextCursor: nextCursor, HasNext: hasNext}
}
