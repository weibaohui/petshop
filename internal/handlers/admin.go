package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"petshop/internal/models"
)

// Admin data storage
var (
	products      = make(map[int64]*models.Product)
	orders        = make(map[int64]*models.Order)
	users         = make(map[int64]*models.User)
	inventoryLogs = make([]models.Inventory, 0)
	carousels     = make(map[int64]*models.Carousel)
	announcements = make(map[int64]*models.Announcement)
	systemConfigs = make(map[string]string)

	dataMu sync.RWMutex
	nextProductID      int64 = 1
	nextOrderID        int64 = 1
	nextUserID         int64 = 1
	nextInventoryID    int64 = 1
	nextCarouselID     int64 = 1
	nextAnnouncementID int64 = 1

	// 库存预警阈值
	inventoryThreshold = 10
)

func init() {
	// 初始化示例数据
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

	users[1] = &models.User{
		ID:        1,
		Username:  "user1",
		Email:     "user1@example.com",
		Phone:     "13800138000",
		Status:    "active",
		Role:      "user",
		CreatedAt: time.Now(),
	}
	users[2] = &models.User{
		ID:        2,
		Username:  "user2",
		Email:     "user2@example.com",
		Phone:     "13800138001",
		Status:    "active",
		Role:      "user",
		CreatedAt: time.Now(),
	}
	nextUserID = 3

	orders[1] = &models.Order{
		ID:      1,
		UserID:  1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "paid",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	nextOrderID = 2

	carousels[1] = &models.Carousel{
		ID:        1,
		ImageURL:  "/static/carousel/banner1.jpg",
		LinkURL:   "/product/1",
		SortOrder: 1,
		Title:     "春季大促",
		Status:    "active",
	}
	nextCarouselID = 2

	announcements[1] = &models.Announcement{
		ID:        1,
		Title:     "春节放假通知",
		Content:   "春节期间客服工作时间调整",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	nextAnnouncementID = 2

	systemConfigs["site_name"] = "宠物商店"
	systemConfigs["inventory_threshold"] = "10"
}

// ==================== 商品管理 ====================

func ListProducts(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	productList := make([]*models.Product, 0, len(products))
	for _, p := range products {
		if p.Status != "deleted" {
			productList = append(productList, p)
		}
	}
	json.NewEncoder(w).Encode(productList)
}

func GetProduct(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dataMu.RLock()
	defer dataMu.RUnlock()

	if p, ok := products[id]; ok {
		json.NewEncoder(w).Encode(p)
		return
	}
	http.Error(w, "product not found", http.StatusNotFound)
}

type CreateProductRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Price       float64  `json:"price"`
	Stock       int      `json:"stock"`
	Images      []string `json:"images"`
}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req CreateProductRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Price <= 0 {
		http.Error(w, "price must be positive", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	now := time.Now()
	p := &models.Product{
		ID:          nextProductID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Price:       req.Price,
		Stock:       req.Stock,
		Status:      "on_sale",
		Images:      req.Images,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	products[nextProductID] = p
	nextProductID++

	// 记录库存入库
	inventoryLogs = append(inventoryLogs, models.Inventory{
		ID:         nextInventoryID,
		ProductID:  p.ID,
		ChangeType: "in",
		Quantity:   req.Stock,
		BeforeStock: 0,
		AfterStock:  req.Stock,
		Reason:     "新增商品",
		Operator:   "system",
		CreatedAt:  now,
	})
	nextInventoryID++

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

type UpdateProductRequest struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Price       float64  `json:"price"`
	Stock       int      `json:"stock"`
	Status      string   `json:"status"`
	Images      []string `json:"images"`
}

func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req UpdateProductRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ID == 0 {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	p, ok := products[req.ID]
	if !ok {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	// 检查库存变动
	stockChange := req.Stock - p.Stock
	if stockChange != 0 {
		changeType := "in"
		if stockChange < 0 {
			changeType = "out"
		}
		inventoryLogs = append(inventoryLogs, models.Inventory{
			ID:          nextInventoryID,
			ProductID:   p.ID,
			ChangeType:  changeType,
			Quantity:    abs(stockChange),
			BeforeStock: p.Stock,
			AfterStock:  req.Stock,
			Reason:      "库存调整",
			Operator:    "admin",
			CreatedAt:   time.Now(),
		})
		nextInventoryID++

		// 检查是否需要预警
		if req.Stock <= inventoryThreshold && p.Stock > inventoryThreshold {
			// 触发预警
		}
	}

	if req.Name != "" {
		p.Name = req.Name
	}
	if req.Description != "" {
		p.Description = req.Description
	}
	if req.Category != "" {
		p.Category = req.Category
	}
	if req.Price > 0 {
		p.Price = req.Price
	}
	p.Stock = req.Stock
	if req.Status != "" {
		p.Status = req.Status
	}
	if req.Images != nil {
		p.Images = req.Images
	}
	p.UpdatedAt = time.Now()

	json.NewEncoder(w).Encode(p)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	p, ok := products[id]
	if !ok {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	// 检查是否有未完成的订单
	for _, order := range orders {
		if order.Status != "delivered" && order.Status != "cancelled" && order.Status != "refunded" {
			for _, item := range order.Products {
				if item.ProductID == id {
					http.Error(w, "product has pending orders", http.StatusConflict)
					return
				}
			}
		}
	}

	p.Status = "deleted"
	p.UpdatedAt = time.Now()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "product deleted"})
}

// ==================== 库存管理 ====================

func ListInventoryLogs(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	json.NewEncoder(w).Encode(inventoryLogs)
}

func GetInventoryAlerts(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	alerts := make([]models.InventoryAlert, 0)
	for _, p := range products {
		if p.Status != "deleted" && p.Stock <= inventoryThreshold {
			alerts = append(alerts, models.InventoryAlert{
				ProductID:    p.ID,
				ProductName:  p.Name,
				Threshold:    inventoryThreshold,
				CurrentStock: p.Stock,
			})
		}
	}
	json.NewEncoder(w).Encode(alerts)
}

type InventoryAdjustRequest struct {
	ProductID int64  `json:"productId"`
	Quantity  int    `json:"quantity"`
	Reason    string `json:"reason"`
}

func AdjustInventory(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req InventoryAdjustRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	p, ok := products[req.ProductID]
	if !ok {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	beforeStock := p.Stock
	p.Stock += req.Quantity
	if p.Stock < 0 {
		p.Stock = 0
	}
	p.UpdatedAt = time.Now()

	changeType := "adjust"
	if req.Quantity > 0 {
		changeType = "in"
	} else if req.Quantity < 0 {
		changeType = "out"
	}

	inventoryLogs = append(inventoryLogs, models.Inventory{
		ID:          nextInventoryID,
		ProductID:   p.ID,
		ChangeType:  changeType,
		Quantity:    abs(req.Quantity),
		BeforeStock: beforeStock,
		AfterStock:  p.Stock,
		Reason:      req.Reason,
		Operator:    "admin",
		CreatedAt:   time.Now(),
	})
	nextInventoryID++

	json.NewEncoder(w).Encode(p)
}

// ==================== 订单管理 ====================

func ListOrders(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	status := r.URL.Query().Get("status")
	orderList := make([]*models.Order, 0, len(orders))

	for _, o := range orders {
		if status == "" || o.Status == status {
			orderList = append(orderList, o)
		}
	}
	json.NewEncoder(w).Encode(orderList)
}

func GetOrder(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dataMu.RLock()
	defer dataMu.RUnlock()

	if o, ok := orders[id]; ok {
		json.NewEncoder(w).Encode(o)
		return
	}
	http.Error(w, "order not found", http.StatusNotFound)
}

type UpdateOrderStatusRequest struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

func UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req UpdateOrderStatusRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	o, ok := orders[req.ID]
	if !ok {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	validStatuses := map[string]bool{
		"pending": true, "paid": true, "shipped": true, "delivered": true,
		"cancelled": true, "refunding": true, "refunded": true,
	}
	if !validStatuses[req.Status] {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	o.Status = req.Status
	o.UpdatedAt = time.Now()

	json.NewEncoder(w).Encode(o)
}

type RefundRequest struct {
	OrderID int64  `json:"orderId"`
	Reason  string `json:"reason"`
}

func ProcessRefund(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req RefundRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	o, ok := orders[req.OrderID]
	if !ok {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	o.Status = "refunded"
	o.RefundReason = req.Reason
	o.UpdatedAt = time.Now()

	// 恢复库存
	for _, item := range o.Products {
		if p, ok := products[item.ProductID]; ok {
			beforeStock := p.Stock
			p.Stock += item.Quantity
			inventoryLogs = append(inventoryLogs, models.Inventory{
				ID:          nextInventoryID,
				ProductID:   p.ID,
				ChangeType:  "in",
				Quantity:    item.Quantity,
				BeforeStock: beforeStock,
				AfterStock:  p.Stock,
				Reason:      fmt.Sprintf("退款返还: 订单%d", req.OrderID),
				Operator:    "system",
				CreatedAt:   time.Now(),
			})
			nextInventoryID++
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "refund processed"})
}

// ==================== 用户管理 ====================

func ListUsers(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	userList := make([]*models.User, 0, len(users))
	for _, u := range users {
		userList = append(userList, u)
	}
	json.NewEncoder(w).Encode(userList)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dataMu.RLock()
	defer dataMu.RUnlock()

	if u, ok := users[id]; ok {
		json.NewEncoder(w).Encode(u)
		return
	}
	http.Error(w, "user not found", http.StatusNotFound)
}

type UpdateUserStatusRequest struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

func UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req UpdateUserStatusRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	u, ok := users[req.ID]
	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if req.Status != "active" && req.Status != "disabled" {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	u.Status = req.Status
	json.NewEncoder(w).Encode(u)
}

type ResetPasswordRequest struct {
	UserID int64 `json:"userId"`
}

func ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req ResetPasswordRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	u, ok := users[req.UserID]
	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// 实际应用中这里会发送重置邮件或生成新密码
	// 简化处理，返回成功消息
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message":  fmt.Sprintf("密码已重置，用户%s的新密码已发送至邮箱", u.Username),
		"password": "reset123456",
	})
}

// ==================== 销售统计 ====================

func GetSalesStats(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	dataMu.RLock()
	defer dataMu.RUnlock()

	stats := make([]models.SalesStat, 0)

	switch period {
	case "day":
		// 最近7天日报
		for i := 6; i >= 0; i-- {
			date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			stat := calculateDayStat(date)
			stats = append(stats, stat)
		}
	case "week":
		// 最近4周周报
		for i := 3; i >= 0; i-- {
			weekStart := getWeekStart(time.Now().AddDate(0, 0, -i*7))
			weekEnd := weekStart.AddDate(0, 0, 6)
			stat := calculatePeriodStat(weekStart, weekEnd)
			stat.Date = weekStart.Format("2006-01-02")
			stats = append(stats, stat)
		}
	case "month":
		// 最近6月月报
		for i := 5; i >= 0; i-- {
			month := time.Now().AddDate(0, -i, 0)
			monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.Local)
			monthEnd := monthStart.AddDate(0, 1, -1)
			stat := calculatePeriodStat(monthStart, monthEnd)
			stat.Date = monthStart.Format("2006-01")
			stats = append(stats, stat)
		}
	}

	json.NewEncoder(w).Encode(stats)
}

func calculateDayStat(date string) models.SalesStat {
	stat := models.SalesStat{Date: date}

	for _, o := range orders {
		if o.Status != "cancelled" && o.Status != "refunded" {
			orderDate := o.CreatedAt.Format("2006-01-02")
			if orderDate == date {
				stat.TotalSales += o.TotalAmount
				stat.OrderCount++
			}
		}
	}
	return stat
}

func calculatePeriodStat(start, end time.Time) models.SalesStat {
	stat := models.SalesStat{Date: start.Format("2006-01-02")}

	for _, o := range orders {
		if o.Status != "cancelled" && o.Status != "refunded" {
			if o.CreatedAt.After(start) && o.CreatedAt.Before(end.AddDate(0, 0, 1)) {
				stat.TotalSales += o.TotalAmount
				stat.OrderCount++
			}
		}
	}
	return stat
}

func getWeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, time.Local)
}

