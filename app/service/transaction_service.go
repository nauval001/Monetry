package service

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"monetry-api/app/model"
	"monetry-api/app/repository"
	"monetry-api/database"
	"monetry-api/helper"
)

// TransactionService menangani pencatatan transaksi pemasukan/pengeluaran.
type TransactionService struct {
	transactions repository.TransactionRepository
	categories   repository.CategoryRepository
	db           *sql.DB
	logger       *slog.Logger
}

// NewTransactionService membuat instance TransactionService.
func NewTransactionService(
	transactions repository.TransactionRepository,
	categories repository.CategoryRepository,
	db *sql.DB,
	logger *slog.Logger,
) *TransactionService {
	return &TransactionService{
		transactions: transactions,
		categories:   categories,
		db:           db,
		logger:       logger,
	}
}

// Create menangani POST /api/v1/transactions.
//
// @Summary      Mencatat transaksi
// @Description  Mencatat pemasukan/pengeluaran. Validasi kepemilikan kategori dan insert transaksi dijalankan dalam satu database transaction (WithTx) sehingga atomik. Nominal wajib > 0.
// @Tags         Transactions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body model.CreateTransactionRequest true "Data transaksi"
// @Success      201 {object} helper.Response{data=model.Transaction} "Transaksi dibuat"
// @Failure      400 {object} helper.Response "Body tidak valid"
// @Failure      401 {object} helper.Response "Token tidak valid"
// @Failure      403 {object} helper.Response "Kategori bukan milik user"
// @Failure      404 {object} helper.Response "Kategori tidak ditemukan"
// @Failure      422 {object} helper.Response "Validasi gagal (amount harus > 0)"
// @Router       /transactions [post]
func (s *TransactionService) Create(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	var request model.CreateTransactionRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.NewError(http.StatusBadRequest, "body permintaan tidak valid")
	}
	if validationErrors := helper.Validate(&request); validationErrors != nil {
		return helper.NewValidationError(validationErrors)
	}

	transactionDate := model.NewDate(time.Now().UTC())
	if request.TransactionDate != "" {
		parsed, err := time.ParseInLocation(model.DateLayout, request.TransactionDate, time.UTC)
		if err != nil {
			return helper.NewError(http.StatusBadRequest, "format transaction_date harus YYYY-MM-DD")
		}
		transactionDate = model.NewDate(parsed)
	}

	transaction := &model.Transaction{
		UserID:          userID,
		CategoryID:      request.CategoryID,
		Amount:          request.Amount,
		Description:     strings.TrimSpace(request.Description),
		TransactionDate: transactionDate,
	}

	// WithTx: validasi kategori (row lock FOR SHARE) + insert transaksi
	// berjalan atomik. Jika salah satu gagal, keduanya dibatalkan.
	txErr := database.WithTx(c.Context(), s.db, func(tx *sql.Tx) error {
		category, err := s.categories.FindByIDForShare(c.Context(), tx, transaction.CategoryID)
		if err != nil {
			return translateRepoError(err, "kategori tidak ditemukan")
		}
		if category.UserID != userID {
			return helper.NewError(http.StatusForbidden, "kategori bukan milik anda")
		}

		transaction.CategoryName = category.Name
		transaction.CategoryType = category.Type

		if err := s.transactions.Create(c.Context(), tx, transaction); err != nil {
			return translateRepoError(err, "")
		}

		return nil
	})
	if txErr != nil {
		return txErr
	}

	s.logger.Info("transaksi dibuat",
		"user_id", userID,
		"transaction_id", transaction.ID,
		"category_id", transaction.CategoryID,
		"amount", transaction.Amount,
	)

	return helper.Success(c, http.StatusCreated, "transaksi berhasil dicatat", transaction)
}

