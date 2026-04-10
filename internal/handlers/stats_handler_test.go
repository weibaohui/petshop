package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"petshop/internal/db"
	"petshop/internal/models"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupStatsTestDB initializes the test database for stats tests
func setupStatsTestDB(t *testing.T) *sql.DB {
	database, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	schema := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		category TEXT NOT NULL,
		price REAL NOT NULL,
		stock INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'on_sale',
		images TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		total_amount REAL NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		refund_reason TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS order_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id INTEGER NOT NULL,
		product_id INTEGER NOT NULL,
		product_name TEXT NOT NULL,
		price REAL NOT NULL,
		quantity INTEGER NOT NULL,
		subtotal REAL NOT NULL
	);
	`
	_, err = database.Exec(schema)
	require.NoError(t, err)

	productRepo = db.NewProductRepositoryWithDB(database)
	orderRepo = db.NewOrderRepositoryWithDB(database)
	inventoryRepo = db.NewInventoryRepositoryWithDB(database)

	return database
}

// resetStatsData seeds test data for stats handlers.
func resetStatsData(t *testing.T) *sql.DB {
	dbConn := setupStatsTestDB(t)

	now := time.Now()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	// Order 1: today, paid
	require.NoError(t, orderRepo.Create(&models.Order{
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "Product A", Price: 100, Quantity: 1, Subtotal: 100},
		},
		TotalAmount: 100,
		Status:      "paid",
		CreatedAt:   now,
	}))

	// Order 2: today, shipped (should count)
	require.NoError(t, orderRepo.Create(&models.Order{
		UserID: 2,
		Products: []models.OrderItem{
			{ProductID: 2, ProductName: "Product B", Price: 200, Quantity: 2, Subtotal: 400},
		},
		TotalAmount: 400,
		Status:      "shipped",
		CreatedAt:   now,
	}))

	// Order 3: yesterday, delivered (should count)
	require.NoError(t, orderRepo.Create(&models.Order{
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "Product A", Price: 100, Quantity: 3, Subtotal: 300},
		},
		TotalAmount: 300,
		Status:      "delivered",
		CreatedAt:   now.AddDate(0, 0, -1),
	}))

	// Order 4: today, cancelled (should NOT count)
	require.NoError(t, orderRepo.Create(&models.Order{
		UserID: 3,
		Products: []models.OrderItem{
			{ProductID: 3, ProductName: "Product C", Price: 50, Quantity: 1, Subtotal: 50},
		},
		TotalAmount: 50,
		Status:      "cancelled",
		CreatedAt:   now,
	}))

	// Order 5: today, refunded (should NOT count)
	require.NoError(t, orderRepo.Create(&models.Order{
		UserID: 4,
		Products: []models.OrderItem{
			{ProductID: 2, ProductName: "Product B", Price: 200, Quantity: 1, Subtotal: 200},
		},
		TotalAmount: 200,
		Status:      "refunded",
		CreatedAt:   now,
	}))

	// Fix created_at in DB for orders to match exact dates (bypass auto-timestamp)
	_, err := dbConn.Exec("UPDATE orders SET created_at = ? WHERE id = 1", today)
	require.NoError(t, err)
	_, err = dbConn.Exec("UPDATE orders SET created_at = ? WHERE id = 2", today)
	require.NoError(t, err)
	_, err = dbConn.Exec("UPDATE orders SET created_at = ? WHERE id = 3", yesterday)
	require.NoError(t, err)
	_, err = dbConn.Exec("UPDATE orders SET created_at = ? WHERE id = 4", today)
	require.NoError(t, err)
	_, err = dbConn.Exec("UPDATE orders SET created_at = ? WHERE id = 5", today)
	require.NoError(t, err)

	return dbConn
}

// ==================== Unit tests for pure functions ====================

func TestCalculateDayStat(t *testing.T) {
	date := "2024-01-15"
	orders := []*models.Order{
		{ID: 1, TotalAmount: 100, Status: "paid", CreatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local)},
		{ID: 2, TotalAmount: 200, Status: "shipped", CreatedAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.Local)},
		{ID: 3, TotalAmount: 50, Status: "cancelled", CreatedAt: time.Date(2024, 1, 15, 12, 0, 0, 0, time.Local)},
		{ID: 4, TotalAmount: 300, Status: "refunded", CreatedAt: time.Date(2024, 1, 15, 13, 0, 0, 0, time.Local)},
		{ID: 5, TotalAmount: 400, Status: "delivered", CreatedAt: time.Date(2024, 1, 14, 10, 0, 0, 0, time.Local)},
	}

	stat := calculateDayStat(date, orders)

	assert.Equal(t, date, stat.Date)
	assert.Equal(t, 300.0, stat.TotalSales)
	assert.Equal(t, 2, stat.OrderCount)
}

func TestCalculatePeriodStat(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2024, 1, 5, 0, 0, 0, 0, time.Local)

	orders := []*models.Order{
		{ID: 1, TotalAmount: 100, Status: "paid", CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.Local)},
		{ID: 2, TotalAmount: 200, Status: "shipped", CreatedAt: time.Date(2024, 1, 5, 23, 59, 59, 0, time.Local)},
		{ID: 3, TotalAmount: 50, Status: "cancelled", CreatedAt: time.Date(2024, 1, 3, 10, 0, 0, 0, time.Local)},
		{ID: 4, TotalAmount: 300, Status: "refunded", CreatedAt: time.Date(2024, 1, 4, 10, 0, 0, 0, time.Local)},
		{ID: 5, TotalAmount: 400, Status: "delivered", CreatedAt: time.Date(2023, 12, 31, 23, 59, 59, 0, time.Local)},
		{ID: 6, TotalAmount: 500, Status: "pending", CreatedAt: time.Date(2024, 1, 6, 0, 0, 1, 0, time.Local)},
	}

	stat := calculatePeriodStat(start, end, orders)

	assert.Equal(t, "2024-01-01", stat.Date)
	assert.Equal(t, 300.0, stat.TotalSales)
	assert.Equal(t, 2, stat.OrderCount)
}

func TestGetWeekStart(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "Monday",
			input:    time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local), // Monday
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "Sunday",
			input:    time.Date(2024, 1, 21, 10, 0, 0, 0, time.Local), // Sunday
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),  // Previous Monday
		},
		{
			name:     "Wednesday",
			input:    time.Date(2024, 1, 17, 10, 0, 0, 0, time.Local), // Wednesday
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),  // Monday
		},
		{
			name:     "Saturday",
			input:    time.Date(2024, 1, 20, 10, 0, 0, 0, time.Local), // Saturday
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),  // Monday
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWeekStart(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== HTTP handler tests ====================

func TestGetSalesStats_Handler(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
	}{
		{
			name:           "get daily stats",
			queryString:    "?period=day",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "get weekly stats",
			queryString:    "?period=week",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "get monthly stats",
			queryString:    "?period=month",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "default period",
			queryString:    "",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid period returns error",
			queryString:    "?period=year",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetStatsData(t)
			req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales"+tt.queryString, nil)
			w := httptest.NewRecorder()

			GetSalesStats(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode >= 400 {
				return
			}

			var response []models.SalesStat
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)
			assert.NotNil(t, response)
		})
	}
}

func TestGetSalesStats_DatabaseError(t *testing.T) {
	database := setupStatsTestDB(t)
	database.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales", nil)
	w := httptest.NewRecorder()

	GetSalesStats(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetHotProducts_Handler(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
	}{
		{
			name:           "get hot products with default limit",
			queryString:    "",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "get hot products with custom limit",
			queryString:    "?limit=1",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "get hot products with invalid limit",
			queryString:    "?limit=abc",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetStatsData(t)
			req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products"+tt.queryString, nil)
			w := httptest.NewRecorder()

			GetHotProducts(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var response []models.HotProduct
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)
			assert.NotNil(t, response)
		})
	}
}

func TestGetHotProducts_DatabaseError(t *testing.T) {
	database := setupStatsTestDB(t)
	database.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetHotProducts_SortingAndLimit(t *testing.T) {
	resetStatsData(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products?limit=1", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Len(t, response, 1)
	// Product B has 2 items in order 2 (subtotal 400) vs Product A has 1+3=4 items (subtotal 100+300=400)
	// Order 5 is refunded so doesn't count. Product B has qty 2, Product A has qty 4 -> Product A should be first
	assert.Equal(t, int64(1), response[0].ProductID)
	assert.Equal(t, 4, response[0].SalesCount)
}

func TestGetHotProducts_ExcludesCancelledAndRefunded(t *testing.T) {
	resetStatsData(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Product C (ID 3) only appears in cancelled order, should not be present
	for _, hp := range response {
		assert.NotEqual(t, int64(3), hp.ProductID)
	}
}
