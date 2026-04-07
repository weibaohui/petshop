package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetStatsData resets the global in-memory data to a known initial state for stats tests.
func resetStatsData() {
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
	today := time.Now()

	// 今天的订单
	orders[1] = &models.Order{
		ID:     1,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 2, Subtotal: 598.00},
		},
		TotalAmount: 598.00,
		Status:      "paid",
		CreatedAt:   today,
	}

	// 昨天的订单
	orders[2] = &models.Order{
		ID:     2,
		UserID: 2,
		Products: []models.OrderItem{
			{ProductID: 2, ProductName: "猫粮 5kg", Price: 199.00, Quantity: 1, Subtotal: 199.00},
		},
		TotalAmount: 199.00,
		Status:      "delivered",
		CreatedAt:   today.AddDate(0, 0, -1),
	}

	// 3天前的订单
	orders[3] = &models.Order{
		ID:     3,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
			{ProductID: 2, ProductName: "猫粮 5kg", Price: 199.00, Quantity: 2, Subtotal: 398.00},
		},
		TotalAmount: 697.00,
		Status:      "paid",
		CreatedAt:   today.AddDate(0, 0, -3),
	}

	// 取消的订单（不应该被统计）
	orders[4] = &models.Order{
		ID:     4,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "cancelled",
		CreatedAt:   today,
	}

	// 退款订单（不应该被统计）
	orders[5] = &models.Order{
		ID:     5,
		UserID: 2,
		Products: []models.OrderItem{
			{ProductID: 2, ProductName: "猫粮 5kg", Price: 199.00, Quantity: 1, Subtotal: 199.00},
		},
		TotalAmount: 199.00,
		Status:      "refunded",
		CreatedAt:   today.AddDate(0, 0, -2),
	}

	nextOrderID = 6

	inventoryLogs = make([]models.Inventory, 0)
	nextInventoryID = 1
}

// resetEmptyData resets data with no orders
func resetEmptyData() {
	dataMu.Lock()
	defer dataMu.Unlock()

	orders = make(map[int64]*models.Order)
	nextOrderID = 1
}

func TestStatsHandler_GetSalesStats_Day(t *testing.T) {
	resetStatsData()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales?period=day", nil)
	w := httptest.NewRecorder()

	GetSalesStats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.SalesStat
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 应该返回7天的数据
	assert.Len(t, result, 7)

	// 检查今天的数据（有1个有效订单，总额598）
	todayStr := time.Now().Format("2006-01-02")
	foundToday := false
	for _, stat := range result {
		if stat.Date == todayStr {
			foundToday = true
			assert.Equal(t, 1, stat.OrderCount)
			assert.InDelta(t, 598.00, stat.TotalSales, 0.01)
		}
	}
	assert.True(t, foundToday, "should find today's stat")

	// 检查昨天的数据
	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	for _, stat := range result {
		if stat.Date == yesterdayStr {
			assert.Equal(t, 1, stat.OrderCount)
			assert.InDelta(t, 199.00, stat.TotalSales, 0.01)
		}
	}
}

func TestStatsHandler_GetSalesStats_Week(t *testing.T) {
	resetStatsData()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales?period=week", nil)
	w := httptest.NewRecorder()

	GetSalesStats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.SalesStat
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 应该返回4周的数据
	assert.Len(t, result, 4)
}

func TestStatsHandler_GetSalesStats_Month(t *testing.T) {
	resetStatsData()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales?period=month", nil)
	w := httptest.NewRecorder()

	GetSalesStats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.SalesStat
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 应该返回6个月的数据
	assert.Len(t, result, 6)
}

func TestStatsHandler_GetSalesStats_DefaultPeriod(t *testing.T) {
	resetStatsData()

	// 不提供period参数，应该默认使用day
	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales", nil)
	w := httptest.NewRecorder()

	GetSalesStats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.SalesStat
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 默认返回7天的数据
	assert.Len(t, result, 7)
}

