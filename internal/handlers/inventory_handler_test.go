package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
)

// resetInventoryData resets inventory-related data using database
func resetInventoryData(t *testing.T) {
	resetAdminData(t)

	// Add additional product for deleted status test
	_ = productRepo.Create(&models.Product{
		Name:        "deleted product",
		Description: "已删除商品",
		Category:    "其他",
		Price:       99.00,
		Stock:       5,
		Status:      "deleted",
		Images:      []string{},
	})
}

func TestListInventoryLogs_Handler(t *testing.T) {
	resetInventoryData(t)

	// Create some inventory logs by adjusting inventory
	req := httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust", strings.NewReader(`{"productId":1,"quantity":10,"reason":"测试入库"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	AdjustInventory(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Query inventory logs
	req = httptest.NewRequest(http.MethodGet, "/api/admin/inventory/logs", nil)
	w = httptest.NewRecorder()

	ListInventoryLogs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.Inventory
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	// At least one log should exist
	assert.True(t, len(response) >= 1)
}

func TestGetInventoryAlerts_Handler(t *testing.T) {
	resetInventoryData(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/inventory/alerts", nil)
	w := httptest.NewRecorder()

	GetInventoryAlerts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.InventoryAlert
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	// Product 2 has stock 8, below threshold 10
	assert.True(t, len(response) >= 1)
}

func TestAdjustInventory_Handler(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "increase stock",
			requestBody:    `{"productId":1,"quantity":10,"reason":"补货"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "decrease stock",
			requestBody:    `{"productId":1,"quantity":-5,"reason":"盘点损耗"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "non-existent product",
			requestBody:    `{"productId":999,"quantity":10}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetInventoryData(t)
			req := httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			AdjustInventory(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var product models.Product
				err := json.NewDecoder(w.Body).Decode(&product)
				assert.NoError(t, err)
			}
		})
	}
}

func TestAdjustInventory_NegativeStock(t *testing.T) {
	resetInventoryData(t)

	// Test stock cannot become negative
	req := httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust",
		strings.NewReader(`{"productId":2,"quantity":-100,"reason":"测试负数"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	AdjustInventory(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var product models.Product
	err := json.NewDecoder(w.Body).Decode(&product)
	assert.NoError(t, err)
	// Stock cannot be less than 0
	assert.Equal(t, 0, product.Stock)
}
