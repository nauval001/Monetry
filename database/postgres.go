package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq" // driver PostgreSQL untuk database/sql

	"monetry-api/config"
)

const (
	connectMaxAttempts = 10
	connectRetryDelay  = 2 * time.Second
	pingTimeout        = 3 * time.Second
)

// Connect membuka connection pool PostgreSQL dan memastikan database
// benar-benar bisa diakses. Retry dilakukan agar container API tetap bisa
// menyala ketika container database belum siap (docker compose).
func Connect(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("membuka koneksi database: %w", err)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	var pingErr error
	for attempt := 1; attempt <= connectMaxAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
		pingErr = db.PingContext(pingCtx)
		cancel()

		if pingErr == nil {
			logger.Info("koneksi database berhasil",
				"host", cfg.DBHost,
				"port", cfg.DBPort,
				"database", cfg.DBName,
				"max_open_conns", cfg.DBMaxOpenConns,
			)
			return db, nil
		}

		logger.Warn("database belum siap, mencoba ulang",
			"attempt", attempt,
			"max_attempts", connectMaxAttempts,
			"error", pingErr,
		)

		select {
		case <-ctx.Done():
			_ = db.Close()
			return nil, ctx.Err()
		case <-time.After(connectRetryDelay):
		}
	}

	_ = db.Close()
	return nil, fmt.Errorf("gagal terhubung ke database setelah %d percobaan: %w", connectMaxAttempts, pingErr)
}

// WithTx menjalankan fn di dalam satu database transaction. Jika fn
// mengembalikan error, seluruh perubahan di-rollback sehingga operasi
// (contoh: validasi kategori + insert transaksi) bersifat atomik.
func WithTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("memulai transaksi: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return fmt.Errorf("%w (rollback gagal: %v)", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaksi: %w", err)
	}

	return nil
}
