package service

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"monetry-api/app/model"
	"monetry-api/app/repository"
	"monetry-api/helper"
)

// CategoryService menangani CRUD kategori milik user.
type CategoryService struct {
	categories repository.CategoryRepository
	logger     *slog.Logger
}

// NewCategoryService membuat instance CategoryService.
func NewCategoryService(categories repository.CategoryRepository, logger *slog.Logger) *CategoryService {
	return &CategoryService{categories: categories, logger: logger}
}

// Create menangani POST /api/v1/categories.
//
// @Summary      Membuat kategori
// @Description  Membuat kategori baru (income/expense) milik user yang sedang login. Nama kategori unik per user.
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body model.CreateCategoryRequest true "Data kategori"
// @Success      201 {object} helper.Response{data=model.Category} "Kategori dibuat"
// @Failure      400 {object} helper.Response "Body tidak valid"
// @Failure      401 {object} helper.Response "Token tidak valid"
// @Failure      409 {object} helper.Response "Nama kategori sudah ada"
// @Failure      422 {object} helper.Response "Validasi gagal"
// @Router       /categories [post]
func (s *CategoryService) Create(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	var request model.CreateCategoryRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.NewError(http.StatusBadRequest, "body permintaan tidak valid")
	}
	if validationErrors := helper.Validate(&request); validationErrors != nil {
		return helper.NewValidationError(validationErrors)
	}

	name := strings.TrimSpace(request.Name)

	existing, err := s.categories.FindByUserAndName(c.Context(), nil, userID, name, 0)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return translateRepoError(err, "")
	}
	if existing != nil {
		return helper.NewError(http.StatusConflict, "kategori dengan nama tersebut sudah ada")
	}

	category := &model.Category{
		UserID: userID,
		Name:   name,
		Type:   request.Type,
	}

	if err := s.categories.Create(c.Context(), nil, category); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.NewError(http.StatusConflict, "kategori dengan nama tersebut sudah ada")
		}
		return translateRepoError(err, "")
	}

	s.logger.Info("kategori dibuat", "user_id", userID, "category_id", category.ID)

	return helper.Success(c, http.StatusCreated, "kategori berhasil dibuat", category)
}

// List menangani GET /api/v1/categories.
//
// @Summary      Daftar kategori
// @Description  Menampilkan kategori milik user yang sedang login dengan paginasi, filter tipe, dan pencarian nama.
// @Tags         Categories
// @Produce      json
// @Security     BearerAuth
// @Param        page   query int    false "Halaman (default 1)"
// @Param        limit  query int    false "Jumlah data per halaman (default 10, maks 100)"
// @Param        type   query string false "Filter tipe" Enums(income, expense)
// @Param        search query string false "Pencarian nama kategori"
// @Success      200 {object} helper.Response{data=[]model.Category} "Daftar kategori"
// @Failure      401 {object} helper.Response "Token tidak valid"
// @Router       /categories [get]
func (s *CategoryService) List(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	page, limit := helper.ParsePagination(c)

	filter := model.CategoryFilter{
		Type:   c.Query("type"),
		Search: strings.TrimSpace(c.Query("search")),
	}
	if filter.Type != "" &&
		filter.Type != model.CategoryTypeIncome &&
		filter.Type != model.CategoryTypeExpense {
		return helper.NewError(http.StatusBadRequest, "tipe harus income atau expense")
	}

	categories, total, err := s.categories.ListByUser(
		c.Context(), nil, userID, filter, limit, helper.Offset(page, limit),
	)
	if err != nil {
		return translateRepoError(err, "")
	}

	return helper.SuccessWithMeta(c, http.StatusOK, "daftar kategori berhasil diambil",
		categories, helper.PaginationMeta(page, limit, total))
}

// Update menangani PUT /api/v1/categories/:id.
//
// @Summary      Mengubah kategori
// @Description  Mengubah nama dan/atau tipe kategori milik user. Ownership check dilakukan di service layer.
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int                          true "ID kategori"
// @Param        request body model.UpdateCategoryRequest true "Data kategori"
// @Success      200 {object} helper.Response{data=model.Category} "Kategori diperbarui"
// @Failure      400 {object} helper.Response "ID/body tidak valid"
// @Failure      403 {object} helper.Response "Kategori bukan milik user"
// @Failure      404 {object} helper.Response "Kategori tidak ditemukan"
// @Failure      409 {object} helper.Response "Nama kategori sudah dipakai"
// @Router       /categories/{id} [put]
func (s *CategoryService) Update(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}
	categoryID, err := idParam(c)
	if err != nil {
		return err
	}

	var request model.UpdateCategoryRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.NewError(http.StatusBadRequest, "body permintaan tidak valid")
	}
	if validationErrors := helper.Validate(&request); validationErrors != nil {
		return helper.NewValidationError(validationErrors)
	}

	category, err := s.categories.FindByID(c.Context(), nil, categoryID)
	if err != nil {
		return translateRepoError(err, "kategori tidak ditemukan")
	}
	if category.UserID != userID {
		return helper.ErrForbidden
	}

	name := strings.TrimSpace(request.Name)

	duplicate, err := s.categories.FindByUserAndName(c.Context(), nil, userID, name, categoryID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return translateRepoError(err, "")
	}
	if duplicate != nil {
		return helper.NewError(http.StatusConflict, "kategori dengan nama tersebut sudah ada")
	}

	category.Name = name
	category.Type = request.Type

	if err := s.categories.Update(c.Context(), nil, category); err != nil {
		return translateRepoError(err, "")
	}

	s.logger.Info("kategori diperbarui", "user_id", userID, "category_id", category.ID)

	return helper.Success(c, http.StatusOK, "kategori berhasil diperbarui", category)
}

// Delete menangani DELETE /api/v1/categories/:id.
//
// @Summary      Menghapus kategori
// @Description  Menghapus kategori milik user. Kategori yang masih memiliki transaksi tidak bisa dihapus (restriksi relasi).
// @Tags         Categories
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID kategori"
// @Success      200 {object} helper.Response "Kategori dihapus"
// @Failure      400 {object} helper.Response "ID tidak valid"
// @Failure      403 {object} helper.Response "Kategori bukan milik user"
// @Failure      404 {object} helper.Response "Kategori tidak ditemukan"
// @Failure      409 {object} helper.Response "Kategori masih memiliki transaksi"
// @Router       /categories/{id} [delete]
func (s *CategoryService) Delete(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}
	categoryID, err := idParam(c)
	if err != nil {
		return err
	}

	category, err := s.categories.FindByID(c.Context(), nil, categoryID)
	if err != nil {
		return translateRepoError(err, "kategori tidak ditemukan")
	}
	if category.UserID != userID {
		return helper.ErrForbidden
	}

	transactionCount, err := s.categories.CountTransactions(c.Context(), nil, categoryID)
	if err != nil {
		return translateRepoError(err, "")
	}
	if transactionCount > 0 {
		return helper.NewError(http.StatusConflict,
			"kategori tidak dapat dihapus karena masih memiliki transaksi")
	}

	if err := s.categories.Delete(c.Context(), nil, categoryID); err != nil {
		if errors.Is(err, repository.ErrInUse) {
			return helper.NewError(http.StatusConflict,
				"kategori tidak dapat dihapus karena masih digunakan oleh data lain")
		}
		return translateRepoError(err, "kategori tidak ditemukan")
	}

	s.logger.Info("kategori dihapus", "user_id", userID, "category_id", categoryID)

	return helper.Success(c, http.StatusOK, "kategori berhasil dihapus", nil)
}