// List menangani GET /api/v1/transactions.
//
// @Summary      Daftar transaksi
// @Description  Menampilkan riwayat transaksi user dengan filter bulan/tahun, tipe, kategori, serta cursor-based pagination.
// @Tags         Transactions
// @Produce      json
// @Security     BearerAuth
// @Param        limit       query int    false "Jumlah data per halaman (default 10, maks 100)"
// @Param        cursor      query string false "Cursor halaman berikutnya (dari meta.next_cursor)"
// @Param        month       query int    false "Filter bulan (1-12)"
// @Param        year        query int    false "Filter tahun, wajib dikirim bersama month"
// @Param        type        query string false "Filter tipe" Enums(income, expense)
// @Param        category_id query int    false "Filter kategori"
// @Success      200 {object} helper.Response{data=[]model.Transaction} "Daftar transaksi"
// @Failure      400 {object} helper.Response "Parameter tidak valid"
// @Failure      401 {object} helper.Response "Token tidak valid"
// @Router       /transactions [get]
func (s *TransactionService) List(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	limit := helper.QueryInt(c, "limit", 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	filter := model.TransactionFilter{
		UserID:     userID,
		Limit:      limit,
		Type:       c.Query("type"),
		CategoryID: helper.QueryInt64(c, "category_id", 0),
	}

	if filter.Type != "" &&
		filter.Type != model.CategoryTypeIncome &&
		filter.Type != model.CategoryTypeExpense {
		return helper.NewError(http.StatusBadRequest, "type harus income atau expense")
	}

	month := helper.QueryInt(c, "month", 0)
	year := helper.QueryInt(c, "year", 0)
	if (month > 0) != (year > 0) {
		return helper.NewError(http.StatusBadRequest, "parameter month dan year harus dikirim bersamaan")
	}
	if month < 0 || month > 12 {
		return helper.NewError(http.StatusBadRequest, "month harus berada di antara 1 dan 12")
	}
	if year < 0 {
		return helper.NewError(http.StatusBadRequest, "year tidak valid")
	}
	if month > 0 && year > 0 {
		filter.Month = month
		filter.Year = year
	}

	if rawCursor := c.Query("cursor"); rawCursor != "" {
		cursorDate, cursorID, err := helper.DecodeCursor(rawCursor)
		if err != nil {
			return helper.NewError(http.StatusBadRequest, "cursor tidak valid")
		}
		filter.CursorDate = &cursorDate
		filter.CursorID = cursorID
	}

	transactions, hasMore, err := s.transactions.List(c.Context(), nil, filter)
	if err != nil {
		return translateRepoError(err, "")
	}

	nextCursor := ""
	if hasMore && len(transactions) > 0 {
		last := transactions[len(transactions)-1]
		nextCursor = helper.EncodeCursor(last.TransactionDate.Time(), last.ID)
	}

	return helper.SuccessWithMeta(c, http.StatusOK, "daftar transaksi berhasil diambil",
		transactions, helper.CursorMeta(limit, nextCursor, hasMore))
}

// Update menangani PUT /api/v1/transactions/:id.
//
// @Summary      Mengubah transaksi
// @Description  Mengubah transaksi milik user. Ownership check dilakukan di service layer dan seluruh perubahan dijalankan dalam satu database transaction.
// @Tags         Transactions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int                             true "ID transaksi"
// @Param        request body model.UpdateTransactionRequest true "Perubahan transaksi (field opsional)"
// @Success      200 {object} helper.Response{data=model.Transaction} "Transaksi diperbarui"
// @Failure      400 {object} helper.Response "ID/body/parameter tidak valid"
// @Failure      403 {object} helper.Response "Transaksi atau kategori bukan milik user"
// @Failure      404 {object} helper.Response "Transaksi/kategori tidak ditemukan"
// @Failure      422 {object} helper.Response "Validasi gagal"
// @Router       /transactions/{id} [put]
func (s *TransactionService) Update(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}
	transactionID, err := idParam(c)
	if err != nil {
		return err
	}

	var request model.UpdateTransactionRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.NewError(http.StatusBadRequest, "body permintaan tidak valid")
	}
	if validationErrors := helper.Validate(&request); validationErrors != nil {
		return helper.NewValidationError(validationErrors)
	}

	var updated *model.Transaction

	txErr := database.WithTx(c.Context(), s.db, func(tx *sql.Tx) error {
		current, err := s.transactions.FindByID(c.Context(), tx, transactionID)
		if err != nil {
			return translateRepoError(err, "transaksi tidak ditemukan")
		}
		// Ownership check: user hanya boleh mengubah datanya sendiri.
		if current.UserID != userID {
			return helper.ErrForbidden
		}

		if request.CategoryID != nil {
			category, err := s.categories.FindByIDForShare(c.Context(), tx, *request.CategoryID)
			if err != nil {
				return translateRepoError(err, "kategori tidak ditemukan")
			}
			if category.UserID != userID {
				return helper.NewError(http.StatusForbidden, "kategori bukan milik anda")
			}
			current.CategoryID = category.ID
			current.CategoryName = category.Name
			current.CategoryType = category.Type
		}

		if request.Amount != nil {
			current.Amount = *request.Amount
		}
		if request.Description != nil {
			current.Description = strings.TrimSpace(*request.Description)
		}
		if request.TransactionDate != nil {
			parsed, err := time.ParseInLocation(model.DateLayout, *request.TransactionDate, time.UTC)
			if err != nil {
				return helper.NewError(http.StatusBadRequest, "format transaction_date harus YYYY-MM-DD")
			}
			current.TransactionDate = model.NewDate(parsed)
		}

		if err := s.transactions.Update(c.Context(), tx, current); err != nil {
			return translateRepoError(err, "transaksi tidak ditemukan")
		}

		refreshed, err := s.transactions.FindByID(c.Context(), tx, transactionID)
		if err != nil {
			return translateRepoError(err, "transaksi tidak ditemukan")
		}
		updated = refreshed

		return nil
	})
	if txErr != nil {
		return txErr
	}

	s.logger.Info("transaksi diperbarui", "user_id", userID, "transaction_id", transactionID)

	return helper.Success(c, http.StatusOK, "transaksi berhasil diperbarui", updated)
}

// Delete menangani DELETE /api/v1/transactions/:id.
//
// @Summary      Menghapus transaksi
// @Description  Menghapus transaksi milik user. Ownership check dilakukan di dalam database transaction.
// @Tags         Transactions
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID transaksi"
// @Success      200 {object} helper.Response "Transaksi dihapus"
// @Failure      400 {object} helper.Response "ID tidak valid"
// @Failure      403 {object} helper.Response "Transaksi bukan milik user"
// @Failure      404 {object} helper.Response "Transaksi tidak ditemukan"
// @Router       /transactions/{id} [delete]
func (s *TransactionService) Delete(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}
	transactionID, err := idParam(c)
	if err != nil {
		return err
	}

	txErr := database.WithTx(c.Context(), s.db, func(tx *sql.Tx) error {
		current, err := s.transactions.FindByID(c.Context(), tx, transactionID)
		if err != nil {
			return translateRepoError(err, "transaksi tidak ditemukan")
		}
		if current.UserID != userID {
			return helper.ErrForbidden
		}

		if err := s.transactions.Delete(c.Context(), tx, transactionID); err != nil {
			return translateRepoError(err, "transaksi tidak ditemukan")
		}

		return nil
	})
	if txErr != nil {
		return txErr
	}

	s.logger.Info("transaksi dihapus", "user_id", userID, "transaction_id", transactionID)

	return helper.Success(c, http.StatusOK, "transaksi berhasil dihapus", nil)
}
