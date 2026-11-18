package helper

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	dateLayout   = "2006-01-02"
	defaultLimit = 10
	maxLimit     = 100
)

// EncodeCursor membentuk cursor opaque dari pasangan (transaction_date, id).
// Cursor dipakai untuk cursor-based pagination yang stabil walaupun data baru
// terus bertambah.
func EncodeCursor(date time.Time, id int64) string {
	raw := fmt.Sprintf("%s|%d", date.Format(dateLayout), id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor membalikkan cursor menjadi (tanggal transaksi, id).
func DecodeCursor(cursor string) (time.Time, int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, 0, errors.New("cursor tidak valid")
	}

	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, errors.New("cursor tidak valid")
	}

	date, err := time.ParseInLocation(dateLayout, parts[0], time.UTC)
	if err != nil {
		return time.Time{}, 0, errors.New("cursor tidak valid")
	}

	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || id <= 0 {
		return time.Time{}, 0, errors.New("cursor tidak valid")
	}

	return date, id, nil
}

// ParsePagination membaca query `page` & `limit` dengan nilai default
// dan batas maksimum agar query tetap terkendali.
func ParsePagination(c *fiber.Ctx) (page int, limit int) {
	page = QueryInt(c, "page", 1)
	limit = QueryInt(c, "limit", defaultLimit)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

// Offset menghitung offset dari page & limit.
func Offset(page, limit int) int {
	return (page - 1) * limit
}

// QueryInt membaca query integer dengan fallback.
func QueryInt(c *fiber.Ctx, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

// QueryInt64 membaca query integer 64 bit dengan fallback.
func QueryInt64(c *fiber.Ctx, key string, fallback int64) int64 {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

// ParseMonthYear membaca query `month` & `year`. Bila tidak diisi,
// digunakan bulan & tahun berjalan (zona waktu UTC).
func ParseMonthYear(c *fiber.Ctx) (month, year int) {
	now := time.Now().UTC()
	month = QueryInt(c, "month", int(now.Month()))
	year = QueryInt(c, "year", now.Year())

	if month < 1 || month > 12 {
		month = int(now.Month())
	}
	if year < 1970 || year > 9999 {
		year = now.Year()
	}
	return month, year
}

// ParseDate mem-parse tanggal dengan layout YYYY-MM-DD.
func ParseDate(value string) (time.Time, error) {
	return time.ParseInLocation(dateLayout, value, time.UTC)
}

// FormatDate memformat tanggal menjadi YYYY-MM-DD.
func FormatDate(t time.Time) string {
	return t.Format(dateLayout)
}
