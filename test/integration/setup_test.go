package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"monetry-api/app/repository"
	"monetry-api/app/service"
	"monetry-api/config"
	"monetry-api/database"
	"monetry-api/helper"
	"monetry-api/route"
)

type testEnv struct {
	t   *testing.T
	app *fiber.App
	db  *sql.DB
}

// setupTestEnv merakit aplikasi lengkap (route + middleware + service) yang
// terhubung ke PostgreSQL sungguhan. Integration test dilewati otomatis bila
// TEST_DATABASE_URL belum diset.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL belum diset, integration test dilewati")
	}

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(pingCtx), "database test tidak dapat diakses")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	require.NoError(t, database.Migrate(ctx, db, os.DirFS("../.."), "migrations", logger))

	cfg := &config.Config{
		AppName:          "Monetry Test",
		AppEnv:           "test",
		AppPort:          "0",
		JWTSecret:        "integration-test-secret-key-monetry",
		JWTAccessExpiry:  15 * time.Minute,
		JWTRefreshExpiry: time.Hour,
		RateLimitMax:     10_000,
	}

	jwtManager := helper.NewJWTManager(helper.JWTConfig{
		Secret:        cfg.JWTSecret,
		AccessExpiry:  cfg.JWTAccessExpiry,
		RefreshExpiry: cfg.JWTRefreshExpiry,
	})

	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	budgetRepo := repository.NewBudgetRepository(db)

	app := config.NewFiberApp(cfg, logger)
	route.Setup(app, route.Handlers{
		Auth:        service.NewAuthService(userRepo, tokenRepo, jwtManager, db, logger),
		User:        service.NewUserService(userRepo),
		Category:    service.NewCategoryService(categoryRepo, logger),
		Transaction: service.NewTransactionService(transactionRepo, categoryRepo, db, logger),
		Budget:      service.NewBudgetService(budgetRepo, categoryRepo, logger),
		Dashboard:   service.NewDashboardService(transactionRepo),
		JWT:         jwtManager,
	})

	return &testEnv{t: t, app: app, db: db}
}

type apiClient struct {
	t     *testing.T
	app   *fiber.App
	token string
}

func (e *testEnv) client() *apiClient {
	return &apiClient{t: e.t, app: e.app}
}

func (c *apiClient) request(method, path string, body any) (int, map[string]any) {
	c.t.Helper()

	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(c.t, err)
		payload = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, payload)
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	if c.token != "" {
		req.Header.Set(fiber.HeaderAuthorization, "Bearer "+c.token)
	}

	resp, err := c.app.Test(req, -1)
	require.NoError(c.t, err)
	defer resp.Body.Close()

	var decoded map[string]any
	require.NoError(c.t, json.NewDecoder(resp.Body).Decode(&decoded),
		"response harus berupa JSON")

	return resp.StatusCode, decoded
}

func dataOf(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()
	data, ok := payload["data"].(map[string]any)
	require.True(t, ok, "response tidak memiliki object data: %v", payload)
	return data
}

func idOf(t *testing.T, payload map[string]any) int64 {
	t.Helper()
	raw, ok := dataOf(t, payload)["id"].(float64)
	require.True(t, ok, "response tidak memiliki id: %v", payload)
	return int64(raw)
}

func cleanupUsers(t *testing.T, db *sql.DB, usernames ...string) {
	t.Helper()
	for _, username := range usernames {
		_, err := db.Exec(`DELETE FROM users WHERE username = $1`, username)
		require.NoError(t, err)
	}
}
