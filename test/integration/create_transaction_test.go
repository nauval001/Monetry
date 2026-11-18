package integration

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPassword = "Rahasia#123"

func (c *apiClient) registerAndLogin(t *testing.T, username, email string) string {
	t.Helper()

	status, payload := c.request(http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username": username,
		"email":    email,
		"password": testPassword,
	})
	require.Equal(t, http.StatusCreated, status, "register harus berhasil: %v", payload)

	status, payload = c.request(http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": testPassword,
	})
	require.Equal(t, http.StatusOK, status, "login harus berhasil: %v", payload)

	token, ok := dataOf(t, payload)["access_token"].(string)
	require.True(t, ok, "response login tidak memiliki access_token")

	return token
}

// TestCreateTransactionIntegration memenuhi kebutuhan integration test
// endpoint POST /api/v1/transactions (PRD bagian 9 poin 2).
//
// Jalankan dengan PostgreSQL aktif:
//
//	$env:TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/monetry?sslmode=disable"
//	go test ./test/integration -run TestCreateTransactionIntegration -v
func TestCreateTransactionIntegration(t *testing.T) {
	env := setupTestEnv(t)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	ownerUsername := "itowner" + suffix
	otherUsername := "itother" + suffix
	t.Cleanup(func() { cleanupUsers(t, env.db, ownerUsername, otherUsername) })

	client := env.client()

	// 1. Daftar & login sebagai pemilik data.
	ownerToken := client.registerAndLogin(t, ownerUsername, ownerUsername+"@example.com")
	client.token = ownerToken

	// 2. Buat kategori pengeluaran.
	status, payload := client.request(http.MethodPost, "/api/v1/categories", map[string]any{
		"name": "Makan Harian",
		"type": "expense",
	})
	require.Equal(t, http.StatusCreated, status, "kategori harus berhasil dibuat: %v", payload)
	categoryID := idOf(t, payload)

	// 3. Catat transaksi (skenario utama).
	status, payload = client.request(http.MethodPost, "/api/v1/transactions", map[string]any{
		"category_id":      categoryID,
		"amount":           75000,
		"description":      "Makan siang",
		"transaction_date": "2026-11-18",
	})
	require.Equal(t, http.StatusCreated, status, "transaksi harus berhasil dicatat: %v", payload)

	transaction := dataOf(t, payload)
	require.NotZero(t, transaction["id"])
	assert.InDelta(t, 75000, transaction["amount"].(float64), 0.001)
	assert.Equal(t, float64(categoryID), transaction["category_id"].(float64))
	assert.Equal(t, "expense", transaction["category_type"])
	assert.Equal(t, "2026-11-18", transaction["transaction_date"])
	assert.Equal(t, "Makan siang", transaction["description"])

	// 4. Nominal 0 ditolak validasi (amount harus > 0).
	status, payload = client.request(http.MethodPost, "/api/v1/transactions", map[string]any{
		"category_id": categoryID,
		"amount":      0,
	})
	assert.Equal(t, http.StatusUnprocessableEntity, status, "amount 0 harus ditolak: %v", payload)

	// 5. Ownership check: user lain tidak boleh memakai kategori milik orang lain.
	otherToken := client.registerAndLogin(t, otherUsername, otherUsername+"@example.com")
	client.token = otherToken

	status, payload = client.request(http.MethodPost, "/api/v1/transactions", map[string]any{
		"category_id": categoryID,
		"amount":      10000,
	})
	assert.Equal(t, http.StatusForbidden, status, "kategori milik user lain harus ditolak: %v", payload)

	// 6. Dashboard bulan berjalan mengikuti transaksi yang baru dibuat.
	client.token = ownerToken
	status, payload = client.request(http.MethodGet, "/api/v1/dashboard/summary?month=11&year=2026", nil)
	require.Equal(t, http.StatusOK, status, payload)

	summary := dataOf(t, payload)
	assert.InDelta(t, 75000, summary["total_expense"].(float64), 0.001)
	assert.InDelta(t, -75000, summary["balance"].(float64), 0.001)
	assert.InDelta(t, 1, summary["transaction_count"].(float64), 0.001)
}