func TestStatsHandler_GetSalesStats_EmptyData(t *testing.T) {
	resetEmptyData()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales?period=day", nil)
	w := httptest.NewRecorder()

	GetSalesStats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.SalesStat
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 应该返回7天的数据，但所有数据都是0
	assert.Len(t, result, 7)
	for _, stat := range result {
		assert.Equal(t, 0, stat.OrderCount)
		assert.InDelta(t, 0.0, stat.TotalSales, 0.01)
	}
}

func TestStatsHandler_GetHotProducts_DefaultLimit(t *testing.T) {
	resetStatsData()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 应该返回2个商品
	assert.Len(t, result, 2)

	// 猫粮应该排在第一位（销量3 > 狗粮销量3，但猫粮订单更多）
	// 实际上：狗粮在订单1中有2个，在订单3中有1个，共3个
	// 猫粮在订单2中有1个，在订单3中有2个，共3个
	// 由于排序是按照SalesCount降序，相同数量时顺序不确定

	// 验证返回的商品数据
	var dogFood, catFood *models.HotProduct
	for i := range result {
		if result[i].ProductID == 1 {
			dogFood = &result[i]
		}
		if result[i].ProductID == 2 {
			catFood = &result[i]
		}
	}

	require.NotNil(t, dogFood)
	require.NotNil(t, catFood)

	assert.Equal(t, "狗粮 10kg", dogFood.ProductName)
	assert.Equal(t, 3, dogFood.SalesCount)               // 2 + 1
	assert.InDelta(t, 897.00, dogFood.SalesAmount, 0.01) // 598 + 299

	assert.Equal(t, "猫粮 5kg", catFood.ProductName)
	assert.Equal(t, 3, catFood.SalesCount)               // 1 + 2
	assert.InDelta(t, 597.00, catFood.SalesAmount, 0.01) // 199 + 398
}

func TestStatsHandler_GetHotProducts_WithLimit(t *testing.T) {
	resetStatsData()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products?limit=1", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 应该只返回1个商品
	assert.Len(t, result, 1)
}

func TestStatsHandler_GetHotProducts_InvalidLimit(t *testing.T) {
	resetStatsData()

	// 无效的limit参数应该使用默认值10
	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products?limit=abc", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 应该返回所有商品（最多10个）
	assert.Len(t, result, 2)
}

func TestStatsHandler_GetHotProducts_LimitZero(t *testing.T) {
	resetStatsData()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products?limit=0", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// limit为0应该返回空列表
	assert.Len(t, result, 0)
}

func TestStatsHandler_GetHotProducts_EmptyData(t *testing.T) {
	resetEmptyData()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 空数据应该返回空列表
	assert.Len(t, result, 0)
}

func TestStatsHandler_CalculateDayStat(t *testing.T) {
	resetStatsData()

	tests := []struct {
		name       string
		date       string
		wantOrders int
		wantSales  float64
	}{
		{
			name:       "today with orders",
			date:       time.Now().Format("2006-01-02"),
			wantOrders: 1, // 只有订单1（598），订单4被取消
			wantSales:  598.00,
		},
		{
			name:       "yesterday with orders",
			date:       time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
			wantOrders: 1, // 订单2
			wantSales:  199.00,
		},
		{
			name:       "3 days ago with multiple products",
			date:       time.Now().AddDate(0, 0, -3).Format("2006-01-02"),
			wantOrders: 1, // 订单3
			wantSales:  697.00,
		},
		{
			name:       "date with no orders",
			date:       "2020-01-01",
			wantOrders: 0,
			wantSales:  0.0,
		},
		{
			name:       "2 days ago with refunded order",
			date:       time.Now().AddDate(0, 0, -2).Format("2006-01-02"),
			wantOrders: 0, // 订单5是refunded状态，不应统计
			wantSales:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetStatsData()
			stat := calculateDayStat(tt.date)

			assert.Equal(t, tt.date, stat.Date)
			assert.Equal(t, tt.wantOrders, stat.OrderCount)
			assert.InDelta(t, tt.wantSales, stat.TotalSales, 0.01)
		})
	}
}

