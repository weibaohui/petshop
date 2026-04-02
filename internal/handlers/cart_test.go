package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"petshop/internal/db"
	"petshop/internal/middleware"
	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupCartTestDB initializes the test database for cart tests
func setupCartTestDB(t *testing.T) {
	// Reset and initialize with in-memory database
	db.ResetForTesting()
	err := db.InitDB(":memory:")
	require.NoError(t, err)
}

// resetCartData resets the products data to initial state
func resetCartData() {
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
		Name:        "缺货商品",
		Description: "测试缺货",
		Category:    "测试",
		Price:       99.00,
		Stock:       0,
		Status:      "on_sale",
		Images:      []string{},
	}
	products[4] = &models.Product{
		ID:          4,
		Name:        "下架商品",
		Description: "测试下架",
		Category:    "测试",
		Price:       99.00,
		Stock:       10,
		Status:      "off_sale",
		Images:      []string{},
	}
	nextProductID = 5
}

// createRequestWithUser creates an HTTP request with user ID in context
func createRequestWithUser(method, path, body string, userID int64) *http.Request {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}

	req := httptest.NewRequest(method, path, bodyReader)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	return req.WithContext(ctx)
}

func TestGetCart(t *testing.T) {
	setupCartTestDB(t)
	resetCartData()

	tests := []struct {
		name           string
		userID         int64
		setupCart      func()
		wantStatusCode int
		wantSuccess    bool
		wantItemCount  int
		wantTotalItems int
		wantTotalPrice float64
	}{
		{
			name:           "get empty cart",
			userID:         1,
			setupCart:      func() {},
			wantStatusCode: http.StatusOK,
			wantSuccess:    true,
			wantItemCount:  0,
			wantTotalItems: 0,
			wantTotalPrice: 0,
		},
		{
			name:   "get cart with items",
			userID: 2,
			setupCart: func() {
				repo := db.NewCartRepository()
				_, err := repo.AddCartItem(2, 1, "狗粮 10kg", 299.00, 2)
				require.NoError(t, err)
			},
			wantStatusCode: http.StatusOK,
			wantSuccess:    true,
			wantItemCount:  1,
			wantTotalItems: 2,
			wantTotalPrice: 598.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupCart()

			req := createRequestWithUser(http.MethodGet, "/api/cart", "", tt.userID)
			w := httptest.NewRecorder()

			GetCart(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var response models.CartResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSuccess, response.Success)

			if tt.wantSuccess {
				assert.NotNil(t, response.Cart)
				assert.Len(t, response.Cart.Items, tt.wantItemCount)
				assert.Equal(t, tt.wantTotalItems, response.Cart.TotalItems)
				assert.Equal(t, tt.wantTotalPrice, response.Cart.TotalPrice)
			}
		})
	}
}

