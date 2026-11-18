package model

import "math"

// ComputeBalance menghitung saldo bersih (pemasukan - pengeluaran)
// dan membulatkannya ke 2 desimal.
func ComputeBalance(totalIncome, totalExpense float64) float64 {
	return math.Round((totalIncome-totalExpense)*100) / 100
}

// DashboardSummary adalah ringkasan keuangan pada bulan tertentu.
type DashboardSummary struct {
	Month            int     `json:"month" example:"11"`
	Year             int     `json:"year" example:"2026"`
	TotalIncome      float64 `json:"total_income" example:"5000000"`
	TotalExpense     float64 `json:"total_expense" example:"2750000"`
	Balance          float64 `json:"balance" example:"2250000"`
	TransactionCount int64   `json:"transaction_count" example:"24"`
}
