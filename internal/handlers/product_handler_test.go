package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetProductData resets the global in-memory data to a known initial state for product tests.
func resetProductData() {
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
		Name:        "已删除商品",
		Description: "测试删除",
		Category:    "测试",
		Price:       99.00,
		Stock:       10,
		Status:      "deleted",
		Images:      []string{},
	}
	nextProductID = 4

	orders = make(map[int64]*models.Order)
	orders[1] = &models.Order{
		ID:     1,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "paid",
	}
	orders[2] = &models.Order{
		ID:     2,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 2, ProductName: "猫粮 5kg", Price: 199.00, Quantity: 1, Subtotal: 199.00},
		},
		TotalAmount: 199.00,
		Status:      "delivered",
	}
	nextOrderID = 3

	inventoryLogs = make([]models.Inventory, 0)
	nextInventoryID = 1
	inventoryThreshold = 10
}

func TestProductHandler_ListProducts(t *testing.T) {
	resetProductData()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/products", nil)
	w := httptest.NewRecorder()

	ListProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []*models.Product
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// Should return only non-deleted products
	assert.Len(t, result, 2)
	for _, p := range result {
		assert.NotEqual(t, "deleted", p.Status)
	}
}

func TestProductHandler_GetProduct(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		wantStatusCode int
		wantFound      bool
		wantName       string
	}{
		{
			name:           "normal get",
			url:            "/api/admin/product?id=1",
			wantStatusCode: http.StatusOK,
			wantFound:      true,
			wantName:       "狗粮 10kg",
		},
		{
			name:           "missing id",
			url:            "/api/admin/product",
			wantStatusCode: http.StatusBadRequest,
			wantFound:      false,
		},
		{
			name:           "invalid id format",
			url:            "/api/admin/product?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantFound:      false,
		},
		{
			name:           "product not found",
			url:            "/api/admin/product?id=999",
			wantStatusCode: http.StatusNotFound,
			wantFound:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProductData()

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			GetProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantFound {
				var result models.Product
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				assert.Equal(t, tt.wantName, result.Name)
			}
		})
	}
}

func TestProductHandler_CreateProduct(t *testing.T) {
	tests := []struct {
		name              string
		body              string
		wantStatusCode    int
		wantCreated       bool
		wantName          string
		wantStock         int
		wantInventoryLen  int
		wantInventoryType string
	}{
		{
			name:              "normal create",
			body:              `{"name":"测试商品","description":"测试描述","category":"测试","price":99.99,"stock":20,"images":["img1.jpg"]}`,
			wantStatusCode:    http.StatusCreated,
			wantCreated:       true,
			wantName:          "测试商品",
			wantStock:         20,
			wantInventoryLen:  1,
			wantInventoryType: "in",
		},
		{
			name:           "missing name",
			body:           `{"description":"测试描述","price":99.99,"stock":10}`,
			wantStatusCode: http.StatusBadRequest,
			wantCreated:    false,
		},
		{
			name:           "invalid price zero",
			body:           `{"name":"测试商品","price":0,"stock":10}`,
			wantStatusCode: http.StatusBadRequest,
			wantCreated:    false,
		},
		{
			name:           "invalid price negative",
			body:           `{"name":"测试商品","price":-10,"stock":10}`,
			wantStatusCode: http.StatusBadRequest,
			wantCreated:    false,
		},
		{
			name:           "invalid JSON",
			body:           `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantCreated:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProductData()

			req := httptest.NewRequest(http.MethodPost, "/api/admin/products", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			CreateProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantCreated {
				var result models.Product
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				assert.Equal(t, tt.wantName, result.Name)
				assert.Equal(t, tt.wantStock, result.Stock)
				assert.Equal(t, "on_sale", result.Status)

				// Verify inventory log
				dataMu.RLock()
				assert.Len(t, inventoryLogs, tt.wantInventoryLen)
				if len(inventoryLogs) > 0 {
					lastLog := inventoryLogs[len(inventoryLogs)-1]
					assert.Equal(t, tt.wantInventoryType, lastLog.ChangeType)
					assert.Equal(t, 0, lastLog.BeforeStock)
					assert.Equal(t, tt.wantStock, lastLog.AfterStock)
					assert.Equal(t, tt.wantStock, lastLog.Quantity)
				}
				dataMu.RUnlock()
			}
		})
	}
}

func TestProductHandler_UpdateProduct(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		wantStatusCode   int
		wantUpdated      bool
		wantName         string
		wantPrice        float64
		wantStock        int
		wantStatus       string
		wantInventoryLen int
		checkInventory   bool
		wantInvType      string
		wantBeforeStock  int
		wantAfterStock   int
	}{
		{
			name:             "normal update",
			body:             `{"id":1,"name":"更新狗粮","price":349.00,"stock":60,"status":"on_sale"}`,
			wantStatusCode:   http.StatusOK,
			wantUpdated:      true,
			wantName:         "更新狗粮",
			wantPrice:        349.00,
			wantStock:        60,
			wantStatus:       "on_sale",
			wantInventoryLen: 1,
			checkInventory:   true,
			wantInvType:      "in",
			wantBeforeStock:  50,
			wantAfterStock:   60,
		},
		{
			name:           "missing id",
			body:           `{"name":"更新狗粮","price":349.00}`,
			wantStatusCode: http.StatusBadRequest,
			wantUpdated:    false,
		},
		{
			name:           "product not found",
			body:           `{"id":999,"name":"更新狗粮","price":349.00}`,
			wantStatusCode: http.StatusNotFound,
			wantUpdated:    false,
		},
		{
			name:             "partial update - only name with explicit stock",
			body:             `{"id":1,"name":"仅改名","stock":50}`,
			wantStatusCode:   http.StatusOK,
			wantUpdated:      true,
			wantName:         "仅改名",
			wantPrice:        299.00, // unchanged
			wantStock:        50,     // unchanged because explicitly provided
			wantStatus:       "on_sale",
			wantInventoryLen: 0,
			checkInventory:   false,
		},
		{
			name:             "partial update - only description with explicit stock",
			body:             `{"id":1,"description":"新描述","stock":50}`,
			wantStatusCode:   http.StatusOK,
			wantUpdated:      true,
			wantName:         "狗粮 10kg", // unchanged
			wantPrice:        299.00,    // unchanged
			wantStock:        50,        // unchanged because explicitly provided
			wantStatus:       "on_sale",
			wantInventoryLen: 0,
			checkInventory:   false,
		},
		{
			name:             "partial update without stock sets stock to 0",
			body:             `{"id":1,"name":"仅改名"}`,
			wantStatusCode:   http.StatusOK,
			wantUpdated:      true,
			wantName:         "仅改名",
			wantPrice:        299.00, // unchanged
			wantStock:        0,      // stock defaults to 0 when omitted
			wantStatus:       "on_sale",
			wantInventoryLen: 1,
			checkInventory:   true,
			wantInvType:      "out",
			wantBeforeStock:  50,
			wantAfterStock:   0,
		},
		{
			name:             "stock decrease",
			body:             `{"id":1,"stock":30}`,
			wantStatusCode:   http.StatusOK,
			wantUpdated:      true,
			wantName:         "狗粮 10kg", // unchanged
			wantPrice:        299.00,    // unchanged
			wantStock:        30,
			wantStatus:       "on_sale",
			wantInventoryLen: 1,
			checkInventory:   true,
			wantInvType:      "out",
			wantBeforeStock:  50,
			wantAfterStock:   30,
		},
		{
			name:             "no stock change with explicit current stock",
			body:             `{"id":1,"status":"off_sale","stock":50}`,
			wantStatusCode:   http.StatusOK,
			wantUpdated:      true,
			wantName:         "狗粮 10kg",
			wantPrice:        299.00,
			wantStock:        50,
			wantStatus:       "off_sale",
			wantInventoryLen: 0,
			checkInventory:   false,
		},
		{
			name:           "invalid JSON",
			body:           `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantUpdated:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProductData()

			req := httptest.NewRequest(http.MethodPut, "/api/admin/product", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantUpdated {
				var result models.Product
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				assert.Equal(t, tt.wantName, result.Name)
				assert.Equal(t, tt.wantPrice, result.Price)
				assert.Equal(t, tt.wantStock, result.Stock)
				assert.Equal(t, tt.wantStatus, result.Status)

				dataMu.RLock()
				assert.Len(t, inventoryLogs, tt.wantInventoryLen)
				if tt.checkInventory && len(inventoryLogs) > 0 {
					lastLog := inventoryLogs[len(inventoryLogs)-1]
					assert.Equal(t, int64(1), lastLog.ProductID)
					assert.Equal(t, tt.wantInvType, lastLog.ChangeType)
					assert.Equal(t, tt.wantBeforeStock, lastLog.BeforeStock)
					assert.Equal(t, tt.wantAfterStock, lastLog.AfterStock)
				}
				dataMu.RUnlock()
			}
		})
	}
}

