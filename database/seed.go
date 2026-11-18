package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"

	"monetry-api/app/model"
	"monetry-api/config"
)

// SeedAdmin membuat akun admin pertama dari environment SEED_ADMIN_*
// sehingga RBAC bisa langsung dicoba setelah `docker compose up`.
func SeedAdmin(ctx context.Context, db *sql.DB, cfg *config.Config, logger *slog.Logger) error {
	if cfg.SeedAdminPassword == "" {
		logger.Warn("SEED_ADMIN_PASSWORD kosong, seeding admin dilewati")
		return nil
	}

	var existingID int64
	err := db.QueryRowContext(ctx,
		`SELECT id FROM users WHERE email = $1 OR username = $2 LIMIT 1`,
		cfg.SeedAdminEmail, cfg.SeedAdminUsername,
	).Scan(&existingID)
	if err == nil {
		logger.Info("akun admin sudah tersedia, seeding dilewati", "user_id", existingID)
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("memeriksa akun admin: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.SeedAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password admin: %w", err)
	}

	var id int64
	err = db.QueryRowContext(ctx,
		`INSERT INTO users (username, email, password, role) VALUES ($1, $2, $3, $4) RETURNING id`,
		cfg.SeedAdminUsername, cfg.SeedAdminEmail, string(hash), model.RoleAdmin,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("membuat akun admin: %w", err)
	}

	logger.Info("akun admin berhasil dibuat", "user_id", id, "email", cfg.SeedAdminEmail)
	return nil
}
