package model

import "time"

// Tipe kategori yang diizinkan.
const (
	CategoryTypeIncome  = "income"
	CategoryTypeExpense = "expense"
)

// Category adalah entitas kategori milik user.
type Category struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateCategoryRequest adalah payload pembuatan kategori.
type CreateCategoryRequest struct {
	Name string `json:"name" validate:"required,min=2,max=50" example:"Makan Harian"`
	Type string `json:"type" validate:"required,oneof=income expense" example:"expense"`
}

// UpdateCategoryRequest adalah payload perubahan kategori.
type UpdateCategoryRequest struct {
	Name string `json:"name" validate:"required,min=2,max=50" example:"Transportasi"`
	Type string `json:"type" validate:"required,oneof=income expense" example:"expense"`
}

// CategoryFilter adalah filter opsional pada daftar kategori.
type CategoryFilter struct {
	Type   string
	Search string
}
