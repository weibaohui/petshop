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

// resetAdminData 重置所有 admin 数据到初始状态
func resetAdminData() {
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
	nextProductID = 3

	// 重置用户
	users = make(map[int64]*models.User)
	users[1] = &models.User{
		ID:       1,
		Username: "user1",
		Email:    "user1@example.com",
		Phone:    "13800138000",
		Status:   "active",
		Role:     "user",
	}
	users[2] = &models.User{
		ID:       2,
		Username: "user2",
		Email:    "user2@example.com",
		Phone:    "13800138001",
		Status:   "active",
		Role:     "user",
	}
	nextUserID = 3

	// 重置订单
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
	nextOrderID = 2

	// 重置库存记录
	inventoryLogs = make([]models.Inventory, 0)
	nextInventoryID = 1

	// 重置轮播图
	carousels = make(map[int64]*models.Carousel)
	carousels[1] = &models.Carousel{
		ID:        1,
		ImageURL:  "/static/carousel/banner1.jpg",
		LinkURL:   "/product/1",
		SortOrder: 1,
		Title:     "春季大促",
		Status:    "active",
	}
	nextCarouselID = 2

	// 重置公告
	announcements = make(map[int64]*models.Announcement)
	announcements[1] = &models.Announcement{
		ID:      1,
		Title:   "春节放假通知",
		Content: "春节期间客服工作时间调整",
		Status:  "active",
	}
	nextAnnouncementID = 2

	// 重置系统配置
	systemConfigs = make(map[string]string)
	systemConfigs["site_name"] = "宠物商店"
	systemConfigs["inventory_threshold"] = "10"
	inventoryThreshold = 10
}

// ==================== Product Handler Tests ====================

func TestListProducts(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/products", nil)
	w := httptest.NewRecorder()

	ListProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*models.Product
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 2)
}

func TestGetProduct(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "get existing product",
			queryString:    "?id=1",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "get non-existent product",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id parameter",
			queryString:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id format",
			queryString:    "?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminData()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/product"+tt.queryString, nil)
			w := httptest.NewRecorder()

			GetProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var product models.Product
				err := json.NewDecoder(w.Body).Decode(&product)
				assert.NoError(t, err)
				assert.Equal(t, "狗粮 10kg", product.Name)
			}
		})
	}
}

func TestCreateProduct(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "create product with valid data",
			requestBody:    `{"name":"鸟粮 1kg","description":"营养鸟粮","category":"鸟粮","price":59.99,"stock":100,"images":["/static/bird.jpg"]}`,
			wantStatusCode: http.StatusCreated,
			wantErr:        false,
		},
		{
			name:           "missing name",
			requestBody:    `{"description":"no name","price":10.0}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid price",
			requestBody:    `{"name":"test","price":-10}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "zero price",
			requestBody:    `{"name":"test","price":0}`,
			wantStatusCode: http.StatusBadRequest,
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
			resetAdminData()
			req := httptest.NewRequest(http.MethodPost, "/api/admin/products", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			CreateProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var product models.Product
				err := json.NewDecoder(w.Body).Decode(&product)
				assert.NoError(t, err)
				assert.NotZero(t, product.ID)
				assert.Equal(t, "鸟粮 1kg", product.Name)
			}
		})
	}
}

func TestUpdateProduct(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "update existing product",
			requestBody:    `{"id":1,"name":"狗粮 20kg","price":399.00,"stock":60,"status":"on_sale"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "update non-existent product",
			requestBody:    `{"id":999,"name":"not exist"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id",
			requestBody:    `{"name":"no id"}`,
			wantStatusCode: http.StatusBadRequest,
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
			resetAdminData()
			req := httptest.NewRequest(http.MethodPut, "/api/admin/product", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var product models.Product
				err := json.NewDecoder(w.Body).Decode(&product)
				assert.NoError(t, err)
				assert.Equal(t, "狗粮 20kg", product.Name)

				// 验证库存变更记录
				assert.Len(t, inventoryLogs, 1)
				if len(inventoryLogs) > 0 {
					assert.Equal(t, int64(1), inventoryLogs[0].ProductID)
					assert.Equal(t, 10, inventoryLogs[0].Quantity) // 库存从50增加到60
					assert.Equal(t, "in", inventoryLogs[0].ChangeType)
					assert.Equal(t, 50, inventoryLogs[0].BeforeStock)
					assert.Equal(t, 60, inventoryLogs[0].AfterStock)
				}
			}
		})
	}
}

