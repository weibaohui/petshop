// Package handlers provides HTTP handlers for the petshop API.
//
// @Description 购物车相关处理器
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"petshop/internal/db"
	"petshop/internal/middleware"
	"petshop/internal/models"
)

// Cart handlers use database for persistence and get user ID from context

type AddToCartRequest struct {
	ProductID int64 `json:"productId"`
	Quantity  int   `json:"quantity"`
}

type UpdateCartItemRequest struct {
	ItemID   int64 `json:"itemId"`
	Quantity int   `json:"quantity"`
}

type DeleteCartItemRequest struct {
	ItemIDs []int64 `json:"itemIds"`
}

// GetCart 获取当前用户的购物车
// @Summary 获取购物车
// @Description 获取当前登录用户的购物车内容
// @Tags 购物车
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.CartResponse "购物车信息"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/cart [get]
func GetCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	repo := db.NewCartRepository()
	items, err := repo.GetCartItems(userID)
	if err != nil {
		http.Error(w, "failed to get cart", http.StatusInternalServerError)
		return
	}

	cart := calculateCart(userID, items)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Cart retrieved", Cart: cart})
}

// AddToCart 添加商品到购物车
// @Summary 添加商品到购物车
// @Description 将商品添加到当前用户的购物车
// @Tags 购物车
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AddToCartRequest true "添加商品请求"
// @Success 201 {object} models.CartResponse "添加成功后的购物车"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "商品不存在"
// @Router /api/cart [post]
func AddToCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req AddToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ProductID <= 0 {
		http.Error(w, "productId is required", http.StatusBadRequest)
		return
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	// Get product from server data to verify price (issue #2: price verification)
	dataMu.RLock()
	product, exists := products[req.ProductID]
	dataMu.RUnlock()

	if !exists || product.Status != "on_sale" {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	// Verify stock - consider existing cart quantity
	dataMu.RLock()
	existingQty := getExistingCartQuantity(userID, req.ProductID)
	dataMu.RUnlock()
	if product.Stock < req.Quantity+existingQty {
		http.Error(w, "insufficient stock", http.StatusBadRequest)
		return
	}

	// Use authoritative product name and price from server (issue #2)
	repo := db.NewCartRepository()
	_, err := repo.AddCartItem(userID, req.ProductID, product.Name, product.Price, req.Quantity)
	if err != nil {
		http.Error(w, "failed to add item to cart", http.StatusInternalServerError)
		return
	}

	// Return full cart
	items, err := repo.GetCartItems(userID)
	if err != nil {
		http.Error(w, "failed to get cart", http.StatusInternalServerError)
		return
	}
	cart := calculateCart(userID, items)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Item added to cart", Cart: cart})
}

// UpdateCartItem 更新购物车商品数量
// @Summary 更新购物车商品数量
// @Description 更新购物车中指定商品的数量
// @Tags 购物车
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateCartItemRequest true "更新请求"
// @Success 200 {object} models.CartResponse "更新后的购物车"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "购物车项不存在"
// @Router /api/cart [put]
func UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdateCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ItemID <= 0 {
		http.Error(w, "itemId is required", http.StatusBadRequest)
		return
	}
	if req.Quantity <= 0 {
		http.Error(w, "quantity must be positive", http.StatusBadRequest)
		return
	}

	repo := db.NewCartRepository()

	// Verify the item belongs to the user
	cartItems, err := repo.GetCartItems(userID)
	if err != nil {
		http.Error(w, "failed to get cart", http.StatusInternalServerError)
		return
	}

	var itemToUpdate *models.CartItem
	for _, item := range cartItems {
		if item.ID == req.ItemID {
			itemCopy := *item
			itemToUpdate = &itemCopy
			break
		}
	}

	if itemToUpdate == nil {
		http.Error(w, "cart item not found", http.StatusNotFound)
		return
	}

	// Verify stock for the product
	dataMu.RLock()
	product, exists := products[itemToUpdate.ProductID]
	dataMu.RUnlock()

	if !exists || product.Status != "on_sale" {
		http.Error(w, "product no longer available", http.StatusBadRequest)
		return
	}

	// Stock check: the new quantity should be available (not added to existing since we're replacing)
	if product.Stock < req.Quantity {
		http.Error(w, "insufficient stock", http.StatusBadRequest)
		return
	}

	// Update quantity
	if err := repo.UpdateCartItemQuantity(userID, req.ItemID, req.Quantity); err != nil {
		http.Error(w, "failed to update cart", http.StatusInternalServerError)
		return
	}

	// Return full cart after update
	items, err := repo.GetCartItems(userID)
	if err != nil {
		http.Error(w, "failed to get cart", http.StatusInternalServerError)
		return
	}
	cart := calculateCart(userID, items)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Cart item updated", Cart: cart})
}