func GetHotProducts(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	dataMu.RLock()
	defer dataMu.RUnlock()

	productSales := make(map[int64]struct {
		name   string
		count  int
		amount float64
	})

	for _, o := range orders {
		if o.Status != "cancelled" && o.Status != "refunded" {
			for _, item := range o.Products {
				if ps, ok := productSales[item.ProductID]; ok {
					ps.count += item.Quantity
					ps.amount += item.Subtotal
					productSales[item.ProductID] = ps
				} else {
					productSales[item.ProductID] = struct {
						name   string
						count  int
						amount float64
					}{
						name:   item.ProductName,
						count:  item.Quantity,
						amount: item.Subtotal,
					}
				}
			}
		}
	}

	hotList := make([]models.HotProduct, 0, len(productSales))
	for pid, ps := range productSales {
		hotList = append(hotList, models.HotProduct{
			ProductID:   pid,
			ProductName: ps.name,
			SalesCount:  ps.count,
			SalesAmount: ps.amount,
		})
	}

	// 排序
	for i := 0; i < len(hotList)-1; i++ {
		for j := i + 1; j < len(hotList); j++ {
			if hotList[j].SalesCount > hotList[i].SalesCount {
				hotList[i], hotList[j] = hotList[j], hotList[i]
			}
		}
	}

	if len(hotList) > limit {
		hotList = hotList[:limit]
	}

	json.NewEncoder(w).Encode(hotList)
}

