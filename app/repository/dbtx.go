package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

// DBTX adalah abstraksi dari *sql.DB dan *sql.Tx sehingga query yang sama
// dapat dijalankan di luar maupun di dalam database transaction.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// rowScanner mengabstraksi sql.Row & sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// Error domain yang dipakai lintas layer repository & service.
var (
	// ErrNotFound menandakan data tidak ditemukan.
	ErrNotFound = errors.New("data tidak ditemukan")
	// ErrDuplicate menandakan pelanggaran unique constraint.
	ErrDuplicate = errors.New("data sudah ada")
	// ErrInUse menandakan data masih direferensikan tabel lain.
	ErrInUse = errors.New("data masih digunakan oleh data lain")
)

// conn memilih transaction jika tersedia, jika tidak memakai *sql.DB.
func conn(tx DBTX, db *sql.DB) DBTX {
	if tx != nil {
		return tx
	}
	return db
}

// mapError menerjemahkan error driver PostgreSQL menjadi error domain.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505": // unique_violation
			return ErrDuplicate
		case "23503": // foreign_key_violation
			return ErrInUse
		}
	}

	return err
}
