package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"monetry-api/app/model"
)

// CategoryRepository mendefinisikan operasi persistence untuk kategori.
type CategoryRepository interface {
	Create(ctx context.Context, tx DBTX, category *model.Category) error
	FindByID(ctx context.Context, tx DBTX, id int64) (*model.Category, error)
	FindByIDForShare(ctx context.Context, tx DBTX, id int64) (*model.Category, error)
	FindByUserAndName(ctx context.Context, tx DBTX, userID int64, name string, excludeID int64) (*model.Category, error)
	ListByUser(ctx context.Context, tx DBTX, userID int64, filter model.CategoryFilter, limit, offset int) ([]model.Category, int64, error)
	Update(ctx context.Context, tx DBTX, category *model.Category) error
	Delete(ctx context.Context, tx DBTX, id int64) error
	CountTransactions(ctx context.Context, tx DBTX, categoryID int64) (int64, error)
}

type categoryRepository struct {
	db *sql.DB
}

// NewCategoryRepository membuat implementasi CategoryRepository.
func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

const categoryColumns = `id, user_id, name, type, created_at`

func (r *categoryRepository) Create(ctx context.Context, tx DBTX, category *model.Category) error {
	query := `
		INSERT INTO categories (user_id, name, type)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	err := conn(tx, r.db).QueryRowContext(ctx, query,
		category.UserID, category.Name, category.Type,
	).Scan(&category.ID, &category.CreatedAt)

	return mapError(err)
}

func (r *categoryRepository) FindByID(ctx context.Context, tx DBTX, id int64) (*model.Category, error) {
	query := `SELECT ` + categoryColumns + ` FROM categories WHERE id = $1`
	return scanCategory(conn(tx, r.db).QueryRowContext(ctx, query, id))
}

// FindByIDForShare mengambil kategori dengan row-level lock (FOR SHARE).
// Dipakai di dalam transaction agar kategori tidak terhapus saat transaksi
// baru sedang divalidasi.
func (r *categoryRepository) FindByIDForShare(ctx context.Context, tx DBTX, id int64) (*model.Category, error) {
	query := `SELECT ` + categoryColumns + ` FROM categories WHERE id = $1 FOR SHARE`
	return scanCategory(conn(tx, r.db).QueryRowContext(ctx, query, id))
}

func (r *categoryRepository) FindByUserAndName(ctx context.Context, tx DBTX, userID int64, name string, excludeID int64) (*model.Category, error) {
	query := `SELECT ` + categoryColumns + ` FROM categories
		WHERE user_id = $1 AND LOWER(name) = LOWER($2) AND ($3 = 0 OR id <> $3)
		LIMIT 1`
	return scanCategory(conn(tx, r.db).QueryRowContext(ctx, query, userID, name, excludeID))
}

func (r *categoryRepository) ListByUser(ctx context.Context, tx DBTX, userID int64, filter model.CategoryFilter, limit, offset int) ([]model.Category, int64, error) {
	db := conn(tx, r.db)

	conditions := []string{"user_id = $1"}
	args := []any{userID}

	if filter.Type != "" {
		args = append(args, filter.Type)
		conditions = append(conditions, fmt.Sprintf("type = $%d", len(args)))
	}
	if filter.Search != "" {
		args = append(args, filter.Search)
		conditions = append(conditions, fmt.Sprintf("name ILIKE '%%' || $%d || '%%'", len(args)))
	}

	where := " WHERE " + strings.Join(conditions, " AND ")

	var total int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories`+where, args...).Scan(&total); err != nil {
		return nil, 0, mapError(err)
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	listQuery := `SELECT ` + categoryColumns + ` FROM categories` + where +
		fmt.Sprintf(` ORDER BY id DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)

	rows, err := db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, mapError(err)
	}
	defer rows.Close()

	categories := make([]model.Category, 0, limit)
	for rows.Next() {
		category, err := scanCategory(rows)
		if err != nil {
			return nil, 0, err
		}
		categories = append(categories, *category)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapError(err)
	}

	return categories, total, nil
}

func (r *categoryRepository) Update(ctx context.Context, tx DBTX, category *model.Category) error {
	query := `UPDATE categories SET name = $1, type = $2 WHERE id = $3`
	result, err := conn(tx, r.db).ExecContext(ctx, query, category.Name, category.Type, category.ID)
	if err != nil {
		return mapError(err)
	}
	return ensureAffected(result)
}

func (r *categoryRepository) Delete(ctx context.Context, tx DBTX, id int64) error {
	result, err := conn(tx, r.db).ExecContext(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	return ensureAffected(result)
}

func (r *categoryRepository) CountTransactions(ctx context.Context, tx DBTX, categoryID int64) (int64, error) {
	var total int64
	query := `SELECT COUNT(*) FROM transactions WHERE category_id = $1`
	if err := conn(tx, r.db).QueryRowContext(ctx, query, categoryID).Scan(&total); err != nil {
		return 0, mapError(err)
	}
	return total, nil
}

func scanCategory(row rowScanner) (*model.Category, error) {
	var category model.Category
	err := row.Scan(&category.ID, &category.UserID, &category.Name, &category.Type, &category.CreatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &category, nil
}

func ensureAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
