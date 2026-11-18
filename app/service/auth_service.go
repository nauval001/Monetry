package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"

	"monetry-api/app/model"
	"monetry-api/app/repository"
	"monetry-api/database"
	"monetry-api/helper"
)

// AuthService menangani autentikasi: register, login, refresh, dan logout.
type AuthService struct {
	users  repository.UserRepository
	tokens repository.RefreshTokenRepository
	jwt    *helper.JWTManager
	db     *sql.DB
	logger *slog.Logger
}

// NewAuthService membuat instance AuthService.
func NewAuthService(
	users repository.UserRepository,
	tokens repository.RefreshTokenRepository,
	jwtManager *helper.JWTManager,
	db *sql.DB,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{users: users, tokens: tokens, jwt: jwtManager, db: db, logger: logger}
}

// Register menangani POST /api/v1/auth/register.
//
// @Summary      Registrasi user baru
// @Description  Mendaftarkan akun baru. Password di-hash dengan bcrypt dan role default adalah "user".
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body model.RegisterRequest true "Data registrasi"
// @Success      201 {object} helper.Response{data=model.UserResponse} "Registrasi berhasil"
// @Failure      400 {object} helper.Response "Body tidak valid"
// @Failure      409 {object} helper.Response "Email atau username sudah terdaftar"
// @Failure      422 {object} helper.Response "Validasi gagal"
// @Router       /auth/register [post]
func (s *AuthService) Register(c *fiber.Ctx) error {
	var request model.RegisterRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.NewError(http.StatusBadRequest, "body permintaan tidak valid")
	}
	if validationErrors := helper.Validate(&request); validationErrors != nil {
		return helper.NewValidationError(validationErrors)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return helper.NewErrorWithCause(http.StatusInternalServerError, "gagal memproses password", err)
	}

	user := &model.User{
		Username: strings.TrimSpace(request.Username),
		Email:    strings.ToLower(strings.TrimSpace(request.Email)),
		Password: string(hash),
		Role:     model.RoleUser,
		IsActive: true,
	}

	if err := s.users.Create(c.Context(), nil, user); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.NewError(http.StatusConflict, "email atau username sudah terdaftar")
		}
		return translateRepoError(err, "")
	}

	s.logger.Info("user baru terdaftar", "user_id", user.ID, "username", user.Username)

	return helper.Success(c, http.StatusCreated, "registrasi berhasil", model.NewUserResponse(user))
}

// Login menangani POST /api/v1/auth/login.
//
// @Summary      Login user
// @Description  Login memakai email atau username. Menghasilkan access token (15 menit) dan refresh token (7 hari).
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body model.LoginRequest true "Kredensial login"
// @Success      200 {object} helper.Response{data=model.LoginResponse} "Login berhasil"
// @Failure      400 {object} helper.Response "Body tidak valid"
// @Failure      401 {object} helper.Response "Kredensial salah"
// @Failure      403 {object} helper.Response "Akun tidak aktif"
// @Failure      422 {object} helper.Response "Validasi gagal"
// @Router       /auth/login [post]
func (s *AuthService) Login(c *fiber.Ctx) error {
	var request model.LoginRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.NewError(http.StatusBadRequest, "body permintaan tidak valid")
	}
	if validationErrors := helper.Validate(&request); validationErrors != nil {
		return helper.NewValidationError(validationErrors)
	}

	identifier := strings.TrimSpace(request.Email)
	if identifier == "" {
		identifier = strings.TrimSpace(request.Username)
	}
	if identifier == "" {
		return helper.NewError(http.StatusBadRequest, "email atau username wajib diisi")
	}

	user, err := s.users.FindByIdentifier(c.Context(), nil, identifier)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NewError(http.StatusUnauthorized, "email/username atau password salah")
		}
		return translateRepoError(err, "")
	}

	if !user.IsActive {
		return helper.NewError(http.StatusForbidden, "akun tidak aktif")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)); err != nil {
		return helper.NewError(http.StatusUnauthorized, "email/username atau password salah")
	}

	tokenPair, err := s.issueTokens(c.Context(), nil, user)
	if err != nil {
		return err
	}

	return helper.Success(c, http.StatusOK, "login berhasil", model.LoginResponse{
		User:      model.NewUserResponse(user),
		TokenPair: tokenPair,
	})
}

