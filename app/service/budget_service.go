package service

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"monetry-api/app/model"
	"monetry-api/app/repository"
	"monetry-api/helper"
)

// BudgetService menangani penetapan & pemantauan budget bulanan.
type BudgetService struct {
	budgets    repository.BudgetRepository
	categories repository.CategoryRepository
	logger     *slog.Logger
}

// NewBudgetService membuat instance BudgetService.
func NewBudgetService(
	budgets repository.BudgetRepository,
	categories repository.CategoryRepository,
	logger *slog.Logger,
) *BudgetService {
	return &BudgetService{budgets: budgets, categories: categories, logger: logger}
}

// Set menangani POST /api/v1/budgets.
//
// @Summary      Menetapkan budget bulanan
// @Description  Menetapkan atau memperbarui nominal budget untuk kategori pengeluaran pada bulan & tahun tertentu. Kombinasi (user, kategori, bulan, tahun) bersifat unik.
// @Tags         Budgets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body model.SetBudgetRequest true "Data budget"
// @Success      200 {object} helper.Response{data=model.Budget} "Budget disimpan"
// @Failure      400 {object} helper.Response "Body tidak valid / kategori bukan pengeluaran"
// @Failure      401 {object} helper.Response "Token tidak valid"
// @Failure      403 {object} helper.Response "Kategori bukan milik user"
// @Failure      404 {object} helper.Response "Kategori tidak ditemukan"
// @Failure      422 {object} helper.Response "Validasi gagal"
// @Router       /budgets [post]
func (s *BudgetService) Set(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	var request model.SetBudgetRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.NewError(http.StatusBadRequest, "body permintaan tidak valid")
	}
	if validationErrors := helper.Validate(&request); validationErrors != nil {
		return helper.NewValidationError(validationErrors)
	}

	category, err := s.categories.FindByID(c.Context(), nil, request.CategoryID)
	if err != nil {
		return translateRepoError(err, "kategori tidak ditemukan")
	}
	if category.UserID != userID {
		return helper.NewError(http.StatusForbidden, "kategori bukan milik anda")
	}
	if category.Type != model.CategoryTypeExpense {
		return helper.NewError(http.StatusBadRequest, "budget hanya dapat ditetapkan untuk kategori pengeluaran")
	}

	budget := &model.Budget{
		UserID:       userID,
		CategoryID:   category.ID,
		CategoryName: category.Name,
		Amount:       request.Amount,
		Month:        request.Month,
		Year:         request.Year,
	}

	if err := s.budgets.Upsert(c.Context(), nil, budget); err != nil {
		return translateRepoError(err, "")
	}

	s.logger.Info("budget ditetapkan",
		"user_id", userID,
		"budget_id", budget.ID,
		"category_id", budget.CategoryID,
		"month", budget.Month,
		"year", budget.Year,
	)

	return helper.Success(c, http.StatusOK, "budget berhasil disimpan", budget)
}

// List menangani GET /api/v1/budgets.
//
// @Summary      Status & sisa budget
// @Description  Menampilkan budget pada suatu periode beserta realisasi pengeluaran dan sisa budget per kategori. Sisa = budget - total pengeluaran kategori pada bulan tersebut. Default periode adalah bulan berjalan.
// @Tags         Budgets
// @Produce      json
// @Security     BearerAuth
// @Param        month query int false "Bulan (1-12), default bulan berjalan"
// @Param        year  query int false "Tahun, default tahun berjalan"
// @Success      200 {object} helper.Response{data=model.BudgetListResponse} "Status budget"
// @Failure      401 {object} helper.Response "Token tidak valid"
// @Router       /budgets [get]
func (s *BudgetService) List(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	month, year := helper.ParseMonthYear(c)

	statuses, err := s.budgets.ListStatusByPeriod(c.Context(), nil, userID, month, year)
	if err != nil {
		return translateRepoError(err, "")
	}

	response := model.NewBudgetListResponse(month, year, statuses)

	return helper.Success(c, http.StatusOK, "status budget berhasil diambil", response)
}
