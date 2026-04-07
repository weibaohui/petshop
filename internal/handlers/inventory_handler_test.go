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

// resetInventoryData 重置库存相关数据到初始状态
func resetInventoryData() {
	dataMu.Lock()
	defer dataMu.Unlock()

	// 重置产品
	products = make(map[int64]*models.Product)
	products[1] = &models.Product{
		ID:          1,
		Name:        "狗粮 10kg",
		Description: "优质狗粮",
		Category:    "狗粮",
		Price:       299.00,
		Stock:       50,
		Status:      "on_sale",
		Images:      []string{"/static/images/dog_food.jpg"},
	}
	products[2] = &models.Product{
		ID:          2,
		Name:        "猫粮 5kg",
		Description: "天然猫粮",
		Category:    "猫粮",
		Price:       199.00,
		Stock:       8,
		Status:      "on_sale",
		Images:      []string{"/static/images/cat_food.jpg"},
	}
	products[3] = &models.Product{
		ID:          3,
		Name:        "deleted product",
		Description: "已删除商品",
		Category:    "其他",
		Price:       99.00,
		Stock:       5,
		Status:      "deleted",
		Images:      []string{},
	}
	nextProductID = 4

	// 重置库存记录
	inventoryLogs = make([]models.Inventory, 0)
	nextInventoryID = 1

	// 重置库存阈值
	inventoryThreshold = 10
}

// ==================== ListInventoryLogs Tests ====================

