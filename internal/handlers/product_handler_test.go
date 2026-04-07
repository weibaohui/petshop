package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"petshop/internal/db"
	"petshop/internal/models"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupProductTestDB initializes the test database for product tests
func setupProductTestDB(t *testing.T) *sql.DB {
	database, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create tables
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

	CREATE TABLE IF NOT EXISTS inventory_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER NOT NULL,
		change_type TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		before_stock INTEGER NOT NULL,
		after_stock INTEGER NOT NULL,
		reason TEXT,
		operator TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = database.Exec(schema)
	require.NoError(t, err)

	// Initialize repositories with test database
	productRepo = db.NewProductRepositoryWithDB(database)
	orderRepo = db.NewOrderRepositoryWithDB(database)
	inventoryRepo = db.NewInventoryRepositoryWithDB(database)

	return database
}

// resetProductData resets the products data to initial state
func resetProductData(t *testing.T) {
	// Reset database and repositories
	setupProductTestDB(t)

	// Create test products
	productRepo.Create(&models.Product{
		Name:        "狗粮 10kg",
		Description: "优质狗粮",
		Category:    "狗粮",
		Price:       299.00,
		Stock:       50,
		Status:      "on_sale",
		Images:      []string{"/static/images/dog_food.jpg"},
	})
	productRepo.Create(&models.Product{
		Name:        "猫粮 5kg",
		Description: "天然猫粮",
		Category:    "猫粮",
		Price:       199.00,
		Stock:       8,
		Status:      "on_sale",
		Images:      []string{"/static/images/cat_food.jpg"},
	})
}

// ==================== ListProducts Tests ====================

func TestListProducts_Handler(t *testing.T) {
	tests := []struct {
		name           string
		setupData      func()
		wantStatusCode int
		wantCount      int
	}{
		{
			name: "list products successfully",
			setupData: func() {
				resetProductData(t)
			},
			wantStatusCode: http.StatusOK,
			wantCount:      2,
		},
		{
			name: "list products with empty database",
			setupData: func() {
				setupProductTestDB(t)
			},
			wantStatusCode: http.StatusOK,
			wantCount:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupData()

			req := httptest.NewRequest(http.MethodGet, "/api/admin/products", nil)
			w := httptest.NewRecorder()

			ListProducts(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var products []*models.Product
			err := json.NewDecoder(w.Body).Decode(&products)
			require.NoError(t, err)
			assert.Len(t, products, tt.wantCount)
		})
	}
}

func TestListProducts_Handler_DatabaseError(t *testing.T) {
	// Create a test with closed database to simulate error
	database := setupProductTestDB(t)
	database.Close() // Close database to cause error

	req := httptest.NewRequest(http.MethodGet, "/api/admin/products", nil)
	w := httptest.NewRecorder()

	ListProducts(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ==================== GetProduct Tests ====================

func TestGetProduct_Handler(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantErr        bool
		checkProduct   func(t *testing.T, p *models.Product)
	}{
		{
			name:           "get product successfully",
			queryString:    "?id=1",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkProduct: func(t *testing.T, p *models.Product) {
				assert.Equal(t, "狗粮 10kg", p.Name)
				assert.Equal(t, "狗粮", p.Category)
				assert.Equal(t, 299.00, p.Price)
				assert.Equal(t, 50, p.Stock)
			},
		},
		{
			name:           "get second product",
			queryString:    "?id=2",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkProduct: func(t *testing.T, p *models.Product) {
				assert.Equal(t, "猫粮 5kg", p.Name)
				assert.Equal(t, 199.00, p.Price)
			},
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
		{
			name:           "negative id",
			queryString:    "?id=-1",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "product not found",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "zero id",
			queryString:    "?id=0",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProductData(t)

			req := httptest.NewRequest(http.MethodGet, "/api/admin/product"+tt.queryString, nil)
			w := httptest.NewRecorder()

			GetProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var product models.Product
				err := json.NewDecoder(w.Body).Decode(&product)
				require.NoError(t, err)
				tt.checkProduct(t, &product)
			}
		})
	}
}

// ==================== CreateProduct Tests ====================