func TestStatsHandler_CalculatePeriodStat(t *testing.T) {
	resetStatsData()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	tests := []struct {
		name       string
		start      time.Time
		end        time.Time
		wantOrders int
		wantSales  float64
	}{
		{
			name:       "single day period",
			start:      today,
			end:        today,
			wantOrders: 1, // 只有订单1，订单4被取消
			wantSales:  598.00,
		},
		{
			name:       "3 days period",
			start:      today.AddDate(0, 0, -3),
			end:        today,
			wantOrders: 3,       // 订单1, 2, 3
			wantSales:  1494.00, // 598 + 199 + 697
		},
		{
			name:       "period with no orders",
			start:      today.AddDate(0, 0, -30),
			end:        today.AddDate(0, 0, -20),
			wantOrders: 0,
			wantSales:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetStatsData()
			stat := calculatePeriodStat(tt.start, tt.end)

			assert.Equal(t, tt.start.Format("2006-01-02"), stat.Date)
			assert.Equal(t, tt.wantOrders, stat.OrderCount)
			assert.InDelta(t, tt.wantSales, stat.TotalSales, 0.01)
		})
	}
}

func TestStatsHandler_GetWeekStart(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "Monday",
			input:    time.Date(2024, 1, 8, 10, 30, 0, 0, time.Local), // Monday
			expected: time.Date(2024, 1, 8, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "Tuesday",
			input:    time.Date(2024, 1, 9, 10, 30, 0, 0, time.Local), // Tuesday
			expected: time.Date(2024, 1, 8, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "Wednesday",
			input:    time.Date(2024, 1, 10, 10, 30, 0, 0, time.Local), // Wednesday
			expected: time.Date(2024, 1, 8, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "Sunday",
			input:    time.Date(2024, 1, 14, 10, 30, 0, 0, time.Local), // Sunday
			expected: time.Date(2024, 1, 8, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "Saturday",
			input:    time.Date(2024, 1, 13, 10, 30, 0, 0, time.Local), // Saturday
			expected: time.Date(2024, 1, 8, 0, 0, 0, 0, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWeekStart(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatsHandler_GetHotProducts_ExcludedStatuses(t *testing.T) {
	// 测试被取消和退款的订单不被计入热销商品
	dataMu.Lock()
	orders = make(map[int64]*models.Order)

	today := time.Now()

	// 添加一个正常订单
	orders[1] = &models.Order{
		ID:     1,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "paid",
		CreatedAt:   today,
	}

	// 添加一个被取消的订单，包含不同商品
	orders[2] = &models.Order{
		ID:     2,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 99, ProductName: "被取消商品", Price: 999.00, Quantity: 100, Subtotal: 99900.00},
		},
		TotalAmount: 99900.00,
		Status:      "cancelled",
		CreatedAt:   today,
	}

	// 添加一个退款订单
	orders[3] = &models.Order{
		ID:     3,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 88, ProductName: "退款商品", Price: 888.00, Quantity: 50, Subtotal: 44400.00},
		},
		TotalAmount: 44400.00,
		Status:      "refunded",
		CreatedAt:   today,
	}
	dataMu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 只有正常订单的商品被计入
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), result[0].ProductID)
	assert.Equal(t, "狗粮 10kg", result[0].ProductName)
	assert.Equal(t, 1, result[0].SalesCount)
}

func TestStatsHandler_GetHotProducts_LimitLargerThanData(t *testing.T) {
	resetStatsData()

	// limit大于实际商品数量
	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products?limit=100", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 应该返回所有可用商品（2个）
	assert.Len(t, result, 2)
}

func TestStatsHandler_GetHotProducts_LimitEdgeCase(t *testing.T) {
	resetStatsData()

	// limit为1，测试边界情况
	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products?limit=1", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	// 应该只返回1个商品
	assert.Len(t, result, 1)
}
