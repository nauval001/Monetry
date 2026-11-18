package middleware

import (
	"github.com/gofiber/fiber/v2"

	"monetry-api/helper"
)

// Auth memverifikasi access token JWT pada header Authorization.
func Auth(jwtManager *helper.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := BearerToken(c)
		if !ok {
			return helper.ErrUnauthorized
		}

		claims, err := jwtManager.ParseAccess(token)
		if err != nil {
			return helper.NewErrorWithCause(fiber.StatusUnauthorized, "token tidak valid atau sudah kedaluwarsa", err)
		}

		c.Locals(LocalsUserID, claims.UserID)
		c.Locals(LocalsUsername, claims.Username)
		c.Locals(LocalsRole, claims.Role)

		return c.Next()
	}
}

// RequireRole membatasi endpoint hanya untuk role tertentu (RBAC).
func RequireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(LocalsRole).(string)
		if _, ok := allowed[role]; !ok {
			return helper.ErrForbidden
		}
		return c.Next()
	}
}

// UserIDFromContext mengambil user ID yang sudah diverifikasi middleware.
func UserIDFromContext(c *fiber.Ctx) (int64, error) {
	userID, ok := c.Locals(LocalsUserID).(int64)
	if !ok || userID <= 0 {
		return 0, helper.ErrUnauthorized
	}
	return userID, nil
}

// RoleFromContext mengambil role user dari context.
func RoleFromContext(c *fiber.Ctx) string {
	role, _ := c.Locals(LocalsRole).(string)
	return role
}
