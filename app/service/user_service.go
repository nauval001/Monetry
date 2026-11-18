package service

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"monetry-api/app/model"
	"monetry-api/app/repository"
	"monetry-api/helper"
)

// UserService menangani data user yang sedang login & moderasi oleh admin.
type UserService struct {
	users repository.UserRepository
}

// NewUserService membuat instance UserService.
func NewUserService(users repository.UserRepository) *UserService {
	return &UserService{users: users}
}

// Me menangani GET /api/v1/auth/me.
//
// @Summary      Profil user yang sedang login
// @Description  Mengembalikan data user beserta role berdasarkan access token.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} helper.Response{data=model.UserResponse} "Profil user"
// @Failure      401 {object} helper.Response "Token tidak valid"
// @Failure      404 {object} helper.Response "User tidak ditemukan"
// @Router       /auth/me [get]
func (s *UserService) Me(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	user, err := s.users.FindByID(c.Context(), nil, userID)
	if err != nil {
		return translateRepoError(err, "user tidak ditemukan")
	}

	return helper.Success(c, http.StatusOK, "profil user berhasil diambil", model.NewUserResponse(user))
}

// ListUsers menangani GET /api/v1/users (khusus admin).
//
// @Summary      Daftar seluruh user (Admin)
// @Description  Menampilkan seluruh user pada sistem dengan paginasi dan pencarian username/email. Hanya bisa diakses role admin.
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        page   query int    false "Halaman (default 1)"
// @Param        limit  query int    false "Jumlah data per halaman (default 10, maks 100)"
// @Param        search query string false "Pencarian username atau email"
// @Success      200 {object} helper.Response{data=[]model.UserResponse} "Daftar user"
// @Failure      401 {object} helper.Response "Token tidak valid"
// @Failure      403 {object} helper.Response "Bukan admin"
// @Router       /users [get]
func (s *UserService) ListUsers(c *fiber.Ctx) error {
	page, limit := helper.ParsePagination(c)

	query := model.UserListQuery{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
	}

	users, total, err := s.users.FindAll(c.Context(), nil, query)
	if err != nil {
		return translateRepoError(err, "")
	}

	responses := make([]model.UserResponse, 0, len(users))
	for i := range users {
		responses = append(responses, model.NewUserResponse(&users[i]))
	}

	return helper.SuccessWithMeta(c, http.StatusOK, "daftar user berhasil diambil",
		responses, helper.PaginationMeta(page, limit, total))
}
