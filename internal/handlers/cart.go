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

// GetCart handles GET /api/cart and returns the user's cart
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

// AddToCart handles POST /api/cart and adds an item to the cart
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

	if !exists || product.Status == "deleted" {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	// Verify stock
	if product.Stock < req.Quantity {
		http.Error(w, "insufficient stock", http.StatusBadRequest)
		return
	}

	// Use authoritative product name and price from server (issue #2)
	repo := db.NewCartRepository()
	item, err := repo.AddCartItem(userID, req.ProductID, product.Name, product.Price, req.Quantity)
	if err != nil {
		http.Error(w, "failed to add item to cart", http.StatusInternalServerError)
		return
	}

	items := []*models.CartItem{item}
	cart := calculateCart(userID, items)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Item added to cart", Cart: cart})
}

// UpdateCartItem handles PUT /api/cart and updates a cart item quantity
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

	if !exists || product.Status == "deleted" {
		http.Error(w, "product no longer available", http.StatusBadRequest)
		return
	}

	if product.Stock < req.Quantity {
		http.Error(w, "insufficient stock", http.StatusBadRequest)
		return
	}

	// Update quantity
	if err := repo.UpdateCartItemQuantity(userID, req.ItemID, req.Quantity); err != nil {
		http.Error(w, "failed to update cart", http.StatusInternalServerError)
		return
	}

	updatedItem, err := repo.GetCartItemByID(req.ItemID)
	if err != nil {
		http.Error(w, "failed to get updated item", http.StatusInternalServerError)
		return
	}

	items := []*models.CartItem{updatedItem}
	cart := calculateCart(userID, items)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Cart item updated", Cart: cart})
}

// DeleteCartItem handles DELETE /api/cart and removes items from the cart
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

	// Delete items
	for _, itemID := range req.ItemIDs {
		repo.RemoveCartItem(userID, itemID)
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

// ClearCart handles DELETE /api/cart/clear and removes all items from the user's cart
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

// parseUserID is kept for backwards compatibility but handlers should use context
func parseUserID(r *http.Request) (int64, error) {
	userIDStr := r.URL.Query().Get("userId")
	if userIDStr == "" {
		return 0, &parseError{message: "userId is required"}
	}

	var userID int64
	_, err := parseInt(userIDStr, &userID)
	if err != nil {
		return 0, &parseError{message: "invalid userId format"}
	}

	return userID, nil
}

type parseError struct {
	message string
}

func (e *parseError) Error() string {
	return e.message
}

// parseInt helper to parse string to int64
func parseInt(s string, target *int64) (int, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &parseError{message: "invalid number"}
		}
		n = n*10 + int64(c-'0')
	}
	*target = n
	return len(s), nil
}