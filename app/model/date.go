package model

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DateLayout adalah format tanggal yang dipakai pada API.
const DateLayout = "2006-01-02"

// Date merepresentasikan tanggal (tanpa jam) yang di-serialize menjadi
// "YYYY-MM-DD" pada JSON dan dipetakan ke kolom DATE PostgreSQL.
type Date time.Time

// NewDate mengubah time.Time menjadi Date.
func NewDate(t time.Time) Date {
	return Date(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC))
}

// Time mengembalikan time.Time dari Date.
func (d Date) Time() time.Time { return time.Time(d) }

// String memformat Date menjadi YYYY-MM-DD.
func (d Date) String() string { return d.Time().Format(DateLayout) }

// MarshalJSON mengubah Date menjadi string "YYYY-MM-DD".
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(d.String())), nil
}

// UnmarshalJSON membaca string "YYYY-MM-DD" menjadi Date.
func (d *Date) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(string(data), `"`)
	t, err := time.ParseInLocation(DateLayout, raw, time.UTC)
	if err != nil {
		return fmt.Errorf("format tanggal harus %s", DateLayout)
	}
	*d = Date(t)
	return nil
}

// Value dipakai database/sql saat menyimpan Date.
func (d Date) Value() (driver.Value, error) {
	return d.Time(), nil
}

// Scan dipakai database/sql saat membaca kolom DATE.
func (d *Date) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*d = Date(time.Time{})
		return nil
	case time.Time:
		*d = NewDate(v)
		return nil
	case []byte:
		return d.parse(string(v))
	case string:
		return d.parse(v)
	default:
		return fmt.Errorf("tidak bisa memindai tipe %T ke Date", value)
	}
}

func (d *Date) parse(value string) error {
	t, err := time.ParseInLocation(DateLayout, value[:min(len(value), 10)], time.UTC)
	if err != nil {
		return fmt.Errorf("gagal mem-parse tanggal %q: %w", value, err)
	}
	*d = Date(t)
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