// ==================== 系统配置 ====================

func ListCarousels(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	carouselList := make([]*models.Carousel, 0, len(carousels))
	for _, c := range carousels {
		carouselList = append(carouselList, c)
	}
	json.NewEncoder(w).Encode(carouselList)
}

type CreateCarouselRequest struct {
	ImageURL  string `json:"imageUrl"`
	LinkURL   string `json:"linkUrl"`
	SortOrder int    `json:"sortOrder"`
	Title     string `json:"title"`
}

func CreateCarousel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req CreateCarouselRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	c := &models.Carousel{
		ID:        nextCarouselID,
		ImageURL:  req.ImageURL,
		LinkURL:   req.LinkURL,
		SortOrder: req.SortOrder,
		Title:     req.Title,
		Status:    "active",
	}
	carousels[nextCarouselID] = c
	nextCarouselID++

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

func UpdateCarousel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var c models.Carousel
	if err := json.Unmarshal(body, &c); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	if existing, ok := carousels[c.ID]; ok {
		if c.ImageURL != "" {
			existing.ImageURL = c.ImageURL
		}
		if c.LinkURL != "" {
			existing.LinkURL = c.LinkURL
		}
		existing.SortOrder = c.SortOrder
		if c.Title != "" {
			existing.Title = c.Title
		}
		if c.Status != "" {
			existing.Status = c.Status
		}
		json.NewEncoder(w).Encode(existing)
		return
	}
	http.Error(w, "carousel not found", http.StatusNotFound)
}

