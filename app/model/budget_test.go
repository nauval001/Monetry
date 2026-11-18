package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"monetry-api/app/model"
)

// TestComputeBudgetUsage memenuhi kebutuhan unit test perhitungan sisa budget
// (PRD bagian 9 poin 2).
func TestComputeBudgetUsage(t *testing.T) {
	testCases := []struct {
		name           string
		budgetAmount   float64
		spentAmount    float64
		wantRemaining  float64
		wantPercentage float64
	}{
		{
			name:           "pemakaian sebagian",
			budgetAmount:   1_000_000,
			spentAmount:    250_000,
			wantRemaining:  750_000,
			wantPercentage: 25,
		},
		{
			name:           "budget terlampaui",
			budgetAmount:   1_000_000,
			spentAmount:    1_500_000,
			wantRemaining:  -500_000,
			wantPercentage: 150,
		},
		{
			name:           "belum ada pengeluaran",
			budgetAmount:   500_000,
			spentAmount:    0,
			wantRemaining:  500_000,
			wantPercentage: 0,
		},
		{
			name:           "nominal desimal dibulatkan dua angka",
			budgetAmount:   333_333.33,
			spentAmount:    111_111.11,
			wantRemaining:  222_222.22,
			wantPercentage: 33.33,
		},
		{
			name:           "budget nol tidak membagi nol",
			budgetAmount:   0,
			spentAmount:    0,
			wantRemaining:  0,
			wantPercentage: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			remaining, percentage := model.ComputeBudgetUsage(tc.budgetAmount, tc.spentAmount)

			assert.InDelta(t, tc.wantRemaining, remaining, 0.001, "sisa budget tidak sesuai")
			assert.InDelta(t, tc.wantPercentage, percentage, 0.001, "persentase pemakaian tidak sesuai")
		})
	}
}

// TestNewBudgetStatus memastikan status budget lengkap (termasuk flag
// terlampaui) terbentuk benar.
func TestNewBudgetStatus(t *testing.T) {
	budget := &model.Budget{
		ID:           7,
		CategoryID:   3,
		CategoryName: "Makan Harian",
		Amount:       800_000,
		Month:        11,
		Year:         2026,
	}

	t.Run("budget masih tersisa", func(t *testing.T) {
		status := model.NewBudgetStatus(budget, 300_000)

		require.Equal(t, int64(7), status.BudgetID)
		require.Equal(t, "Makan Harian", status.CategoryName)
		assert.InDelta(t, 500_000, status.RemainingAmount, 0.001)
		assert.InDelta(t, 37.5, status.UsagePercentage, 0.001)
		assert.False(t, status.IsExceeded)
	})

	t.Run("budget terlampaui", func(t *testing.T) {
		status := model.NewBudgetStatus(budget, 900_000)

		assert.InDelta(t, -100_000, status.RemainingAmount, 0.001)
		assert.InDelta(t, 112.5, status.UsagePercentage, 0.001)
		assert.True(t, status.IsExceeded)
	})
}

// TestNewBudgetListResponse memastikan agregat total budget, pemakaian,
// dan sisa dihitung dengan benar.
func TestNewBudgetListResponse(t *testing.T) {
	statuses := []model.BudgetStatus{
		{BudgetAmount: 500_000, SpentAmount: 200_000, RemainingAmount: 300_000},
		{BudgetAmount: 300_000, SpentAmount: 350_000, RemainingAmount: -50_000},
	}

	response := model.NewBudgetListResponse(11, 2026, statuses)

	assert.Equal(t, 11, response.Month)
	assert.Equal(t, 2026, response.Year)
	assert.InDelta(t, 800_000, response.TotalBudget, 0.001)
	assert.InDelta(t, 550_000, response.TotalSpent, 0.001)
	assert.InDelta(t, 250_000, response.TotalRemaining, 0.001)
	require.Len(t, response.Budgets, 2)
}
