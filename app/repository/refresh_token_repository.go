package repository

import (
	"context"
	"database/sql"

	"monetry-api/app/model"
)

// RefreshTokenRepository menyimpan & mencabut refresh token.
type RefreshTokenRepository interface {
	Create(ctx context.Context, tx DBTX, token *model.RefreshToken) error
	FindByJTI(ctx context.Context, tx DBTX, jti string) (*model.RefreshToken, error)
	Revoke(ctx context.Context, tx DBTX, jti string) error
	RevokeByUserID(ctx context.Context, tx DBTX, userID int64) error
	DeleteExpired(ctx context.Context, tx DBTX) (int64, error)
}

type refreshTokenRepository struct {
	db *sql.DB
}

// NewRefreshTokenRepository membuat implementasi RefreshTokenRepository.
func NewRefreshTokenRepository(db *sql.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, tx DBTX, token *model.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, jti, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	err := conn(tx, r.db).QueryRowContext(ctx, query,
		token.UserID, token.JTI, token.TokenHash, token.ExpiresAt,
	).Scan(&token.ID, &token.CreatedAt)

	return mapError(err)
}

func (r *refreshTokenRepository) FindByJTI(ctx context.Context, tx DBTX, jti string) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, jti, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE jti = $1`

	var token model.RefreshToken
	var revokedAt sql.NullTime

	err := conn(tx, r.db).QueryRowContext(ctx, query, jti).Scan(
		&token.ID, &token.UserID, &token.JTI, &token.TokenHash,
		&token.ExpiresAt, &revokedAt, &token.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}

	return &token, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, tx DBTX, jti string) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE jti = $1 AND revoked_at IS NULL`
	result, err := conn(tx, r.db).ExecContext(ctx, query, jti)
	if err != nil {
		return mapError(err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *refreshTokenRepository) RevokeByUserID(ctx context.Context, tx DBTX, userID int64) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := conn(tx, r.db).ExecContext(ctx, query, userID)
	return mapError(err)
}

func (r *refreshTokenRepository) DeleteExpired(ctx context.Context, tx DBTX) (int64, error) {
	query := `DELETE FROM refresh_tokens WHERE expires_at < NOW() OR revoked_at IS NOT NULL`
	result, err := conn(tx, r.db).ExecContext(ctx, query)
	if err != nil {
		return 0, mapError(err)
	}
	return result.RowsAffected()
}