func DeleteCarousel(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	if _, ok := carousels[id]; ok {
		delete(carousels, id)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
		return
	}
	http.Error(w, "carousel not found", http.StatusNotFound)
}

func ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	announcementList := make([]*models.Announcement, 0, len(announcements))
	for _, a := range announcements {
		announcementList = append(announcementList, a)
	}
	json.NewEncoder(w).Encode(announcementList)
}

type CreateAnnouncementRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req CreateAnnouncementRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	now := time.Now()
	dataMu.Lock()
	defer dataMu.Unlock()

	a := &models.Announcement{
		ID:        nextAnnouncementID,
		Title:     req.Title,
		Content:   req.Content,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	announcements[nextAnnouncementID] = a
	nextAnnouncementID++

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

func UpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var a models.Announcement
	if err := json.Unmarshal(body, &a); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	if existing, ok := announcements[a.ID]; ok {
		if a.Title != "" {
			existing.Title = a.Title
		}
		if a.Content != "" {
			existing.Content = a.Content
		}
		if a.Status != "" {
			existing.Status = a.Status
		}
		existing.UpdatedAt = time.Now()
		json.NewEncoder(w).Encode(existing)
		return
	}
	http.Error(w, "announcement not found", http.StatusNotFound)
}

func DeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	if _, ok := announcements[id]; ok {
		delete(announcements, id)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
		return
	}
	http.Error(w, "announcement not found", http.StatusNotFound)
}

func GetSystemConfigs(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	configs := make([]models.SystemConfig, 0, len(systemConfigs))
	for k, v := range systemConfigs {
		configs = append(configs, models.SystemConfig{Key: k, Value: v})
	}
	json.NewEncoder(w).Encode(configs)
}

type SetSystemConfigRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func SetSystemConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req SetSystemConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	systemConfigs[req.Key] = req.Value

	// 更新库存预警阈值
	if req.Key == "inventory_threshold" {
		if v, err := strconv.Atoi(req.Value); err == nil {
			inventoryThreshold = v
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "config updated"})
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