func TestDeleteProduct(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "delete existing product",
			queryString:    "?id=2",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "delete non-existent product",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id",
			queryString:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id format",
			queryString:    "?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminData()
			req := httptest.NewRequest(http.MethodDelete, "/api/admin/product"+tt.queryString, nil)
			w := httptest.NewRecorder()

			DeleteProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestDeleteProductWithPendingOrder(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	// 尝试删除有关联未完成订单的商品 (product ID 1 在订单 1 中)
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/product?id=1", nil)
	w := httptest.NewRecorder()

	DeleteProduct(w, req)

	// 应该返回冲突状态码，因为该商品有待处理订单
	assert.Equal(t, http.StatusConflict, w.Code)
}

// ==================== User Handler Tests ====================

func TestListUsers(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	w := httptest.NewRecorder()

	ListUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*models.User
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 2)
}

func TestGetUser(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "get existing user",
			queryString:    "?id=1",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "get non-existent user",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id",
			queryString:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id",
			queryString:    "?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminData()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/user"+tt.queryString, nil)
			w := httptest.NewRecorder()

			GetUser(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var user models.User
				err := json.NewDecoder(w.Body).Decode(&user)
				assert.NoError(t, err)
				assert.Equal(t, "user1", user.Username)
			}
		})
	}
}

func TestUpdateUserStatus(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "disable user",
			requestBody:    `{"id":1,"status":"disabled"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "activate user",
			requestBody:    `{"id":1,"status":"active"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "invalid status",
			requestBody:    `{"id":1,"status":"invalid"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "non-existent user",
			requestBody:    `{"id":999,"status":"disabled"}`,
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
			resetAdminData()
			req := httptest.NewRequest(http.MethodPut, "/api/admin/user", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateUserStatus(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestResetUserPassword(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "reset password for existing user",
			requestBody:    `{"userId":1}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "reset password for non-existent user",
			requestBody:    `{"userId":999}`,
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
			resetAdminData()
			req := httptest.NewRequest(http.MethodPost, "/api/admin/user/reset-password", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ResetUserPassword(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response, "password")
			}
		})
	}
}

// ==================== Order Handler Tests ====================

func TestListOrders(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantLen        int
	}{
		{
			name:           "list all orders",
			queryString:    "",
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "filter by status",
			queryString:    "?status=paid",
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "filter by non-matching status",
			queryString:    "?status=cancelled",
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminData()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/orders"+tt.queryString, nil)
			w := httptest.NewRecorder()

			ListOrders(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var response []*models.Order
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Len(t, response, tt.wantLen)
		})
	}
}

func TestGetOrder(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "get existing order",
			queryString:    "?id=1",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "get non-existent order",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id",
			queryString:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id",
			queryString:    "?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminData()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/order"+tt.queryString, nil)
			w := httptest.NewRecorder()

			GetOrder(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var order models.Order
				err := json.NewDecoder(w.Body).Decode(&order)
				assert.NoError(t, err)
				assert.Equal(t, int64(1), order.ID)
			}
		})
	}
}

