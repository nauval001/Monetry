package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strings"
)

// Migrate menjalankan seluruh file .sql di dalam folder migration secara
// berurutan (berdasarkan nama file) dan mencatat versi yang sudah dijalankan
// pada tabel schema_migrations sehingga aman dieksekusi berulang kali.
func Migrate(ctx context.Context, db *sql.DB, files fs.FS, dir string, logger *slog.Logger) error {
	const createTable = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`
	if _, err := db.ExecContext(ctx, createTable); err != nil {
		return fmt.Errorf("membuat tabel schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(files, dir)
	if err != nil {
		return fmt.Errorf("membaca folder migration: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	applied := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version := entry.Name()

		var exists bool
		err := db.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`,
			version,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("memeriksa status migration %s: %w", version, err)
		}
		if exists {
			continue
		}

		content, err := fs.ReadFile(files, path.Join(dir, version))
		if err != nil {
			return fmt.Errorf("membaca file migration %s: %w", version, err)
		}

		err = WithTx(ctx, db, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, string(content)); err != nil {
				return fmt.Errorf("menjalankan migration %s: %w", version, err)
			}
			_, err := tx.ExecContext(ctx,
				`INSERT INTO schema_migrations (version) VALUES ($1)`, version,
			)
			return err
		})
		if err != nil {
			return err
		}

		applied++
		logger.Info("migration diterapkan", "version", version)
	}

	logger.Info("migration selesai", "applied", applied, "total_files", len(entries))
	return nil
}