func TestListInventoryLogsExtended(t *testing.T) {
	tests := []struct {
		name           string
		setupLogs      func()
		wantStatusCode int
		wantLen        int
		wantProductID  int64
		wantChangeType string
	}{
		{
			name:           "empty logs",
			setupLogs:      func() {},
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
		{
			name: "single log",
			setupLogs: func() {
				inventoryLogs = append(inventoryLogs, models.Inventory{
					ID:          1,
					ProductID:   1,
					ChangeType:  "in",
					Quantity:    50,
					BeforeStock: 0,
					AfterStock:  50,
					Reason:      "初始入库",
					Operator:    "admin",
				})
			},
			wantStatusCode: http.StatusOK,
			wantLen:        1,
			wantProductID:  1,
			wantChangeType: "in",
		},
		{
			name: "multiple logs",
			setupLogs: func() {
				inventoryLogs = append(inventoryLogs, models.Inventory{
					ID:          1,
					ProductID:   1,
					ChangeType:  "in",
					Quantity:    50,
					BeforeStock: 0,
					AfterStock:  50,
					Reason:      "初始入库",
					Operator:    "admin",
				})
				inventoryLogs = append(inventoryLogs, models.Inventory{
					ID:          2,
					ProductID:   2,
					ChangeType:  "out",
					Quantity:    5,
					BeforeStock: 10,
					AfterStock:  5,
					Reason:      "销售出库",
					Operator:    "system",
				})
				inventoryLogs = append(inventoryLogs, models.Inventory{
					ID:          3,
					ProductID:   1,
					ChangeType:  "adjust",
					Quantity:    2,
					BeforeStock: 50,
					AfterStock:  48,
					Reason:      "盘点调整",
					Operator:    "admin",
				})
			},
			wantStatusCode: http.StatusOK,
			wantLen:        3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetInventoryData()
			t.Cleanup(resetInventoryData)

			tt.setupLogs()

			req := httptest.NewRequest(http.MethodGet, "/api/admin/inventory/logs", nil)
			w := httptest.NewRecorder()

			ListInventoryLogs(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var response []models.Inventory
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Len(t, response, tt.wantLen)

			if tt.wantLen > 0 && tt.wantProductID > 0 {
				assert.Equal(t, tt.wantProductID, response[0].ProductID)
				assert.Equal(t, tt.wantChangeType, response[0].ChangeType)
			}
		})
	}
}

func TestListInventoryLogsWithAdjustInventory(t *testing.T) {
	resetInventoryData()
	t.Cleanup(resetInventoryData)

	// 先调整库存创建日志
	req := httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust",
		strings.NewReader(`{"productId":1,"quantity":10,"reason":"补货"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	AdjustInventory(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 再查询日志
	req = httptest.NewRequest(http.MethodGet, "/api/admin/inventory/logs", nil)
	w = httptest.NewRecorder()
	ListInventoryLogs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var logs []models.Inventory
	err := json.NewDecoder(w.Body).Decode(&logs)
	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, int64(1), logs[0].ProductID)
	assert.Equal(t, "in", logs[0].ChangeType)
	assert.Equal(t, 10, logs[0].Quantity)
	assert.Equal(t, 50, logs[0].BeforeStock)
	assert.Equal(t, 60, logs[0].AfterStock)
	assert.Equal(t, "补货", logs[0].Reason)
	assert.Equal(t, "admin", logs[0].Operator)
}

// ==================== GetInventoryAlerts Tests ====================

func TestGetInventoryAlertsExtended(t *testing.T) {
	tests := []struct {
		name             string
		setupProducts    func()
		threshold        int
		wantStatusCode   int
		wantLen          int
		wantProductID    int64
		wantProductName  string
		wantThreshold    int
		wantCurrentStock int
	}{
		{
			name: "no alerts when all stock above threshold",
			setupProducts: func() {
				products[1].Stock = 50
				products[2].Stock = 20
			},
			threshold:      10,
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
		{
			name: "single alert for low stock",
			setupProducts: func() {
				products[1].Stock = 50
				products[2].Stock = 8 // 低于阈值 10
			},
			threshold:        10,
			wantStatusCode:   http.StatusOK,
			wantLen:          1,
			wantProductID:    2,
			wantProductName:  "猫粮 5kg",
			wantThreshold:    10,
			wantCurrentStock: 8,
		},
		{
			name: "multiple alerts for low stock",
			setupProducts: func() {
				products[1].Stock = 5 // 低于阈值 10
				products[2].Stock = 8 // 低于阈值 10
			},
			threshold:      10,
			wantStatusCode: http.StatusOK,
			wantLen:        2,
		},
		{
			name: "alert at exact threshold boundary",
			setupProducts: func() {
				products[1].Stock = 10 // 等于阈值，应该触发
				products[2].Stock = 20 // 高于阈值，不触发
			},
			threshold:        10,
			wantStatusCode:   http.StatusOK,
			wantLen:          1,
			wantProductID:    1,
			wantProductName:  "狗粮 10kg",
			wantThreshold:    10,
			wantCurrentStock: 10,
		},
		{
			name: "no alert for deleted products",
			setupProducts: func() {
				// product 3 is deleted with stock 5
			},
			threshold:      10,
			wantStatusCode: http.StatusOK,
			wantLen:        1, // only product 2 (猫粮) has stock 8
		},
		{
			name: "custom threshold",
			setupProducts: func() {
				products[1].Stock = 50
				products[2].Stock = 8
			},
			threshold:      5,
			wantStatusCode: http.StatusOK,
			wantLen:        0, // 8 > 5, so no alerts
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetInventoryData()
			t.Cleanup(resetInventoryData)

			inventoryThreshold = tt.threshold
			tt.setupProducts()

			req := httptest.NewRequest(http.MethodGet, "/api/admin/inventory/alerts", nil)
			w := httptest.NewRecorder()

			GetInventoryAlerts(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var alerts []models.InventoryAlert
			err := json.NewDecoder(w.Body).Decode(&alerts)
			assert.NoError(t, err)
			assert.Len(t, alerts, tt.wantLen)

			if tt.wantLen > 0 && tt.wantProductID > 0 {
				found := false
				for _, alert := range alerts {
					if alert.ProductID == tt.wantProductID {
						found = true
						assert.Equal(t, tt.wantProductName, alert.ProductName)
						assert.Equal(t, tt.wantThreshold, alert.Threshold)
						assert.Equal(t, tt.wantCurrentStock, alert.CurrentStock)
						break
					}
				}
				assert.True(t, found, "expected product ID %d not found in alerts", tt.wantProductID)
			}
		})
	}
}

func TestGetInventoryAlertsThresholdChange(t *testing.T) {
	resetInventoryData()
	t.Cleanup(resetInventoryData)

	// 初始阈值 10，product 2 库存 8 应该触发预警
	req := httptest.NewRequest(http.MethodGet, "/api/admin/inventory/alerts", nil)
	w := httptest.NewRecorder()
	GetInventoryAlerts(w, req)

	var alerts []models.InventoryAlert
	err := json.NewDecoder(w.Body).Decode(&alerts)
	assert.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.Equal(t, int64(2), alerts[0].ProductID)
	assert.Equal(t, 10, alerts[0].Threshold)
	assert.Equal(t, 8, alerts[0].CurrentStock)

	// 修改阈值为 5
	inventoryThreshold = 5

	// 再次查询，应该没有预警
	req = httptest.NewRequest(http.MethodGet, "/api/admin/inventory/alerts", nil)
	w = httptest.NewRecorder()
	GetInventoryAlerts(w, req)

	err = json.NewDecoder(w.Body).Decode(&alerts)
	assert.NoError(t, err)
	assert.Len(t, alerts, 0)
}

// ==================== AdjustInventory Tests ====================

func TestAdjustInventoryExtended(t *testing.T) {
	tests := []struct {
		name               string
		requestBody        string
		wantStatusCode     int
		wantErr            bool
		wantStock          int
		wantChangeType     string
		wantLogQuantity    int
		wantLogBeforeStock int
		wantLogAfterStock  int
	}{
		{
			name:               "increase stock",
			requestBody:        `{"productId":1,"quantity":10,"reason":"补货"}`,
			wantStatusCode:     http.StatusOK,
			wantErr:            false,
			wantStock:          60,
			wantChangeType:     "in",
			wantLogQuantity:    10,
			wantLogBeforeStock: 50,
			wantLogAfterStock:  60,
		},
		{
			name:               "decrease stock",
			requestBody:        `{"productId":1,"quantity":-20,"reason":"销售出库"}`,
			wantStatusCode:     http.StatusOK,
			wantErr:            false,
			wantStock:          30,
			wantChangeType:     "out",
			wantLogQuantity:    20,
			wantLogBeforeStock: 50,
			wantLogAfterStock:  30,
		},
		{
			name:               "adjust to same stock",
			requestBody:        `{"productId":1,"quantity":0,"reason":"零调整"}`,
			wantStatusCode:     http.StatusOK,
			wantErr:            false,
			wantStock:          50,
			wantChangeType:     "adjust",
			wantLogQuantity:    0,
			wantLogBeforeStock: 50,
			wantLogAfterStock:  50,
		},
		{
			name:           "product not found",
			requestBody:    `{"productId":999,"quantity":10,"reason":"test"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing productId",
			requestBody:    `{"quantity":10,"reason":"test"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "zero productId",
			requestBody:    `{"productId":0,"quantity":10,"reason":"test"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "negative productId",
			requestBody:    `{"productId":-1,"quantity":10,"reason":"test"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "empty body",
			requestBody:    ``,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid JSON structure",
			requestBody:    `{"productId":"not_a_number","quantity":10}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:               "without reason field",
			requestBody:        `{"productId":1,"quantity":5}`,
			wantStatusCode:     http.StatusOK,
			wantErr:            false,
			wantStock:          55,
			wantChangeType:     "in",
			wantLogQuantity:    5,
			wantLogBeforeStock: 50,
			wantLogAfterStock:  55,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetInventoryData()
			t.Cleanup(resetInventoryData)

			req := httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			AdjustInventory(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantErr {
				// 错误情况下不应该修改库存
				if !strings.Contains(tt.name, "not found") {
					assert.Equal(t, 50, products[1].Stock)
				}
			} else {
				var product models.Product
				err := json.NewDecoder(w.Body).Decode(&product)
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStock, product.Stock)

				// 验证库存日志
				assert.Len(t, inventoryLogs, 1)
				if len(inventoryLogs) > 0 {
					assert.Equal(t, int64(1), inventoryLogs[0].ProductID)
					assert.Equal(t, tt.wantChangeType, inventoryLogs[0].ChangeType)
					assert.Equal(t, tt.wantLogQuantity, inventoryLogs[0].Quantity)
					assert.Equal(t, tt.wantLogBeforeStock, inventoryLogs[0].BeforeStock)
					assert.Equal(t, tt.wantLogAfterStock, inventoryLogs[0].AfterStock)
					assert.Equal(t, "admin", inventoryLogs[0].Operator)
				}
			}
		})
	}
}

func TestAdjustInventoryNegativeStockBoundary(t *testing.T) {
	resetInventoryData()
	t.Cleanup(resetInventoryData)

	tests := []struct {
		name           string
		productID      int64
		quantity       int
		wantStatusCode int
		wantStock      int
		wantChangeType string
	}{
		{
			name:           "decrease to zero",
			productID:      2, // stock 8
			quantity:       -8,
			wantStatusCode: http.StatusOK,
			wantStock:      0,
			wantChangeType: "out",
		},
		{
			name:           "decrease below zero should clamp to zero",
			productID:      1, // stock 50
			quantity:       -100,
			wantStatusCode: http.StatusOK,
			wantStock:      0,
			wantChangeType: "out",
		},
		{
			name:           "decrease partially",
			productID:      1, // stock 50
			quantity:       -25,
			wantStatusCode: http.StatusOK,
			wantStock:      25,
			wantChangeType: "out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetInventoryData()
			t.Cleanup(resetInventoryData)

			body := `{"productId":` + string(rune('0'+int(tt.productID))) + `,"quantity":` + formatInt(tt.quantity) + `,"reason":"test"}`
			if tt.productID == 1 && tt.quantity == -100 {
				body = `{"productId":1,"quantity":-100,"reason":"test"}`
			}
			if tt.productID == 1 && tt.quantity == -25 {
				body = `{"productId":1,"quantity":-25,"reason":"test"}`
			}
			if tt.productID == 2 {
				body = `{"productId":2,"quantity":` + formatInt(tt.quantity) + `,"reason":"test"}`
			}

			req := httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			AdjustInventory(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var product models.Product
			err := json.NewDecoder(w.Body).Decode(&product)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStock, product.Stock)
			assert.GreaterOrEqual(t, product.Stock, 0) // 确保库存不为负

			// 验证日志
			assert.Len(t, inventoryLogs, 1)
			if len(inventoryLogs) > 0 {
				assert.Equal(t, tt.wantChangeType, inventoryLogs[0].ChangeType)
			}
		})
	}
}

