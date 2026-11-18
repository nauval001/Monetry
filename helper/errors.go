package helper

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError adalah error yang membawa status HTTP dan pesan siap tampil
// ke client. Service layer mengembalikan error ini agar handler cukup
// memanggil helper.Fail.
type AppError struct {
	Status  int
	Message string
	Details any
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

// NewError membuat AppError baru.
func NewError(status int, message string) *AppError {
	return &AppError{Status: status, Message: message}
}

// NewErrorWithCause membuat AppError yang tetap menyimpan error asal
// (dipakai untuk logging internal tanpa membocorkan detail ke client).
func NewErrorWithCause(status int, message string, cause error) *AppError {
	return &AppError{Status: status, Message: message, Err: cause}
}

// NewValidationError membuat error 422 dengan detail per-field.
func NewValidationError(details any) *AppError {
	return &AppError{
		Status:  http.StatusUnprocessableEntity,
		Message: "validasi data gagal",
		Details: details,
	}
}

// Sentinel error yang sering dipakai lintas layer.
var (
	ErrBadRequest      = NewError(http.StatusBadRequest, "permintaan tidak valid")
	ErrUnauthorized    = NewError(http.StatusUnauthorized, "autentikasi diperlukan")
	ErrInvalidToken    = NewError(http.StatusUnauthorized, "token tidak valid atau sudah kedaluwarsa")
	ErrForbidden       = NewError(http.StatusForbidden, "anda tidak memiliki akses ke resource ini")
	ErrNotFound        = NewError(http.StatusNotFound, "data tidak ditemukan")
	ErrConflict        = NewError(http.StatusConflict, "data sudah ada")
	ErrInternal        = NewError(http.StatusInternalServerError, "terjadi kesalahan pada server")
	ErrTooManyRequests = NewError(http.StatusTooManyRequests, "terlalu banyak permintaan")
)

// AsAppError memeriksa apakah err (atau error yang dibungkusnya) adalah AppError.
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