func TestUpdateOrderStatus(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "update to shipped",
			requestBody:    `{"id":1,"status":"shipped"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "update to delivered",
			requestBody:    `{"id":1,"status":"delivered"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "invalid status",
			requestBody:    `{"id":1,"status":"invalid"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "non-existent order",
			requestBody:    `{"id":999,"status":"shipped"}`,
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
			resetAdminData()
			req := httptest.NewRequest(http.MethodPut, "/api/admin/order", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateOrderStatus(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestProcessRefund(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "process refund for existing order",
			requestBody:    `{"orderId":1,"reason":"商品质量问题"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "process refund for non-existent order",
			requestBody:    `{"orderId":999,"reason":"test"}`,
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
			resetAdminData()

			// 对于退款测试，记录退款前的库存
			var stockBefore int
			if tt.name == "process refund for existing order" {
				stockBefore = products[1].Stock // 订单1包含product 1
			}

			req := httptest.NewRequest(http.MethodPost, "/api/admin/order/refund", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ProcessRefund(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, "refund processed", response["message"])

				// 验证库存增加（订单1包含1个product 1）
				if tt.name == "process refund for existing order" {
					assert.Equal(t, stockBefore+1, products[1].Stock)
				}
			}
		})
	}
}

// ==================== Inventory Handler Tests ====================

func TestListInventoryLogs(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	// 使用 AdjustInventory 创建稳定的库存日志记录
	req := httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust", strings.NewReader(`{"productId":1,"quantity":10,"reason":"测试库存调整"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	AdjustInventory(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 创建第二条库存日志记录
	req = httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust", strings.NewReader(`{"productId":2,"quantity":-2,"reason":"测试库存损耗"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	AdjustInventory(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 查询库存日志
	req = httptest.NewRequest(http.MethodGet, "/api/admin/inventory/logs", nil)
	w = httptest.NewRecorder()

	ListInventoryLogs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.Inventory
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	// 验证返回的库存日志数据
	assert.Len(t, response, 2)
	if len(response) >= 2 {
		// 验证第一条日志（product 1，增加10）
		assert.Equal(t, int64(1), response[0].ProductID)
		assert.Equal(t, 10, response[0].Quantity)
		assert.Equal(t, "测试库存调整", response[0].Reason)
		assert.Equal(t, "in", response[0].ChangeType)
		// 验证第二条日志（product 2，减少2，Quantity存储的是绝对值）
		assert.Equal(t, int64(2), response[1].ProductID)
		assert.Equal(t, 2, response[1].Quantity)
		assert.Equal(t, "测试库存损耗", response[1].Reason)
		assert.Equal(t, "out", response[1].ChangeType)
	}
}

func TestGetInventoryAlerts(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/inventory/alerts", nil)
	w := httptest.NewRecorder()

	GetInventoryAlerts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.InventoryAlert
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	// product 2 (猫粮) 库存为 8，低于阈值 10
	assert.Len(t, response, 1)
	if len(response) > 0 {
		assert.Equal(t, int64(2), response[0].ProductID)
		assert.Equal(t, "猫粮 5kg", response[0].ProductName)
	}
}