// issueTokens membuat access + refresh token lalu menyimpan hash refresh token
// pada database (di dalam tx bila tersedia).
func (s *AuthService) issueTokens(ctx context.Context, tx repository.DBTX, user *model.User) (model.TokenPair, error) {
	var pair model.TokenPair

	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Username, user.Role)
	if err != nil {
		return pair, helper.NewErrorWithCause(http.StatusInternalServerError, "gagal membuat access token", err)
	}

	refreshToken, jti, expiresAt, err := s.jwt.GenerateRefreshToken(user.ID, user.Username, user.Role)
	if err != nil {
		return pair, helper.NewErrorWithCause(http.StatusInternalServerError, "gagal membuat refresh token", err)
	}

	stored := &model.RefreshToken{
		UserID:    user.ID,
		JTI:       jti,
		TokenHash: helper.HashToken(refreshToken),
		ExpiresAt: expiresAt,
	}
	if err := s.tokens.Create(ctx, tx, stored); err != nil {
		return pair, translateRepoError(err, "")
	}

	return model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwt.AccessExpiry().Seconds()),
	}, nil
}

// Refresh menangani POST /api/v1/auth/refresh (rotasi refresh token).
//
// @Summary      Rotasi token
// @Description  Menukar refresh token lama dengan pasangan access & refresh token baru. Refresh token lama langsung dicabut (token rotation).
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body model.RefreshRequest true "Refresh token lama"
// @Success      200 {object} helper.Response{data=model.LoginResponse} "Token berhasil dirotasi"
// @Failure      400 {object} helper.Response "Body tidak valid"
// @Failure      401 {object} helper.Response "Refresh token tidak valid / sudah dicabut"
// @Failure      422 {object} helper.Response "Validasi gagal"
// @Router       /auth/refresh [post]
func (s *AuthService) Refresh(c *fiber.Ctx) error {
	var request model.RefreshRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.NewError(http.StatusBadRequest, "body permintaan tidak valid")
	}
	if validationErrors := helper.Validate(&request); validationErrors != nil {
		return helper.NewValidationError(validationErrors)
	}

	claims, err := s.jwt.ParseRefresh(request.RefreshToken)
	if err != nil {
		return helper.ErrInvalidToken
	}

	var response model.LoginResponse

	txErr := database.WithTx(c.Context(), s.db, func(tx *sql.Tx) error {
		stored, err := s.tokens.FindByJTI(c.Context(), tx, claims.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return helper.ErrInvalidToken
			}
			return translateRepoError(err, "")
		}

		if stored.UserID != claims.UserID ||
			stored.TokenHash != helper.HashToken(request.RefreshToken) ||
			!stored.IsUsable() {
			return helper.ErrInvalidToken
		}

		user, err := s.users.FindByID(c.Context(), tx, claims.UserID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return helper.ErrInvalidToken
			}
			return translateRepoError(err, "")
		}
		if !user.IsActive {
			return helper.NewError(http.StatusForbidden, "akun tidak aktif")
		}

		// Rotasi: refresh token lama dicabut, lalu terbitkan pasangan baru.
		if err := s.tokens.Revoke(c.Context(), tx, claims.ID); err != nil {
			return translateRepoError(err, "")
		}

		pair, err := s.issueTokens(c.Context(), tx, user)
		if err != nil {
			return err
		}

		response = model.LoginResponse{User: model.NewUserResponse(user), TokenPair: pair}
		return nil
	})
	if txErr != nil {
		return txErr
	}

	s.logger.Info("refresh token dirotasi", "user_id", response.User.ID)

	return helper.Success(c, http.StatusOK, "token berhasil diperbarui", response)
}

// Logout menangani POST /api/v1/auth/logout.
//
// @Summary      Logout
// @Description  Mencabut refresh token yang dikirim sehingga tidak bisa dipakai lagi.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body model.RefreshRequest true "Refresh token yang akan dicabut"
// @Success      200 {object} helper.Response "Logout berhasil"
// @Failure      400 {object} helper.Response "Body tidak valid"
// @Failure      401 {object} helper.Response "Token tidak valid"
// @Failure      403 {object} helper.Response "Token bukan milik user yang login"
// @Router       /auth/logout [post]
func (s *AuthService) Logout(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	var request model.RefreshRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.NewError(http.StatusBadRequest, "body permintaan tidak valid")
	}
	if validationErrors := helper.Validate(&request); validationErrors != nil {
		return helper.NewValidationError(validationErrors)
	}

	claims, err := s.jwt.ParseRefresh(request.RefreshToken)
	if err != nil {
		return helper.ErrInvalidToken
	}
	if claims.UserID != userID {
		return helper.ErrForbidden
	}

	if err := s.tokens.Revoke(c.Context(), nil, claims.ID); err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return translateRepoError(err, "")
		}
	}

	s.logger.Info("user logout", "user_id", userID)

	return helper.Success(c, http.StatusOK, "logout berhasil", nil)
}
