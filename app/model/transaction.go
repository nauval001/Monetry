package model

import "time"

// Transaction adalah entitas transaksi beserta informasi kategori.
type Transaction struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	CategoryID      int64     `json:"category_id"`
	CategoryName    string    `json:"category_name"`
	CategoryType    string    `json:"category_type"`
	Amount          float64   `json:"amount"`
	Description     string    `json:"description"`
	TransactionDate Date      `json:"transaction_date"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateTransactionRequest adalah payload pencatatan transaksi.
type CreateTransactionRequest struct {
	CategoryID      int64   `json:"category_id" validate:"required,gt=0" example:"1"`
	Amount          float64 `json:"amount" validate:"gt=0" example:"75000"`
	Description     string  `json:"description" validate:"max=500" example:"Makan siang"`
	TransactionDate string  `json:"transaction_date" validate:"omitempty,datetime=2006-01-02" example:"2026-11-18"`
}

// UpdateTransactionRequest adalah payload perubahan transaksi.
// Field yang dikirim saja yang akan diubah.
type UpdateTransactionRequest struct {
	CategoryID      *int64   `json:"category_id" validate:"omitempty,gt=0" example:"2"`
	Amount          *float64 `json:"amount" validate:"omitempty,gt=0" example:"50000"`
	Description     *string  `json:"description" validate:"omitempty,max=500" example:"Makan malam"`
	TransactionDate *string  `json:"transaction_date" validate:"omitempty,datetime=2006-01-02" example:"2026-11-19"`
}

// TransactionFilter adalah filter pada daftar transaksi.
// CursorDate & CursorID diisi jika client mengirim parameter cursor.
type TransactionFilter struct {
	UserID     int64
	Month      int
	Year       int
	Type       string
	CategoryID int64
	Limit      int
	CursorDate *time.Time
	CursorID   int64
}
