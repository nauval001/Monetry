package helper

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
	// Gunakan nama field dari tag json agar pesan error konsisten
	// dengan contract API.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return fld.Name
		}
		return name
	})
}

// Validate memvalidasi struct berdasarkan tag `validate`.
// Mengembalikan map field -> pesan error, atau nil jika valid.
func Validate(payload any) map[string]string {
	if payload == nil {
		return nil
	}

	err := validate.Struct(payload)
	if err == nil {
		return nil
	}

	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string]string{"payload": "payload tidak valid"}
	}

	result := make(map[string]string, len(errs))
	for _, fieldErr := range errs {
		result[fieldErr.Field()] = validationMessage(fieldErr)
	}
	return result
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		return "minimal " + fe.Param()
	case "max":
		return "maksimal " + fe.Param()
	case "gt":
		return "harus lebih besar dari " + fe.Param()
	case "gte":
		return "harus lebih besar atau sama dengan " + fe.Param()
	case "lt":
		return "harus lebih kecil dari " + fe.Param()
	case "lte":
		return "harus lebih kecil atau sama dengan " + fe.Param()
	case "oneof":
		return "harus salah satu dari: " + fe.Param()
	case "datetime":
		return "format tanggal harus " + fe.Param()
	case "alphanum":
		return "hanya boleh berisi huruf dan angka"
	case "numeric":
		return "harus berupa angka"
	case "uuid":
		return "format UUID tidak valid"
	default:
		return "nilai tidak valid"
	}
}