// DeleteCartItem 从购物车删除商品
// @Summary 从购物车删除商品
// @Description 从购物车中删除指定商品
// @Tags 购物车
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body DeleteCartItemRequest true "删除请求"
// @Success 200 {object} models.CartResponse "删除后的购物车"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "购物车项不存在"
// @Router /api/cart [delete]
func DeleteCartItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req DeleteCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.ItemIDs) == 0 {
		http.Error(w, "itemIds is required and must not be empty", http.StatusBadRequest)
		return
	}

	repo := db.NewCartRepository()

	// Verify items belong to the user
	cartItems, err := repo.GetCartItems(userID)
	if err != nil {
		http.Error(w, "failed to get cart", http.StatusInternalServerError)
		return
	}

	idMap := make(map[int64]bool)
	for _, id := range req.ItemIDs {
		idMap[id] = true
	}

	foundCount := 0
	for _, item := range cartItems {
		if idMap[item.ID] {
			foundCount++
		}
	}

	if foundCount == 0 {
		http.Error(w, "cart items not found", http.StatusNotFound)
		return
	}

	// Delete items atomically
	if err := repo.RemoveCartItems(userID, req.ItemIDs); err != nil {
		http.Error(w, "failed to delete items", http.StatusInternalServerError)
		return
	}

	// Get remaining items
	remainingItems, err := repo.GetCartItems(userID)
	if err != nil {
		http.Error(w, "failed to get cart", http.StatusInternalServerError)
		return
	}

	cart := calculateCart(userID, remainingItems)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Items deleted", Cart: cart})
}

// ClearCart 清空购物车
// @Summary 清空购物车
// @Description 清空当前用户的购物车中的所有商品
// @Tags 购物车
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.CartResponse "清空后的购物车"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/cart/clear [delete]
func ClearCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	repo := db.NewCartRepository()
	if err := repo.ClearCart(userID); err != nil {
		http.Error(w, "failed to clear cart", http.StatusInternalServerError)
		return
	}

	cart := calculateCart(userID, []*models.CartItem{})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Cart cleared", Cart: cart})
}

// calculateCart calculates total price and total items for a cart (issue #4: full data copy)
func calculateCart(userID int64, items []*models.CartItem) *models.Cart {
	var totalPrice float64
	var totalItems int

	// Create full independent copies of items for concurrency safety (issue #4)
	copiedItems := make([]models.CartItem, 0, len(items))
	for _, item := range items {
		itemCopy := models.CartItem{
			ID:          item.ID,
			UserID:      item.UserID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Price:       item.Price,
			Quantity:    item.Quantity,
			Subtotal:    item.Price * float64(item.Quantity),
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		}
		copiedItems = append(copiedItems, itemCopy)
		totalPrice += itemCopy.Subtotal
		totalItems += itemCopy.Quantity
	}

	return &models.Cart{
		UserID:     userID,
		Items:      copiedItems,
		TotalPrice: totalPrice,
		TotalItems: totalItems,
		UpdatedAt:  time.Now(),
	}
}

// getExistingCartQuantity gets the quantity of a product already in user's cart
func getExistingCartQuantity(userID, productID int64) int {
	repo := db.NewCartRepository()
	item, err := repo.GetCartItemByProductID(userID, productID)
	if err != nil {
		return 0
	}
	return item.Quantity
}