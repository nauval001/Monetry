package model

import "time"

// Role yang tersedia pada sistem (RBAC).
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// User adalah entitas user sesuai tabel users.
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// IsAdmin menandakan apakah user memiliki role admin.
func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// RegisterRequest adalah payload pendaftaran user baru.
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50,alphanum" example:"budi"`
	Email    string `json:"email" validate:"required,email,max=100" example:"budi@example.com"`
	Password string `json:"password" validate:"required,min=8,max=72" example:"Rahasia#123"`
}

// LoginRequest dapat menggunakan email atau username sebagai identifier.
type LoginRequest struct {
	Email    string `json:"email" validate:"omitempty,email,max=100" example:"budi@example.com"`
	Username string `json:"username" validate:"omitempty,min=3,max=50" example:""`
	Password string `json:"password" validate:"required,min=8,max=72" example:"Rahasia#123"`
}

// RefreshRequest adalah payload untuk menukar refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// TokenPair adalah pasangan token hasil login/refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int64  `json:"expires_in" example:"900"`
}

// UserResponse adalah representasi user yang aman untuk dikirim ke client.
type UserResponse struct {
	ID        int64     `json:"id" example:"1"`
	Username  string    `json:"username" example:"budi"`
	Email     string    `json:"email" example:"budi@example.com"`
	Role      string    `json:"role" example:"user"`
	IsActive  bool      `json:"is_active" example:"true"`
	CreatedAt time.Time `json:"created_at" example:"2026-11-18T10:00:00Z"`
}

// NewUserResponse memetakan entitas User menjadi UserResponse.
func NewUserResponse(user *User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	}
}

// LoginResponse adalah response login/refresh.
type LoginResponse struct {
	User UserResponse `json:"user"`
	TokenPair
}

// UserListQuery adalah filter daftar user untuk admin.
type UserListQuery struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Search string `json:"search"`
}
