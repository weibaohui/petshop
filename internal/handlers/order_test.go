package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetOrderData resets orders, products and inventory data to initial state for order tests.
func resetOrderData() {
	dataMu.Lock()
	defer dataMu.Unlock()

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
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
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
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	nextProductID = 3

	orders = make(map[int64]*models.Order)
	orders[1] = &models.Order{
		ID:     1,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "paid",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	orders[2] = &models.Order{
		ID:     2,
		UserID: 2,
		Products: []models.OrderItem{
			{ProductID: 2, ProductName: "猫粮 5kg", Price: 199.00, Quantity: 2, Subtotal: 398.00},
		},
		TotalAmount: 398.00,
		Status:      "shipped",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	orders[3] = &models.Order{
		ID:     3,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 2, Subtotal: 598.00},
			{ProductID: 2, ProductName: "猫粮 5kg", Price: 199.00, Quantity: 1, Subtotal: 199.00},
		},
		TotalAmount: 797.00,
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	nextOrderID = 4

	inventoryLogs = make([]models.Inventory, 0)
	nextInventoryID = 1
}

func TestOrder_ListOrders(t *testing.T) {
	resetOrderData()

	tests := []struct {
		name           string
		query          string
		wantStatusCode int
		wantLen        int
		wantContainID  int64
	}{
		{
			name:           "list all orders",
			query:          "",
			wantStatusCode: http.StatusOK,
			wantLen:        3,
		},
		{
			name:           "filter by status paid",
			query:          "?status=paid",
			wantStatusCode: http.StatusOK,
			wantLen:        1,
			wantContainID:  1,
		},
		{
			name:           "filter by status shipped",
			query:          "?status=shipped",
			wantStatusCode: http.StatusOK,
			wantLen:        1,
			wantContainID:  2,
		},
		{
			name:           "filter by status pending",
			query:          "?status=pending",
			wantStatusCode: http.StatusOK,
			wantLen:        1,
			wantContainID:  3,
		},
		{
			name:           "filter by non-matching status",
			query:          "?status=delivered",
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/orders"+tt.query, nil)
			w := httptest.NewRecorder()

			ListOrders(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var response []*models.Order
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)
			assert.Len(t, response, tt.wantLen)

			if tt.wantContainID > 0 {
				found := false
				for _, o := range response {
					if o.ID == tt.wantContainID {
						found = true
						break
					}
				}
				assert.True(t, found, "expected order %d in response", tt.wantContainID)
			}
		})
	}
}

func TestOrder_ListOrders_Empty(t *testing.T) {
	resetOrderData()
	t.Cleanup(resetOrderData)

	dataMu.Lock()
	orders = make(map[int64]*models.Order)
	dataMu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/orders", nil)
	w := httptest.NewRecorder()

	ListOrders(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*models.Order
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Empty(t, response)
}

func TestOrder_GetOrder(t *testing.T) {
	resetOrderData()

	tests := []struct {
		name           string
		query          string
		wantStatusCode int
		wantOrderID    int64
		wantErrMsg     string
	}{
		{
			name:           "get existing order",
			query:          "?id=1",
			wantStatusCode: http.StatusOK,
			wantOrderID:    1,
		},
		{
			name:           "get another existing order",
			query:          "?id=2",
			wantStatusCode: http.StatusOK,
			wantOrderID:    2,
		},
		{
			name:           "missing id parameter",
			query:          "",
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg:     "id is required",
		},
		{
			name:           "invalid id format",
			query:          "?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg:     "invalid id",
		},
		{
			name:           "order not found",
			query:          "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErrMsg:     "order not found",
		},
		{
			name:           "id is zero",
			query:          "?id=0",
			wantStatusCode: http.StatusNotFound,
			wantErrMsg:     "order not found",
		},
		{
			name:           "id is negative",
			query:          "?id=-1",
			wantStatusCode: http.StatusNotFound,
			wantErrMsg:     "order not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/order"+tt.query, nil)
			w := httptest.NewRecorder()

			GetOrder(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode == http.StatusOK {
				var response models.Order
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.wantOrderID, response.ID)
			} else {
				assert.Contains(t, w.Body.String(), tt.wantErrMsg)
			}
		})
	}
}