func TestProductHandler_DeleteProduct(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		setupOrders    func()
		wantStatusCode int
		wantDeleted    bool
	}{
		{
			name:           "normal delete",
			url:            "/api/admin/product?id=3",
			setupOrders:    func() {},
			wantStatusCode: http.StatusOK,
			wantDeleted:    true,
		},
		{
			name:           "missing id",
			url:            "/api/admin/product",
			setupOrders:    func() {},
			wantStatusCode: http.StatusBadRequest,
			wantDeleted:    false,
		},
		{
			name:           "invalid id format",
			url:            "/api/admin/product?id=abc",
			setupOrders:    func() {},
			wantStatusCode: http.StatusBadRequest,
			wantDeleted:    false,
		},
		{
			name:           "product not found",
			url:            "/api/admin/product?id=999",
			setupOrders:    func() {},
			wantStatusCode: http.StatusNotFound,
			wantDeleted:    false,
		},
		{
			name:        "product has pending orders",
			url:         "/api/admin/product?id=1",
			setupOrders: func() {},
			// Product 1 is in order 1 which has status "paid" -> pending
			wantStatusCode: http.StatusConflict,
			wantDeleted:    false,
		},
		{
			name: "product in delivered order only",
			url:  "/api/admin/product?id=2",
			setupOrders: func() {
				dataMu.Lock()
				// Ensure order 2 has product 2 with delivered status (already in resetProductData)
				dataMu.Unlock()
			},
			wantStatusCode: http.StatusOK,
			wantDeleted:    true,
		},
		{
			name: "product has pending orders in new order",
			url:  "/api/admin/product?id=2",
			setupOrders: func() {
				dataMu.Lock()
				orders[10] = &models.Order{
					ID:     10,
					UserID: 1,
					Products: []models.OrderItem{
						{ProductID: 2, ProductName: "猫粮 5kg", Price: 199.00, Quantity: 1, Subtotal: 199.00},
					},
					TotalAmount: 199.00,
					Status:      "pending",
				}
				dataMu.Unlock()
			},
			wantStatusCode: http.StatusConflict,
			wantDeleted:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProductData()
			tt.setupOrders()

			req := httptest.NewRequest(http.MethodDelete, tt.url, nil)
			w := httptest.NewRecorder()

			DeleteProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantDeleted {
				var result map[string]string
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				assert.Equal(t, "product deleted", result["message"])
			}
		})
	}
}
