package service

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"monetry-api/app/repository"
	"monetry-api/helper"
)

// DashboardService menyediakan ringkasan keuangan user.
type DashboardService struct {
	transactions repository.TransactionRepository
}

// NewDashboardService membuat instance DashboardService.
func NewDashboardService(transactions repository.TransactionRepository) *DashboardService {
	return &DashboardService{transactions: transactions}
}

// Summary menangani GET /api/v1/dashboard/summary.
//
// @Summary      Ringkasan dashboard
// @Description  Mengembalikan total pemasukan, total pengeluaran, dan saldo bersih (selisih) pada bulan berjalan. Parameter month/year opsional untuk periode lain.
// @Tags         Dashboard
// @Produce      json
// @Security     BearerAuth
// @Param        month query int false "Bulan (1-12), default bulan berjalan"
// @Param        year  query int false "Tahun, default tahun berjalan"
// @Success      200 {object} helper.Response{data=model.DashboardSummary} "Ringkasan keuangan"
// @Failure      401 {object} helper.Response "Token tidak valid"
// @Router       /dashboard/summary [get]
func (s *DashboardService) Summary(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	month, year := helper.ParseMonthYear(c)

	summary, err := s.transactions.SummaryByPeriod(c.Context(), nil, userID, month, year)
	if err != nil {
		return translateRepoError(err, "")
	}

	return helper.Success(c, http.StatusOK, "ringkasan berhasil diambil", summary)
}
