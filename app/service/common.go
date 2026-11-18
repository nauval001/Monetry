package service

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"monetry-api/app/repository"
	"monetry-api/helper"
	"monetry-api/middleware"
)

// translateRepoError memetakan error dari repository menjadi AppError
// sehingga handler/service lain cukup mengembalikan error tersebut.
func translateRepoError(err error, notFoundMessage string) error {
	if err == nil {
		return nil
	}

	if appErr, ok := helper.AsAppError(err); ok {
		return appErr
	}

	switch {
	case errors.Is(err, repository.ErrNotFound):
		if notFoundMessage == "" {
			notFoundMessage = "data tidak ditemukan"
		}
		return helper.NewError(http.StatusNotFound, notFoundMessage)
	case errors.Is(err, repository.ErrDuplicate):
		return helper.NewError(http.StatusConflict, "data sudah ada")
	case errors.Is(err, repository.ErrInUse):
		return helper.NewError(http.StatusConflict, "data masih digunakan oleh data lain")
	default:
		return helper.NewErrorWithCause(http.StatusInternalServerError, "terjadi kesalahan pada server", err)
	}
}

// idParam membaca parameter path `:id` dan memastikan nilainya valid.
func idParam(c *fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, helper.NewError(http.StatusBadRequest, "id tidak valid")
	}
	return id, nil
}

// userIDFromContext mengambil user ID hasil verifikasi JWT middleware.
func userIDFromContext(c *fiber.Ctx) (int64, error) {
	return middleware.UserIDFromContext(c)
}
