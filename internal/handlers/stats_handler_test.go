package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
)

// resetStatsData 重置统计数据相关的测试数据
func resetStatsData() {
	dataMu.Lock()
	defer dataMu.Unlock()

	// 重置产品数据
	products = make(map[int64]*models.Product)
	products[1] = &models.Product{
		ID:    1,
		Name:  "狗粮 10kg",
		Price: 299.00,
		Stock: 50,
	}
	products[2] = &models.Product{
		ID:    2,
		Name:  "猫粮 5kg",
		Price: 199.00,
		Stock: 30,
	}
	products[3] = &models.Product{
		ID:    3,
		Name:  "鸟粮 1kg",
		Price: 59.00,
		Stock: 100,
	}
	nextProductID = 4

	// 重置订单数据
	orders = make(map[int64]*models.Order)
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	weekAgo := today.AddDate(0, 0, -7)
	monthAgo := today.AddDate(0, -1, 0)

	// 今日订单
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

	// 昨日订单
	orders[2] = &models.Order{
		ID:     2,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 2, ProductName: "猫粮 5kg", Price: 199.00, Quantity: 3, Subtotal: 597.00},
		},
		TotalAmount: 597.00,
		Status:      "delivered",
		CreatedAt:   yesterday,
	}

	// 一周前的订单
	orders[3] = &models.Order{
		ID:     3,
		UserID: 2,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
			{ProductID: 3, ProductName: "鸟粮 1kg", Price: 59.00, Quantity: 5, Subtotal: 295.00},
		},
		TotalAmount: 594.00,
		Status:      "shipped",
		CreatedAt:   weekAgo,
	}

	// 一月前的订单
	orders[4] = &models.Order{
		ID:     4,
		UserID: 2,
		Products: []models.OrderItem{
			{ProductID: 3, ProductName: "鸟粮 1kg", Price: 59.00, Quantity: 10, Subtotal: 590.00},
		},
		TotalAmount: 590.00,
		Status:      "delivered",
		CreatedAt:   monthAgo,
	}

	// 被取消的订单（不应计入统计）
	orders[5] = &models.Order{
		ID:     5,
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "cancelled",
		CreatedAt:   today,
	}

	// 已退款的订单（不应计入统计）
	orders[6] = &models.Order{
		ID:     6,
		UserID: 2,
		Products: []models.OrderItem{
			{ProductID: 2, ProductName: "猫粮 5kg", Price: 199.00, Quantity: 1, Subtotal: 199.00},
		},
		TotalAmount: 199.00,
		Status:      "refunded",
		CreatedAt:   yesterday,
	}

	nextOrderID = 7
}

// ==================== GetSalesStats Tests ====================

func TestGetSalesStats_Day(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales?period=day", nil)
	w := httptest.NewRecorder()

	GetSalesStats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response []models.SalesStat
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 7) // 最近7天

	// 验证返回的数据结构
	for _, stat := range response {
		assert.NotEmpty(t, stat.Date)
		assert.GreaterOrEqual(t, stat.TotalSales, 0.0)
		assert.GreaterOrEqual(t, stat.OrderCount, 0)
	}
}

func TestGetSalesStats_Week(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales?period=week", nil)
	w := httptest.NewRecorder()

	GetSalesStats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response []models.SalesStat
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 4) // 最近4周
}

func TestGetSalesStats_Month(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales?period=month", nil)
	w := httptest.NewRecorder()

	GetSalesStats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response []models.SalesStat
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 6) // 最近6个月
}

func TestGetSalesStats_DefaultPeriod(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	// 不传递 period 参数，应该默认使用 day
	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales", nil)
	w := httptest.NewRecorder()

	GetSalesStats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.SalesStat
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 7) // 默认返回7天数据
}

func TestGetSalesStats_InvalidPeriod(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	tests := []struct {
		name         string
		period       string
		wantError    string
	}{
		{
			name:      "invalid period value",
			period:    "invalid",
			wantError: "invalid period",
		},
		{
			name:      "year period not supported",
			period:    "year",
			wantError: "invalid period",
		},
		{
			name:      "empty string after query",
			period:    "xyz",
			wantError: "invalid period",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetStatsData()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/sales?period="+tt.period, nil)
			w := httptest.NewRecorder()

			GetSalesStats(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantError, response["error"])
		})
	}
}

// ==================== GetHotProducts Tests ====================

func TestGetHotProducts_Basic(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	// 验证排序：按 SalesCount 降序排列
	// 预期结果：
	// 鸟粮 1kg (product 3): 15 件 (10 from order 4 + 5 from order 3)
	// 猫粮 5kg (product 2): 3 件 (from order 2)
	// 狗粮 10kg (product 1): 3 件 (2 from order 1 + 1 from order 3)
	if len(response) >= 2 {
		for i := 0; i < len(response)-1; i++ {
			assert.GreaterOrEqual(t, response[i].SalesCount, response[i+1].SalesCount,
				"Product at index %d should have SalesCount >= product at index %d", i, i+1)
		}
	}
}

func TestGetHotProducts_WithLimit(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products?limit=2", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 2) // 限制返回2个

	// 验证排序正确性
	if len(response) == 2 {
		assert.GreaterOrEqual(t, response[0].SalesCount, response[1].SalesCount)
	}
}

func TestGetHotProducts_InvalidLimit(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	// 无效的 limit 参数应该使用默认值 10
	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products?limit=abc", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	// 应该返回所有产品（因为产品数量小于默认 limit 10）
}