func TestCreateProduct_Handler(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, p *models.Product)
	}{
		{
			name:           "create product with valid data",
			requestBody:    `{"name":"鸟粮 1kg","description":"营养鸟粮","category":"鸟粮","price":59.99,"stock":100,"images":["/static/bird.jpg"]}`,
			wantStatusCode: http.StatusCreated,
			wantErr:        false,
			checkResponse: func(t *testing.T, p *models.Product) {
				assert.NotZero(t, p.ID)
				assert.Equal(t, "鸟粮 1kg", p.Name)
				assert.Equal(t, "营养鸟粮", p.Description)
				assert.Equal(t, "鸟粮", p.Category)
				assert.Equal(t, 59.99, p.Price)
				assert.Equal(t, 100, p.Stock)
				assert.Equal(t, "on_sale", p.Status)
			},
		},
		{
			name:           "create product with minimal fields",
			requestBody:    `{"name":"测试商品","price":10.0}`,
			wantStatusCode: http.StatusCreated,
			wantErr:        false,
			checkResponse: func(t *testing.T, p *models.Product) {
				assert.Equal(t, "测试商品", p.Name)
				assert.Equal(t, 10.0, p.Price)
				assert.Equal(t, 0, p.Stock)
				assert.Equal(t, "on_sale", p.Status)
			},
		},
		{
			name:           "missing name field",
			requestBody:    `{"description":"no name","price":10.0}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "empty name",
			requestBody:    `{"name":"","price":10.0}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "negative price",
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
		{
			name:           "empty request body",
			requestBody:    `{}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProductData(t)

			req := httptest.NewRequest(http.MethodPost, "/api/admin/products", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			CreateProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var product models.Product
				err := json.NewDecoder(w.Body).Decode(&product)
				require.NoError(t, err)
				tt.checkResponse(t, &product)
			}
		})
	}
}

func TestCreateProduct_Handler_WithInventoryLog(t *testing.T) {
	resetProductData(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/products",
		strings.NewReader(`{"name":"测试商品","description":"测试","category":"测试","price":99.99,"stock":50}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateProduct(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var product models.Product
	err := json.NewDecoder(w.Body).Decode(&product)
	require.NoError(t, err)
	assert.Equal(t, 50, product.Stock)

	// Verify inventory log was created
	logs, err := inventoryRepo.GetByProductID(product.ID)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "in", logs[0].ChangeType)
	assert.Equal(t, 50, logs[0].Quantity)
	assert.Equal(t, 0, logs[0].BeforeStock)
	assert.Equal(t, 50, logs[0].AfterStock)
	assert.Equal(t, "新增商品", logs[0].Reason)
	assert.Equal(t, "system", logs[0].Operator)
}

func TestCreateProduct_Handler_DatabaseError(t *testing.T) {
	database := setupProductTestDB(t)
	database.Close() // Close database to cause error

	req := httptest.NewRequest(http.MethodPost, "/api/admin/products",
		strings.NewReader(`{"name":"测试商品","price":10.0}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateProduct(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ==================== UpdateProduct Tests ====================

func TestUpdateProduct_Handler(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, p *models.Product)
	}{
		{
			name:           "update product name",
			requestBody:    `{"id":1,"name":"狗粮 20kg"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, p *models.Product) {
				assert.Equal(t, "狗粮 20kg", p.Name)
				// Other fields should remain unchanged
				assert.Equal(t, "狗粮", p.Category)
				assert.Equal(t, 299.00, p.Price)
			},
		},
		{
			name:           "update product price",
			requestBody:    `{"id":1,"price":399.00}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, p *models.Product) {
				assert.Equal(t, 399.00, p.Price)
			},
		},
		{
			name:           "update product stock",
			requestBody:    `{"id":1,"stock":60}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, p *models.Product) {
				assert.Equal(t, 60, p.Stock)
			},
		},
		{
			name:           "update multiple fields",
			requestBody:    `{"id":1,"name":"狗粮 20kg","price":399.00,"stock":60,"status":"on_sale"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, p *models.Product) {
				assert.Equal(t, "狗粮 20kg", p.Name)
				assert.Equal(t, 399.00, p.Price)
				assert.Equal(t, 60, p.Stock)
			},
		},
		{
			name:           "missing id",
			requestBody:    `{"name":"no id","price":10.0}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "zero id",
			requestBody:    `{"id":0,"name":"zero id"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "product not found",
			requestBody:    `{"id":999,"name":"not exist"}`,
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
			name:           "empty request body",
			requestBody:    `{}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProductData(t)

			req := httptest.NewRequest(http.MethodPut, "/api/admin/product", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var product models.Product
				err := json.NewDecoder(w.Body).Decode(&product)
				require.NoError(t, err)
				tt.checkResponse(t, &product)
			}
		})
	}
}