func TestOrder_UpdateOrderStatus(t *testing.T) {
	tests := []struct {
		name           string
		setup          func()
		requestBody    string
		wantStatusCode int
		wantStatus     string
		wantErrMsg     string
	}{
		{
			name:           "update to pending",
			setup:          resetOrderData,
			requestBody:    `{"id":1,"status":"pending"}`,
			wantStatusCode: http.StatusOK,
			wantStatus:     "pending",
		},
		{
			name:           "update to paid",
			setup:          resetOrderData,
			requestBody:    `{"id":1,"status":"paid"}`,
			wantStatusCode: http.StatusOK,
			wantStatus:     "paid",
		},
		{
			name:           "update to shipped",
			setup:          resetOrderData,
			requestBody:    `{"id":1,"status":"shipped"}`,
			wantStatusCode: http.StatusOK,
			wantStatus:     "shipped",
		},
		{
			name:           "update to delivered",
			setup:          resetOrderData,
			requestBody:    `{"id":1,"status":"delivered"}`,
			wantStatusCode: http.StatusOK,
			wantStatus:     "delivered",
		},
		{
			name:           "update to cancelled",
			setup:          resetOrderData,
			requestBody:    `{"id":1,"status":"cancelled"}`,
			wantStatusCode: http.StatusOK,
			wantStatus:     "cancelled",
		},
		{
			name:           "update to refunding",
			setup:          resetOrderData,
			requestBody:    `{"id":1,"status":"refunding"}`,
			wantStatusCode: http.StatusOK,
			wantStatus:     "refunding",
		},
		{
			name:           "update to refunded",
			setup:          resetOrderData,
			requestBody:    `{"id":1,"status":"refunded"}`,
			wantStatusCode: http.StatusOK,
			wantStatus:     "refunded",
		},
		{
			name:           "missing request body",
			setup:          resetOrderData,
			requestBody:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg:     "invalid JSON",
		},
		{
			name:           "invalid JSON",
			setup:          resetOrderData,
			requestBody:    `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg:     "invalid JSON",
		},
		{
			name:           "order not found",
			setup:          resetOrderData,
			requestBody:    `{"id":999,"status":"paid"}`,
			wantStatusCode: http.StatusNotFound,
			wantErrMsg:     "order not found",
		},
		{
			name:           "invalid status value",
			setup:          resetOrderData,
			requestBody:    `{"id":1,"status":"unknown"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg:     "invalid status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			req := httptest.NewRequest(http.MethodPut, "/api/admin/order", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateOrderStatus(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode == http.StatusOK {
				var response models.Order
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, response.Status)
			} else {
				assert.Contains(t, w.Body.String(), tt.wantErrMsg)
			}
		})
	}
}

