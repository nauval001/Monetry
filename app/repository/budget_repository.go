package repository

import (
	"context"
	"database/sql"

	"monetry-api/app/model"
)

// BudgetRepository mendefinisikan operasi persistence budget bulanan.
type BudgetRepository interface {
	Upsert(ctx context.Context, tx DBTX, budget *model.Budget) error
	FindByID(ctx context.Context, tx DBTX, id int64) (*model.Budget, error)
	FindByCategoryPeriod(ctx context.Context, tx DBTX, userID, categoryID int64, month, year int) (*model.Budget, error)
	ListStatusByPeriod(ctx context.Context, tx DBTX, userID int64, month, year int) ([]model.BudgetStatus, error)
}

type budgetRepository struct {
	db *sql.DB
}

// NewBudgetRepository membuat implementasi BudgetRepository.
func NewBudgetRepository(db *sql.DB) BudgetRepository {
	return &budgetRepository{db: db}
}

// Upsert menetapkan budget baru atau memperbarui nominal budget yang sudah ada
// pada kombinasi (user, kategori, bulan, tahun) yang sama.
func (r *budgetRepository) Upsert(ctx context.Context, tx DBTX, budget *model.Budget) error {
	query := `
		INSERT INTO budgets (user_id, category_id, amount, month, year)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, category_id, month, year)
		DO UPDATE SET amount = EXCLUDED.amount
		RETURNING id`

	err := conn(tx, r.db).QueryRowContext(ctx, query,
		budget.UserID, budget.CategoryID, budget.Amount, budget.Month, budget.Year,
	).Scan(&budget.ID)

	return mapError(err)
}

func (r *budgetRepository) FindByID(ctx context.Context, tx DBTX, id int64) (*model.Budget, error) {
	query := `
		SELECT b.id, b.user_id, b.category_id, c.name, b.amount, b.month, b.year
		FROM budgets b
		JOIN categories c ON c.id = b.category_id
		WHERE b.id = $1`

	var budget model.Budget
	err := conn(tx, r.db).QueryRowContext(ctx, query, id).Scan(
		&budget.ID, &budget.UserID, &budget.CategoryID, &budget.CategoryName,
		&budget.Amount, &budget.Month, &budget.Year,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &budget, nil
}

func (r *budgetRepository) FindByCategoryPeriod(ctx context.Context, tx DBTX, userID, categoryID int64, month, year int) (*model.Budget, error) {
	query := `
		SELECT b.id, b.user_id, b.category_id, c.name, b.amount, b.month, b.year
		FROM budgets b
		JOIN categories c ON c.id = b.category_id
		WHERE b.user_id = $1 AND b.category_id = $2 AND b.month = $3 AND b.year = $4`

	var budget model.Budget
	err := conn(tx, r.db).QueryRowContext(ctx, query, userID, categoryID, month, year).Scan(
		&budget.ID, &budget.UserID, &budget.CategoryID, &budget.CategoryName,
		&budget.Amount, &budget.Month, &budget.Year,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &budget, nil
}

// ListStatusByPeriod mengambil seluruh budget user pada periode tertentu
// beserta total pengeluaran per kategori (untuk menghitung sisa budget).
func (r *budgetRepository) ListStatusByPeriod(ctx context.Context, tx DBTX, userID int64, month, year int) ([]model.BudgetStatus, error) {
	query := `
		SELECT b.id, b.category_id, c.name, b.amount, b.month, b.year,
		       COALESCE(SUM(t.amount), 0) AS spent_amount
		FROM budgets b
		JOIN categories c ON c.id = b.category_id
		LEFT JOIN transactions t
		       ON t.category_id = b.category_id
		      AND t.user_id = b.user_id
		      AND EXTRACT(MONTH FROM t.transaction_date) = b.month
		      AND EXTRACT(YEAR FROM t.transaction_date) = b.year
		WHERE b.user_id = $1 AND b.month = $2 AND b.year = $3
		GROUP BY b.id, b.category_id, c.name, b.amount, b.month, b.year
		ORDER BY c.name ASC`

	rows, err := conn(tx, r.db).QueryContext(ctx, query, userID, month, year)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	statuses := make([]model.BudgetStatus, 0)
	for rows.Next() {
		var budget model.Budget
		var spent float64

		err := rows.Scan(
			&budget.ID, &budget.CategoryID, &budget.CategoryName,
			&budget.Amount, &budget.Month, &budget.Year, &spent,
		)
		if err != nil {
			return nil, mapError(err)
		}

		statuses = append(statuses, model.NewBudgetStatus(&budget, spent))
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}

	return statuses, nil
}
