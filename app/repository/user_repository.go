package repository

import (
	"context"
	"database/sql"

	"monetry-api/app/model"
)

// UserRepository mendefinisikan operasi persistence untuk user.
type UserRepository interface {
	Create(ctx context.Context, tx DBTX, user *model.User) error
	FindByID(ctx context.Context, tx DBTX, id int64) (*model.User, error)
	FindByIdentifier(ctx context.Context, tx DBTX, identifier string) (*model.User, error)
	FindAll(ctx context.Context, tx DBTX, query model.UserListQuery) ([]model.User, int64, error)
}

type userRepository struct {
	db *sql.DB
}

// NewUserRepository membuat implementasi UserRepository.
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

const userColumns = `id, username, email, password, role, is_active, created_at`

func (r *userRepository) Create(ctx context.Context, tx DBTX, user *model.User) error {
	query := `
		INSERT INTO users (username, email, password, role, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	err := conn(tx, r.db).QueryRowContext(ctx, query,
		user.Username, user.Email, user.Password, user.Role, user.IsActive,
	).Scan(&user.ID, &user.CreatedAt)

	return mapError(err)
}

func (r *userRepository) FindByID(ctx context.Context, tx DBTX, id int64) (*model.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	return scanUser(conn(tx, r.db).QueryRowContext(ctx, query, id))
}

func (r *userRepository) FindByIdentifier(ctx context.Context, tx DBTX, identifier string) (*model.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE email = $1 OR username = $1 LIMIT 1`
	return scanUser(conn(tx, r.db).QueryRowContext(ctx, query, identifier))
}

func (r *userRepository) FindAll(ctx context.Context, tx DBTX, query model.UserListQuery) ([]model.User, int64, error) {
	db := conn(tx, r.db)

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM users
		WHERE ($1 = '' OR username ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%')`
	if err := db.QueryRowContext(ctx, countQuery, query.Search).Scan(&total); err != nil {
		return nil, 0, mapError(err)
	}

	listQuery := `
		SELECT ` + userColumns + `
		FROM users
		WHERE ($1 = '' OR username ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%')
		ORDER BY id ASC
		LIMIT $2 OFFSET $3`

	rows, err := db.QueryContext(ctx, listQuery, query.Search, query.Limit, (query.Page-1)*query.Limit)
	if err != nil {
		return nil, 0, mapError(err)
	}
	defer rows.Close()

	users := make([]model.User, 0, query.Limit)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapError(err)
	}

	return users, total, nil
}

func scanUser(row rowScanner) (*model.User, error) {
	var user model.User
	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.Role, &user.IsActive, &user.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &user, nil
}