func TestOrder_ProcessRefund(t *testing.T) {
	tests := []struct {
		name                 string
		setup                func()
		requestBody          string
		wantStatusCode       int
		wantErrMsg           string
		wantStockProduct1    int
		wantStockProduct2    int
		wantLogCount         int
		wantLogReason        string
		expectedRefundReason string
	}{
		{
			name: "process refund successfully",
			setup: func() {
				resetOrderData()
			},
			requestBody:          `{"orderId":1,"reason":"质量问题"}`,
			wantStatusCode:       http.StatusOK,
			wantStockProduct1:    51, // 50 + 1
			wantLogCount:         1,
			wantLogReason:        "退款返还: 订单1",
			expectedRefundReason: "质量问题",
		},
		{
			name: "process refund for order with multiple products",
			setup: func() {
				resetOrderData()
			},
			requestBody:          `{"orderId":3,"reason":"缺货"}`,
			wantStatusCode:       http.StatusOK,
			wantStockProduct1:    52, // 50 + 2
			wantStockProduct2:    9,  // 8 + 1
			wantLogCount:         2,
			expectedRefundReason: "缺货",
		},
		{
			name: "missing request body",
			setup: func() {
				resetOrderData()
			},
			requestBody:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg:     "invalid JSON",
		},
		{
			name: "invalid JSON",
			setup: func() {
				resetOrderData()
			},
			requestBody:    `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg:     "invalid JSON",
		},
		{
			name: "order not found",
			setup: func() {
				resetOrderData()
			},
			requestBody:    `{"orderId":999,"reason":"不想要了"}`,
			wantStatusCode: http.StatusNotFound,
			wantErrMsg:     "order not found",
		},
		{
			name: "refund order with non-existent product skips missing product",
			setup: func() {
				resetOrderData()
				dataMu.Lock()
				orders[4] = &models.Order{
					ID:     4,
					UserID: 1,
					Products: []models.OrderItem{
						{ProductID: 999, ProductName: "不存在的商品", Price: 99.00, Quantity: 3, Subtotal: 297.00},
						{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
					},
					TotalAmount: 596.00,
					Status:      "paid",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				nextOrderID = 5
				dataMu.Unlock()
			},
			requestBody:          `{"orderId":4,"reason":"测试混合商品退款"}`,
			wantStatusCode:       http.StatusOK,
			wantStockProduct1:    51, // 50 + 1
			wantLogCount:         1,
			expectedRefundReason: "测试混合商品退款",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			req := httptest.NewRequest(http.MethodPost, "/api/admin/order/refund", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ProcessRefund(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode == http.StatusOK {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, "refund processed", response["message"])

				// verify order status updated
				var reqBody struct {
					OrderID int64 `json:"orderId"`
				}
				err = json.Unmarshal([]byte(tt.requestBody), &reqBody)
				require.NoError(t, err)

				dataMu.RLock()
				o, ok := orders[reqBody.OrderID]
				dataMu.RUnlock()
				require.True(t, ok)
				assert.Equal(t, "refunded", o.Status)
				assert.Equal(t, tt.expectedRefundReason, o.RefundReason)
				// stock verification
				dataMu.RLock()
				if tt.wantStockProduct1 > 0 {
					assert.Equal(t, tt.wantStockProduct1, products[1].Stock)
				}
				if tt.wantStockProduct2 > 0 {
					assert.Equal(t, tt.wantStockProduct2, products[2].Stock)
				}
				assert.Len(t, inventoryLogs, tt.wantLogCount)
				if tt.wantLogCount > 0 && tt.wantLogReason != "" {
					found := false
					for _, log := range inventoryLogs {
						if strings.Contains(log.Reason, tt.wantLogReason) {
							found = true
							break
						}
					}
					assert.True(t, found, "expected inventory log with reason containing %q", tt.wantLogReason)
				}
				dataMu.RUnlock()
			} else {
				assert.Contains(t, w.Body.String(), tt.wantErrMsg)
			}
		})
	}
}

func TestOrder_ProcessRefund_InventoryLogs(t *testing.T) {
	resetOrderData()

	reqBody := `{"orderId":2,"reason":"物流损坏"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/order/refund", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ProcessRefund(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	dataMu.RLock()
	defer dataMu.RUnlock()

	// Order 2 has product 2 quantity 2, stock should be 8+2=10
	assert.Equal(t, 10, products[2].Stock)
	require.Len(t, inventoryLogs, 1)
	assert.Equal(t, int64(2), inventoryLogs[0].ProductID)
	assert.Equal(t, "in", inventoryLogs[0].ChangeType)
	assert.Equal(t, 2, inventoryLogs[0].Quantity)
	assert.Equal(t, 8, inventoryLogs[0].BeforeStock)
	assert.Equal(t, 10, inventoryLogs[0].AfterStock)
	assert.Equal(t, "system", inventoryLogs[0].Operator)
	assert.Contains(t, inventoryLogs[0].Reason, "退款返还")
}