func TestAdjustInventory(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

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
			resetAdminData()
			req := httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			AdjustInventory(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestAdjustInventoryNegativeStock(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	// 测试库存不能变为负数
	req := httptest.NewRequest(http.MethodPost, "/api/admin/inventory/adjust",
		strings.NewReader(`{"productId":2,"quantity":-100,"reason":"测试负数"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	AdjustInventory(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var product models.Product
	err := json.NewDecoder(w.Body).Decode(&product)
	assert.NoError(t, err)
	// 库存不能小于 0
	assert.Equal(t, 0, product.Stock)
}

// ==================== Stats Handler Tests ====================

func TestGetSalesStats(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminData()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales"+tt.queryString, nil)
			w := httptest.NewRecorder()

			GetSalesStats(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var response []models.SalesStat
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.NotNil(t, response)
		})
	}
}

func TestGetHotProducts(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

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
			queryString:    "?limit=5",
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
			resetAdminData()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products"+tt.queryString, nil)
			w := httptest.NewRecorder()

			GetHotProducts(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var response []models.HotProduct
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.NotNil(t, response)
		})
	}
}

// ==================== System Handler Tests ====================

func TestListCarousels(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/carousels", nil)
	w := httptest.NewRecorder()

	ListCarousels(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*models.Carousel
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 1)
}

func TestCreateCarousel(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "create carousel with valid data",
			requestBody:    `{"imageUrl":"/static/banner2.jpg","linkUrl":"/product/2","sortOrder":2,"title":"夏季促销"}`,
			wantStatusCode: http.StatusCreated,
			wantErr:        false,
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
			resetAdminData()
			req := httptest.NewRequest(http.MethodPost, "/api/admin/carousels", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			CreateCarousel(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var carousel models.Carousel
				err := json.NewDecoder(w.Body).Decode(&carousel)
				assert.NoError(t, err)
				assert.NotZero(t, carousel.ID)
			}
		})
	}
}

func TestUpdateCarousel(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "update existing carousel",
			requestBody:    `{"id":1,"title":"更新后的标题","sortOrder":5}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "update non-existent carousel",
			requestBody:    `{"id":999,"title":"not exist"}`,
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
			resetAdminData()
			req := httptest.NewRequest(http.MethodPut, "/api/admin/carousel", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateCarousel(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestDeleteCarousel(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "delete existing carousel",
			queryString:    "?id=1",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "delete non-existent carousel",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id",
			queryString:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id",
			queryString:    "?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminData()
			req := httptest.NewRequest(http.MethodDelete, "/api/admin/carousel"+tt.queryString, nil)
			w := httptest.NewRecorder()

			DeleteCarousel(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestListAnnouncements(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/announcements", nil)
	w := httptest.NewRecorder()

	ListAnnouncements(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*models.Announcement
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 1)
}

func TestCreateAnnouncement(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "create announcement with valid data",
			requestBody:    `{"title":"新公告","content":"公告内容"}`,
			wantStatusCode: http.StatusCreated,
			wantErr:        false,
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
			resetAdminData()
			req := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			CreateAnnouncement(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var announcement models.Announcement
				err := json.NewDecoder(w.Body).Decode(&announcement)
				assert.NoError(t, err)
				assert.NotZero(t, announcement.ID)
			}
		})
	}
}

func TestUpdateAnnouncement(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "update existing announcement",
			requestBody:    `{"id":1,"title":"更新后的公告","content":"更新后的内容"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "update non-existent announcement",
			requestBody:    `{"id":999,"title":"not exist"}`,
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
			resetAdminData()
			req := httptest.NewRequest(http.MethodPut, "/api/admin/announcement", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateAnnouncement(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestDeleteAnnouncement(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "delete existing announcement",
			queryString:    "?id=1",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "delete non-existent announcement",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id",
			queryString:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id",
			queryString:    "?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminData()
			req := httptest.NewRequest(http.MethodDelete, "/api/admin/announcement"+tt.queryString, nil)
			w := httptest.NewRecorder()

			DeleteAnnouncement(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestGetSystemConfigs(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/configs", nil)
	w := httptest.NewRecorder()

	GetSystemConfigs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.SystemConfig
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 2)
}

func TestSetSystemConfig(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "set config with valid data",
			requestBody:    `{"key":"new_key","value":"new_value"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "set inventory threshold",
			requestBody:    `{"key":"inventory_threshold","value":"20"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "missing key",
			requestBody:    `{"value":"no key"}`,
			wantStatusCode: http.StatusBadRequest,
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
			resetAdminData()
			req := httptest.NewRequest(http.MethodPost, "/api/admin/config", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			SetSystemConfig(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestSetSystemConfigUpdatesThreshold(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	// 更新库存阈值
	req := httptest.NewRequest(http.MethodPost, "/api/admin/config",
		strings.NewReader(`{"key":"inventory_threshold","value":"5"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	SetSystemConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证阈值已更新
	req = httptest.NewRequest(http.MethodGet, "/api/admin/inventory/alerts", nil)
	w = httptest.NewRecorder()

	GetInventoryAlerts(w, req)

	var response []models.InventoryAlert
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	// 阈值变为 5，原来库存为 8 的猫粮不应该再触发预警
	assert.Len(t, response, 0)
}