func TestGetHotProducts_SortOrder(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/hot-products", nil)
	w := httptest.NewRecorder()

	GetHotProducts(w, req)

	var response []models.HotProduct
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	// 验证具体的排序结果
	// 根据测试数据：
	// Product 1: 2 (order 1) + 1 (order 3) = 3
	// Product 2: 3 (order 2) = 3
	// Product 3: 5 (order 3) + 10 (order 4) = 15
	// 取消和退款的订单不应计入

	// 第一个应该是鸟粮（销量最高）
	if len(response) > 0 {
		assert.Equal(t, int64(3), response[0].ProductID)
		assert.Equal(t, "鸟粮 1kg", response[0].ProductName)
		assert.Equal(t, 15, response[0].SalesCount)
	}
}

// ==================== calculateDayStat Tests ====================

func TestCalculateDayStat(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	today := time.Now().Format("2006-01-02")

	stat := calculateDayStat(today)

	assert.Equal(t, today, stat.Date)
	// 今日有1个已支付订单（订单1，598元），1个被取消订单（不应计入）
	assert.Equal(t, 1, stat.OrderCount)
	assert.Equal(t, 598.00, stat.TotalSales)
}

func TestCalculateDayStat_NoOrders(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	// 使用一个很久以前的日期，应该没有订单
	oldDate := "2020-01-01"

	stat := calculateDayStat(oldDate)

	assert.Equal(t, oldDate, stat.Date)
	assert.Equal(t, 0, stat.OrderCount)
	assert.Equal(t, 0.0, stat.TotalSales)
}

func TestCalculateDayStat_ExcludesCancelledAndRefunded(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	today := time.Now().Format("2006-01-02")

	stat := calculateDayStat(today)

	// 今日订单：1个已支付（598元），1个已取消（299元，不计入）
	assert.Equal(t, 1, stat.OrderCount)
	assert.Equal(t, 598.00, stat.TotalSales)
}

// ==================== calculatePeriodStat Tests ====================

func TestCalculatePeriodStat(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	// 测试今天的范围
	today := time.Now()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 0, 1)

	stat := calculatePeriodStat(start, end)

	// 今天有1个已支付订单
	assert.Equal(t, 1, stat.OrderCount)
	assert.Equal(t, 598.00, stat.TotalSales)
}

func TestCalculatePeriodStat_MultipleDays(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	// 测试包含今天和昨天的范围
	today := time.Now()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, -1)
	end := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)

	stat := calculatePeriodStat(start, end)

	// 昨天有1个已支付订单（597元），今天有1个已支付订单（598元）
	// 昨天有1个已退款订单（不计入）
	assert.Equal(t, 2, stat.OrderCount)
	assert.Equal(t, 1195.00, stat.TotalSales)
}

func TestCalculatePeriodStat_ExcludesCancelledAndRefunded(t *testing.T) {
	resetStatsData()
	t.Cleanup(resetStatsData)

	// 获取 yesterday 的日期
	yesterday := time.Now().AddDate(0, 0, -1)
	// 计算 yesterday 的开始时间（00:00:00）
	start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.Local)
	// end 设为 yesterday（函数内部会使用 end.AddDate(0,0,1)，所以 yesterday 23:59:59 也会包含在内）
	end := start

	stat := calculatePeriodStat(start, end)

	// 昨天有1个已支付订单（597元），1个已退款订单（不计入）
	assert.Equal(t, 1, stat.OrderCount)
	assert.Equal(t, 597.00, stat.TotalSales)
}

// ==================== getWeekStart Tests ====================

func TestGetWeekStart(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected time.Time
	}{
		{
			name:     "Monday",
			date:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local), // 周一
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "Tuesday",
			date:     time.Date(2024, 1, 16, 10, 30, 0, 0, time.Local), // 周二
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "Wednesday",
			date:     time.Date(2024, 1, 17, 10, 30, 0, 0, time.Local), // 周三
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "Thursday",
			date:     time.Date(2024, 1, 18, 10, 30, 0, 0, time.Local), // 周四
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "Friday",
			date:     time.Date(2024, 1, 19, 10, 30, 0, 0, time.Local), // 周五
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "Saturday",
			date:     time.Date(2024, 1, 20, 10, 30, 0, 0, time.Local), // 周六
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "Sunday",
			date:     time.Date(2024, 1, 21, 10, 30, 0, 0, time.Local), // 周日
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWeekStart(tt.date)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetWeekStart_SundayEdgeCase(t *testing.T) {
	// 测试周日的情况（Weekday() == 0）
	sunday := time.Date(2024, 1, 21, 0, 0, 0, 0, time.Local)
	monday := getWeekStart(sunday)

	// 周日应该返回本周一（1月15日）
	expectedMonday := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	assert.Equal(t, expectedMonday, monday)
}

func TestGetWeekStart_MonthBoundary(t *testing.T) {
	// 测试跨月的情况
	// 2024年1月1日是周一
	jan1Input := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
	jan1Expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	result := getWeekStart(jan1Input)
	assert.Equal(t, jan1Expected, result)

	// 2024年1月7日是周日
	jan7Input := time.Date(2024, 1, 7, 12, 0, 0, 0, time.Local)
	result = getWeekStart(jan7Input)
	assert.Equal(t, jan1Expected, result)
}

func TestGetWeekStart_YearBoundary(t *testing.T) {
	// 测试跨年的情况
	// 2024年1月1日（周一）
	jan1Input := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
	jan1Expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	result := getWeekStart(jan1Input)
	assert.Equal(t, jan1Expected, result)
}