func TestGetCart_Unauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/cart", nil)
	w := httptest.NewRecorder()

	GetCart(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAddToCart(t *testing.T) {
	setupCartTestDB(t)
	resetCartData()

	tests := []struct {
		name           string
		userID         int64
		requestBody    string
		wantStatusCode int
		wantSuccess    bool
	}{
		{
			name:           "add item to cart",
			userID:         1,
			requestBody:    `{"productId":1,"quantity":2}`,
			wantStatusCode: http.StatusCreated,
			wantSuccess:    true,
		},
		{
			name:           "add item without quantity defaults to 1",
			userID:         1,
			requestBody:    `{"productId":2}`,
			wantStatusCode: http.StatusCreated,
			wantSuccess:    true,
		},
		{
			name:           "add item with negative quantity defaults to 1",
			userID:         1,
			requestBody:    `{"productId":2,"quantity":-1}`,
			wantStatusCode: http.StatusCreated,
			wantSuccess:    true,
		},
		{
			name:           "add non-existent product",
			userID:         1,
			requestBody:    `{"productId":999,"quantity":1}`,
			wantStatusCode: http.StatusNotFound,
			wantSuccess:    false,
		},
		{
			name:           "add out of stock product",
			userID:         1,
			requestBody:    `{"productId":3,"quantity":1}`,
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name:           "add off sale product",
			userID:         1,
			requestBody:    `{"productId":4,"quantity":1}`,
			wantStatusCode: http.StatusNotFound,
			wantSuccess:    false,
		},
		{
			name:           "missing productId",
			userID:         1,
			requestBody:    `{"quantity":1}`,
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name:           "invalid productId",
			userID:         1,
			requestBody:    `{"productId":-1,"quantity":1}`,
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name:           "invalid JSON",
			userID:         1,
			requestBody:    `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name:           "insufficient stock",
			userID:         1,
			requestBody:    `{"productId":2,"quantity":100}`,
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequestWithUser(http.MethodPost, "/api/cart", tt.requestBody, tt.userID)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			AddToCart(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantSuccess {
				var response models.CartResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.True(t, response.Success)
				assert.NotNil(t, response.Cart)
			}
		})
	}
}

func TestAddToCart_Unauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/cart", strings.NewReader(`{"productId":1,"quantity":1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	AddToCart(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAddToCart_UpdateExisting(t *testing.T) {
	setupCartTestDB(t)
	resetCartData()

	// Add item first time
	req := createRequestWithUser(http.MethodPost, "/api/cart", `{"productId":1,"quantity":2}`, 1)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	AddToCart(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Add same product again - should update quantity
	req = createRequestWithUser(http.MethodPost, "/api/cart", `{"productId":1,"quantity":3}`, 1)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	AddToCart(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.CartResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, 5, response.Cart.TotalItems) // 2 + 3
}

func TestUpdateCartItem(t *testing.T) {
	setupCartTestDB(t)
	resetCartData()

	tests := []struct {
		name           string
		setupItem      func() int64
		requestBody    func(itemID int64) string
		wantStatusCode int
		wantSuccess    bool
	}{
		{
			name: "update existing item",
			setupItem: func() int64 {
				repo := db.NewCartRepository()
				item, err := repo.AddCartItem(1, 2, "猫粮 5kg", 199.00, 1)
				require.NoError(t, err)
				return item.ID
			},
			requestBody: func(itemID int64) string {
				return `{"itemId":` + formatInt64(itemID) + `,"quantity":5}`
			},
			wantStatusCode: http.StatusOK,
			wantSuccess:    true,
		},
		{
			name: "update non-existent item",
			setupItem: func() int64 {
				return 0
			},
			requestBody: func(itemID int64) string {
				return `{"itemId":999,"quantity":5}`
			},
			wantStatusCode: http.StatusNotFound,
			wantSuccess:    false,
		},
		{
			name: "missing itemId",
			setupItem: func() int64 {
				return 0
			},
			requestBody: func(itemID int64) string {
				return `{"quantity":5}`
			},
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name: "invalid itemId",
			setupItem: func() int64 {
				return 0
			},
			requestBody: func(itemID int64) string {
				return `{"itemId":-1,"quantity":5}`
			},
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name: "zero quantity",
			setupItem: func() int64 {
				return 0
			},
			requestBody: func(itemID int64) string {
				return `{"itemId":1,"quantity":0}`
			},
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name: "negative quantity",
			setupItem: func() int64 {
				return 0
			},
			requestBody: func(itemID int64) string {
				return `{"itemId":1,"quantity":-1}`
			},
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name: "invalid JSON",
			setupItem: func() int64 {
				return 0
			},
			requestBody: func(itemID int64) string {
				return `{invalid}`
			},
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name: "update with insufficient stock",
			setupItem: func() int64 {
				repo := db.NewCartRepository()
				item, err := repo.AddCartItem(1, 2, "猫粮 5kg", 199.00, 1)
				require.NoError(t, err)
				return item.ID
			},
			requestBody: func(itemID int64) string {
				return `{"itemId":` + string(rune('0'+int(itemID))) + `,"quantity":100}`
			},
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			itemID := tt.setupItem()
			reqBody := tt.requestBody(itemID)

			req := createRequestWithUser(http.MethodPut, "/api/cart", reqBody, 1)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateCartItem(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantSuccess {
				var response models.CartResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.True(t, response.Success)
			}
		})
	}
}

func TestUpdateCartItem_Unauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/cart", strings.NewReader(`{"itemId":1,"quantity":5}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateCartItem(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateCartItem_ProductNotOnSale(t *testing.T) {
	setupCartTestDB(t)
	resetCartData()

	// Add item to cart using repository - product 4 is off_sale
	repo := db.NewCartRepository()
	item, err := repo.AddCartItem(1, 4, "下架商品", 99.00, 1)
	require.NoError(t, err)

	// Try to update item quantity
	reqBody := `{"itemId":` + formatInt64(item.ID) + `,"quantity":5}`
	req := createRequestWithUser(http.MethodPut, "/api/cart", reqBody, 1)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateCartItem(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}

func TestDeleteCartItem(t *testing.T) {
	setupCartTestDB(t)
	resetCartData()

	tests := []struct {
		name           string
		setupItems     func() (int64, int64)
		requestBody    func(id1, id2 int64) string
		wantStatusCode int
		wantSuccess    bool
	}{
		{
			name: "delete existing items",
			setupItems: func() (int64, int64) {
				repo := db.NewCartRepository()
				item1, _ := repo.AddCartItem(1, 1, "狗粮 10kg", 299.00, 1)
				return item1.ID, 0
			},
			requestBody: func(id1, id2 int64) string {
				return `{"itemIds":[` + formatInt64(id1) + `]}`
			},
			wantStatusCode: http.StatusOK,
			wantSuccess:    true,
		},
		{
			name: "delete multiple items",
			setupItems: func() (int64, int64) {
				repo := db.NewCartRepository()
				item1, _ := repo.AddCartItem(1, 1, "狗粮 10kg", 299.00, 1)
				item2, _ := repo.AddCartItem(1, 2, "猫粮 5kg", 199.00, 1)
				return item1.ID, item2.ID
			},
			requestBody: func(id1, id2 int64) string {
				return `{"itemIds":[` + formatInt64(id1) + `,` + formatInt64(id2) + `]}`
			},
			wantStatusCode: http.StatusOK,
			wantSuccess:    true,
		},
		{
			name: "delete non-existent items",
			setupItems: func() (int64, int64) {
				return 0, 0
			},
			requestBody: func(id1, id2 int64) string {
				return `{"itemIds":[999]}`
			},
			wantStatusCode: http.StatusNotFound,
			wantSuccess:    false,
		},
		{
			name: "empty itemIds",
			setupItems: func() (int64, int64) {
				return 0, 0
			},
			requestBody: func(id1, id2 int64) string {
				return `{"itemIds":[]}`
			},
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name: "missing itemIds",
			setupItems: func() (int64, int64) {
				return 0, 0
			},
			requestBody: func(id1, id2 int64) string {
				return `{}`
			},
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name: "invalid JSON",
			setupItems: func() (int64, int64) {
				return 0, 0
			},
			requestBody: func(id1, id2 int64) string {
				return `{invalid}`
			},
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id1, id2 := tt.setupItems()
			reqBody := tt.requestBody(id1, id2)

			req := createRequestWithUser(http.MethodDelete, "/api/cart", reqBody, 1)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			DeleteCartItem(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantSuccess {
				var response models.CartResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.True(t, response.Success)
			}
		})
	}
}

func TestDeleteCartItem_Unauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/cart", strings.NewReader(`{"itemIds":[1]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	DeleteCartItem(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestClearCart(t *testing.T) {
	setupCartTestDB(t)
	resetCartData()

	tests := []struct {
		name           string
		setupItems     func()
		wantStatusCode int
		wantEmpty      bool
	}{
		{
			name:           "clear empty cart",
			setupItems:     func() {},
			wantStatusCode: http.StatusOK,
			wantEmpty:      true,
		},
		{
			name: "clear cart with items",
			setupItems: func() {
				repo := db.NewCartRepository()
				_, err := repo.AddCartItem(1, 1, "狗粮 10kg", 299.00, 2)
				require.NoError(t, err)
				_, err = repo.AddCartItem(1, 2, "猫粮 5kg", 199.00, 1)
				require.NoError(t, err)
			},
			wantStatusCode: http.StatusOK,
			wantEmpty:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupItems()

			req := createRequestWithUser(http.MethodDelete, "/api/cart/clear", "", 1)
			w := httptest.NewRecorder()

			ClearCart(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var response models.CartResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)
			assert.True(t, response.Success)
			assert.Equal(t, 0, response.Cart.TotalItems)
		})
	}
}

func TestClearCart_Unauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/cart/clear", nil)
	w := httptest.NewRecorder()

	ClearCart(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCalculateCart(t *testing.T) {
	items := []*models.CartItem{
		{
			ID:          1,
			UserID:      1,
			ProductID:   1,
			ProductName: "Product 1",
			Price:       100.0,
			Quantity:    2,
		},
		{
			ID:          2,
			UserID:      1,
			ProductID:   2,
			ProductName: "Product 2",
			Price:       50.0,
			Quantity:    1,
		},
	}

	cart := calculateCart(1, items)

	assert.Equal(t, int64(1), cart.UserID)
	assert.Len(t, cart.Items, 2)
	assert.Equal(t, 3, cart.TotalItems)
	assert.Equal(t, 250.0, cart.TotalPrice)

	// Verify items are copied correctly
	assert.Equal(t, 200.0, cart.Items[0].Subtotal)
	assert.Equal(t, 50.0, cart.Items[1].Subtotal)
}

func TestCalculateCart_Empty(t *testing.T) {
	cart := calculateCart(1, []*models.CartItem{})

	assert.Equal(t, int64(1), cart.UserID)
	assert.Len(t, cart.Items, 0)
	assert.Equal(t, 0, cart.TotalItems)
	assert.Equal(t, 0.0, cart.TotalPrice)
}

func TestGetExistingCartQuantity(t *testing.T) {
	setupCartTestDB(t)
	resetCartData()

	// Test when item doesn't exist
	qty := getExistingCartQuantity(1, 1)
	assert.Equal(t, 0, qty)

	// Add item and test again
	repo := db.NewCartRepository()
	_, err := repo.AddCartItem(1, 1, "Product 1", 100.0, 3)
	require.NoError(t, err)

	qty = getExistingCartQuantity(1, 1)
	assert.Equal(t, 3, qty)
}
