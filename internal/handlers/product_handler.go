package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"petshop/internal/models"
)

// Product management functions

// CreateProductRequest represents the request body for creating a product.
type CreateProductRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Price       float64  `json:"price"`
	Stock       int      `json:"stock"`
	Images      []string `json:"images"`
}

// UpdateProductRequest represents the request body for updating a product.
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

// ListProducts handles GET /api/admin/products and returns all products.
// @Summary List products
// @Description Get a list of all products (admin only)
// @Tags admin-products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Product
// @Failure 401 {string} string "unauthorized"
// @Router /api/admin/products [get]
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

// GetProduct handles GET /api/admin/product?id=<id> and returns the product.
// @Summary Get product
// @Description Get a product by ID (admin only)
// @Tags admin-products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query int true "Product ID"
// @Success 200 {object} models.Product
// @Failure 400 {string} string "id is required"
// @Failure 404 {string} string "product not found"
// @Router /api/admin/product [get]
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

// CreateProduct handles POST /api/admin/products and creates a new product.
// @Summary Create product
// @Description Create a new product (admin only)
// @Tags admin-products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateProductRequest true "Product data"
// @Success 201 {object} models.Product
// @Failure 400 {string} string "invalid request body"
// @Router /api/admin/products [post]
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
		ID:          nextInventoryID,
		ProductID:   p.ID,
		ChangeType:  "in",
		Quantity:    req.Stock,
		BeforeStock: 0,
		AfterStock:  req.Stock,
		Reason:      "新增商品",
		Operator:    "system",
		CreatedAt:   now,
	})
	nextInventoryID++

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

// UpdateProduct handles PUT /api/admin/product and updates an existing product.
// @Summary Update product
// @Description Update an existing product (admin only)
// @Tags admin-products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateProductRequest true "Product data"
// @Success 200 {object} models.Product
// @Failure 400 {string} string "invalid request body"
// @Failure 404 {string} string "product not found"
// @Router /api/admin/product [put]
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
			// TODO: 触发库存预警逻辑
			_ = inventoryThreshold // 暂时使用变量避免未使用警告
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

// DeleteProduct handles DELETE /api/admin/product?id=<id> and marks a product as deleted.
// @Summary Delete product
// @Description Delete a product by ID (admin only)
// @Tags admin-products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query int true "Product ID"
// @Success 200 {object} map[string]string
// @Failure 400 {string} string "id is required"
// @Failure 404 {string} string "product not found"
// @Failure 409 {string} string "product has pending orders"
// @Router /api/admin/product [delete]
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
