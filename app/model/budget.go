package model

import "math"

// BudgetListResponse adalah response daftar budget beserta agregatnya.
type BudgetListResponse struct {
	Month          int            `json:"month"`
	Year           int            `json:"year"`
	TotalBudget    float64        `json:"total_budget"`
	TotalSpent     float64        `json:"total_spent"`
	TotalRemaining float64        `json:"total_remaining"`
	Budgets        []BudgetStatus `json:"budgets"`
}

// NewBudgetListResponse membentuk response daftar budget sekaligus
// menghitung total budget, total pemakaian, dan total sisa.
func NewBudgetListResponse(month, year int, statuses []BudgetStatus) BudgetListResponse {
	response := BudgetListResponse{
		Month:   month,
		Year:    year,
		Budgets: statuses,
	}

	for _, status := range statuses {
		response.TotalBudget += status.BudgetAmount
		response.TotalSpent += status.SpentAmount
		response.TotalRemaining += status.RemainingAmount
	}

	response.TotalBudget = round2(response.TotalBudget)
	response.TotalSpent = round2(response.TotalSpent)
	response.TotalRemaining = round2(response.TotalRemaining)

	return response
}

// Budget adalah entitas budget bulanan per kategori.
type Budget struct {
	ID           int64   `json:"id"`
	UserID       int64   `json:"user_id"`
	CategoryID   int64   `json:"category_id"`
	CategoryName string  `json:"category_name,omitempty"`
	Amount       float64 `json:"amount"`
	Month        int     `json:"month"`
	Year         int     `json:"year"`
}

// SetBudgetRequest adalah payload penetapan budget bulanan.
type SetBudgetRequest struct {
	CategoryID int64   `json:"category_id" validate:"required,gt=0" example:"1"`
	Amount     float64 `json:"amount" validate:"required,gt=0" example:"1500000"`
	Month      int     `json:"month" validate:"required,min=1,max=12" example:"11"`
	Year       int     `json:"year" validate:"required,min=2000,max=2100" example:"2026"`
}

// BudgetStatus adalah hasil kalkulasi sisa budget pada suatu periode.
type BudgetStatus struct {
	BudgetID        int64   `json:"budget_id"`
	CategoryID      int64   `json:"category_id"`
	CategoryName    string  `json:"category_name"`
	Month           int     `json:"month"`
	Year            int     `json:"year"`
	BudgetAmount    float64 `json:"budget_amount"`
	SpentAmount     float64 `json:"spent_amount"`
	RemainingAmount float64 `json:"remaining_amount"`
	UsagePercentage float64 `json:"usage_percentage"`
	IsExceeded      bool    `json:"is_exceeded"`
}

// ComputeBudgetUsage menghitung sisa budget dan persentase pemakaian.
//
//	remaining  = budget - pengeluaran
//	percentage = pengeluaran / budget * 100
//
// Nilai dibulatkan ke 2 desimal dan persentase tidak pernah negatif.
func ComputeBudgetUsage(budgetAmount, spentAmount float64) (remaining float64, percentage float64) {
	remaining = round2(budgetAmount - spentAmount)
	if budgetAmount > 0 {
		percentage = round2(spentAmount / budgetAmount * 100)
	}
	if percentage < 0 {
		percentage = 0
	}
	return remaining, percentage
}

// NewBudgetStatus membentuk BudgetStatus lengkap dari data budget & realisasi.
func NewBudgetStatus(budget *Budget, spentAmount float64) BudgetStatus {
	remaining, percentage := ComputeBudgetUsage(budget.Amount, spentAmount)

	return BudgetStatus{
		BudgetID:        budget.ID,
		CategoryID:      budget.CategoryID,
		CategoryName:    budget.CategoryName,
		Month:           budget.Month,
		Year:            budget.Year,
		BudgetAmount:    round2(budget.Amount),
		SpentAmount:     round2(spentAmount),
		RemainingAmount: remaining,
		UsagePercentage: percentage,
		IsExceeded:      remaining < 0,
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