func TestAdjustInventoryMultipleOperations(t *testing.T) {
	resetInventoryData()
	t.Cleanup(resetInventoryData)

	// 第一次调整：+10
	req := httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust",
		strings.NewReader(`{"productId":1,"quantity":10,"reason":"第一次补货"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	AdjustInventory(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 60, products[1].Stock)

	// 第二次调整：-5
	req = httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust",
		strings.NewReader(`{"productId":1,"quantity":-5,"reason":"销售"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	AdjustInventory(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 55, products[1].Stock)

	// 第三次调整：+20
	req = httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust",
		strings.NewReader(`{"productId":1,"quantity":20,"reason":"第二次补货"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	AdjustInventory(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 75, products[1].Stock)

	// 验证日志
	assert.Len(t, inventoryLogs, 3)
	if len(inventoryLogs) >= 3 {
		assert.Equal(t, "in", inventoryLogs[0].ChangeType)
		assert.Equal(t, 10, inventoryLogs[0].Quantity)
		assert.Equal(t, "第一次补货", inventoryLogs[0].Reason)

		assert.Equal(t, "out", inventoryLogs[1].ChangeType)
		assert.Equal(t, 5, inventoryLogs[1].Quantity)
		assert.Equal(t, "销售", inventoryLogs[1].Reason)

		assert.Equal(t, "in", inventoryLogs[2].ChangeType)
		assert.Equal(t, 20, inventoryLogs[2].Quantity)
		assert.Equal(t, "第二次补货", inventoryLogs[2].Reason)
	}
}

func TestAdjustInventoryDeletedProduct(t *testing.T) {
	resetInventoryData()
	t.Cleanup(resetInventoryData)

	// 尝试调整已删除商品的库存
	req := httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust",
		strings.NewReader(`{"productId":3,"quantity":10,"reason":"调整已删除商品"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	AdjustInventory(w, req)

	// 已删除的商品仍然可以被调整库存（根据当前实现）
	assert.Equal(t, http.StatusOK, w.Code)

	var product models.Product
	err := json.NewDecoder(w.Body).Decode(&product)
	assert.NoError(t, err)
	assert.Equal(t, 15, product.Stock) // 5 + 10 = 15
}

// formatInt 将整数转换为字符串
func formatInt(n int) string {
	if n < 0 {
		return "-" + formatInt(-n)
	}
	if n == 0 {
		return "0"
	}
	var result string
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