func TestUpdateProduct_Handler_StockChangeCreatesInventoryLog(t *testing.T) {
	resetProductData(t)

	// Update stock from 50 to 70 (increase by 20)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/product",
		strings.NewReader(`{"id":1,"stock":70}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateProduct(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var product models.Product
	err := json.NewDecoder(w.Body).Decode(&product)
	require.NoError(t, err)
	assert.Equal(t, 70, product.Stock)

	// Verify inventory log was created
	logs, err := inventoryRepo.GetByProductID(1)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "in", logs[0].ChangeType)
	assert.Equal(t, 20, logs[0].Quantity)
	assert.Equal(t, 50, logs[0].BeforeStock)
	assert.Equal(t, 70, logs[0].AfterStock)
	assert.Equal(t, "库存调整", logs[0].Reason)
	assert.Equal(t, "admin", logs[0].Operator)
}

func TestUpdateProduct_Handler_StockDecreaseCreatesInventoryLog(t *testing.T) {
	resetProductData(t)

	// Update stock from 50 to 30 (decrease by 20)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/product",
		strings.NewReader(`{"id":1,"stock":30}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateProduct(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify inventory log shows "out" for decrease
	logs, err := inventoryRepo.GetByProductID(1)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "out", logs[0].ChangeType)
	assert.Equal(t, 20, logs[0].Quantity)
	assert.Equal(t, 50, logs[0].BeforeStock)
	assert.Equal(t, 30, logs[0].AfterStock)
}

func TestUpdateProduct_Handler_NoStockChangeNoInventoryLog(t *testing.T) {
	resetProductData(t)

	// Update name but keep stock same (need to include stock value to avoid zero value)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/product",
		strings.NewReader(`{"id":1,"name":"新名称","stock":50}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateProduct(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify no inventory log was created
	logs, err := inventoryRepo.GetByProductID(1)
	require.NoError(t, err)
	assert.Len(t, logs, 0)
}

func TestUpdateProduct_Handler_DatabaseError(t *testing.T) {
	// This test verifies the error handling when database fails during update operation
	// We can't easily simulate a database error on Update() while GetByID() succeeds
	// in the current architecture, so we verify the behavior when GetByID fails
	database := setupProductTestDB(t)
	database.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/admin/product",
		strings.NewReader(`{"id":1,"name":"新名称"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateProduct(w, req)

	// When GetByID fails (database closed), it returns 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ==================== DeleteProduct Tests ====================

func TestDeleteProduct_Handler(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "delete product successfully",
			queryString:    "?id=2",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
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
		{
			name:           "negative id",
			queryString:    "?id=-1",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "zero id",
			queryString:    "?id=0",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "product not found",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProductData(t)

			req := httptest.NewRequest(http.MethodDelete, "/api/admin/product"+tt.queryString, nil)
			w := httptest.NewRecorder()

			DeleteProduct(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, "product deleted", response["message"])
			}
		})
	}
}

func TestDeleteProduct_Handler_StatusChangesToDeleted(t *testing.T) {
	resetProductData(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/product?id=1", nil)
	w := httptest.NewRecorder()

	DeleteProduct(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify product status is now "deleted"
	product, err := productRepo.GetByID(1)
	require.NoError(t, err)
	assert.Equal(t, "deleted", product.Status)
}

func TestDeleteProduct_Handler_WithPendingOrder(t *testing.T) {
	resetProductData(t)

	// Create a pending order for product 1
	orderRepo.Create(&models.Order{
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "pending",
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/product?id=1", nil)
	w := httptest.NewRecorder()

	DeleteProduct(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestDeleteProduct_Handler_WithPaidOrder(t *testing.T) {
	resetProductData(t)

	// Create a paid order for product 1
	orderRepo.Create(&models.Order{
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "paid",
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/product?id=1", nil)
	w := httptest.NewRecorder()

	DeleteProduct(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestDeleteProduct_Handler_WithShippedOrder(t *testing.T) {
	resetProductData(t)

	// Create a shipped order for product 1
	orderRepo.Create(&models.Order{
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "shipped",
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/product?id=1", nil)
	w := httptest.NewRecorder()

	DeleteProduct(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestDeleteProduct_Handler_CanDeleteWithDeliveredOrder(t *testing.T) {
	resetProductData(t)

	// Create a delivered order for product 1 (should not block deletion)
	orderRepo.Create(&models.Order{
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "delivered",
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/product?id=1", nil)
	w := httptest.NewRecorder()

	DeleteProduct(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteProduct_Handler_CanDeleteWithCancelledOrder(t *testing.T) {
	resetProductData(t)

	// Create a cancelled order for product 1 (should not block deletion)
	orderRepo.Create(&models.Order{
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "cancelled",
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/product?id=1", nil)
	w := httptest.NewRecorder()

	DeleteProduct(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteProduct_Handler_CanDeleteWithRefundedOrder(t *testing.T) {
	resetProductData(t)

	// Create a refunded order for product 1 (should not block deletion)
	orderRepo.Create(&models.Order{
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "refunded",
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/product?id=1", nil)
	w := httptest.NewRecorder()

	DeleteProduct(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteProduct_Handler_DatabaseErrorOnCheck(t *testing.T) {
	// This test verifies the error handling when database fails
	// When GetByID fails (database closed), it returns 404
	database := setupProductTestDB(t)
	database.Close()

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/product?id=1", nil)
	w := httptest.NewRecorder()

	DeleteProduct(w, req)

	// When GetByID fails (database closed), it returns 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteProduct_Handler_DeletedProductNotInList(t *testing.T) {
	resetProductData(t)

	// Delete product 1
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/product?id=1", nil)
	w := httptest.NewRecorder()
	DeleteProduct(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// List products should only return non-deleted products
	req = httptest.NewRequest(http.MethodGet, "/api/admin/products", nil)
	w = httptest.NewRecorder()
	ListProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var products []*models.Product
	err := json.NewDecoder(w.Body).Decode(&products)
	require.NoError(t, err)
	assert.Len(t, products, 1)
	assert.Equal(t, "猫粮 5kg", products[0].Name)
}
