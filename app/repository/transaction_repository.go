package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"monetry-api/app/model"
)

// TransactionRepository mendefinisikan operasi persistence transaksi.
type TransactionRepository interface {
	Create(ctx context.Context, tx DBTX, transaction *model.Transaction) error
	FindByID(ctx context.Context, tx DBTX, id int64) (*model.Transaction, error)
	Update(ctx context.Context, tx DBTX, transaction *model.Transaction) error
	Delete(ctx context.Context, tx DBTX, id int64) error
	List(ctx context.Context, tx DBTX, filter model.TransactionFilter) ([]model.Transaction, bool, error)
	SumByTypeAndPeriod(ctx context.Context, tx DBTX, userID int64, transactionType string, month, year int) (float64, error)
	SummaryByPeriod(ctx context.Context, tx DBTX, userID int64, month, year int) (model.DashboardSummary, error)
}

type transactionRepository struct {
	db *sql.DB
}

// NewTransactionRepository membuat implementasi TransactionRepository.
func NewTransactionRepository(db *sql.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

const transactionSelect = `
	SELECT t.id, t.user_id, t.category_id, c.name, c.type, t.amount,
	       COALESCE(t.description, ''), t.transaction_date, t.created_at
	FROM transactions t
	JOIN categories c ON c.id = t.category_id`

func (r *transactionRepository) Create(ctx context.Context, tx DBTX, transaction *model.Transaction) error {
	query := `
		INSERT INTO transactions (user_id, category_id, amount, description, transaction_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	err := conn(tx, r.db).QueryRowContext(ctx, query,
		transaction.UserID, transaction.CategoryID, transaction.Amount,
		nullString(transaction.Description), transaction.TransactionDate.Time(),
	).Scan(&transaction.ID, &transaction.CreatedAt)

	return mapError(err)
}

func (r *transactionRepository) FindByID(ctx context.Context, tx DBTX, id int64) (*model.Transaction, error) {
	query := transactionSelect + ` WHERE t.id = $1`
	return scanTransaction(conn(tx, r.db).QueryRowContext(ctx, query, id))
}

func (r *transactionRepository) Update(ctx context.Context, tx DBTX, transaction *model.Transaction) error {
	query := `
		UPDATE transactions
		SET category_id = $1, amount = $2, description = $3, transaction_date = $4
		WHERE id = $5`

	result, err := conn(tx, r.db).ExecContext(ctx, query,
		transaction.CategoryID, transaction.Amount, nullString(transaction.Description),
		transaction.TransactionDate.Time(), transaction.ID,
	)
	if err != nil {
		return mapError(err)
	}
	return ensureAffected(result)
}

func (r *transactionRepository) Delete(ctx context.Context, tx DBTX, id int64) error {
	result, err := conn(tx, r.db).ExecContext(ctx, `DELETE FROM transactions WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	return ensureAffected(result)
}

// List mengambil transaksi dengan filter bulan/tahun/tipe/kategori dan
// cursor-based pagination. Query mengambil limit+1 baris untuk mengetahui
// apakah masih ada halaman berikutnya.
func (r *transactionRepository) List(ctx context.Context, tx DBTX, filter model.TransactionFilter) ([]model.Transaction, bool, error) {
	conditions := []string{"t.user_id = $1"}
	args := []any{filter.UserID}

	if filter.Month > 0 && filter.Year > 0 {
		args = append(args, filter.Month, filter.Year)
		conditions = append(conditions, fmt.Sprintf(
			"EXTRACT(MONTH FROM t.transaction_date) = $%d AND EXTRACT(YEAR FROM t.transaction_date) = $%d",
			len(args)-1, len(args),
		))
	}
	if filter.Type != "" {
		args = append(args, filter.Type)
		conditions = append(conditions, fmt.Sprintf("c.type = $%d", len(args)))
	}
	if filter.CategoryID > 0 {
		args = append(args, filter.CategoryID)
		conditions = append(conditions, fmt.Sprintf("t.category_id = $%d", len(args)))
	}
	if filter.CursorDate != nil && filter.CursorID > 0 {
		args = append(args, filter.CursorDate.Format(model.DateLayout), filter.CursorID)
		conditions = append(conditions, fmt.Sprintf(
			"(t.transaction_date, t.id) < ($%d::date, $%d)", len(args)-1, len(args),
		))
	}

	query := transactionSelect +
		" WHERE " + strings.Join(conditions, " AND ") +
		fmt.Sprintf(" ORDER BY t.transaction_date DESC, t.id DESC LIMIT $%d", len(args)+1)

	args = append(args, filter.Limit+1)

	rows, err := conn(tx, r.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, mapError(err)
	}
	defer rows.Close()

	transactions := make([]model.Transaction, 0, filter.Limit+1)
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return nil, false, err
		}
		transactions = append(transactions, *transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, false, mapError(err)
	}

	hasMore := len(transactions) > filter.Limit
	if hasMore {
		transactions = transactions[:filter.Limit]
	}

	return transactions, hasMore, nil
}

func (r *transactionRepository) SumByTypeAndPeriod(ctx context.Context, tx DBTX, userID int64, transactionType string, month, year int) (float64, error) {
	query := `
		SELECT COALESCE(SUM(t.amount), 0)
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		WHERE t.user_id = $1
		  AND c.type = $2
		  AND EXTRACT(MONTH FROM t.transaction_date) = $3
		  AND EXTRACT(YEAR FROM t.transaction_date) = $4`

	var total float64
	err := conn(tx, r.db).QueryRowContext(ctx, query, userID, transactionType, month, year).Scan(&total)
	return total, mapError(err)
}

func (r *transactionRepository) SummaryByPeriod(ctx context.Context, tx DBTX, userID int64, month, year int) (model.DashboardSummary, error) {
	summary := model.DashboardSummary{Month: month, Year: year}

	query := `
		SELECT
			COALESCE(SUM(CASE WHEN c.type = 'income' THEN t.amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN c.type = 'expense' THEN t.amount ELSE 0 END), 0) AS total_expense,
			COUNT(*) AS transaction_count
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		WHERE t.user_id = $1
		  AND EXTRACT(MONTH FROM t.transaction_date) = $2
		  AND EXTRACT(YEAR FROM t.transaction_date) = $3`

	err := conn(tx, r.db).QueryRowContext(ctx, query, userID, month, year).
		Scan(&summary.TotalIncome, &summary.TotalExpense, &summary.TransactionCount)
	if err != nil {
		return summary, mapError(err)
	}

	summary.Balance = model.ComputeBalance(summary.TotalIncome, summary.TotalExpense)
	return summary, nil
}

func scanTransaction(row rowScanner) (*model.Transaction, error) {
	var transaction model.Transaction
	err := row.Scan(
		&transaction.ID, &transaction.UserID, &transaction.CategoryID,
		&transaction.CategoryName, &transaction.CategoryType, &transaction.Amount,
		&transaction.Description, &transaction.TransactionDate, &transaction.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &transaction, nil
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
