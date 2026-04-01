package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"petshop/internal/models"
)

var cartMu sync.RWMutex

// Cart storage - in production this would be a database
// Map from userID to cart items
var userCarts = map[int64][]models.CartItem{}
var cartItemIDCounter int64 = 1

// GetCart handles GET /api/cart?userId=<userId> and returns the user's cart
func GetCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, err := parseUserID(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: err.Error()})
		return
	}

	cartMu.RLock()
	items := userCarts[userID]
	cartMu.RUnlock()

	cart := calculateCart(userID, items)
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Cart retrieved", Cart: cart})
}

// AddToCart handles POST /api/cart and adds an item to the user's cart
func AddToCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "Invalid request body"})
		return
	}

	var req models.AddToCartRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "Invalid JSON format"})
		return
	}

	if req.UserID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "userId is required"})
		return
	}

	if req.ProductID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "productId is required"})
		return
	}

	if req.Quantity <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "quantity must be greater than 0"})
		return
	}

	cartMu.Lock()
	defer cartMu.Unlock()

	items := userCarts[req.UserID]

	// Check if product already exists in cart
	for i, item := range items {
		if item.ProductID == req.ProductID {
			items[i].Quantity += req.Quantity
			items[i].Subtotal = items[i].Price * float64(items[i].Quantity)
			items[i].UpdatedAt = time.Now()
			userCarts[req.UserID] = items
			cart := calculateCart(req.UserID, items)
			json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Item quantity updated", Cart: cart})
			return
		}
	}

	// Add new item
	newItem := models.CartItem{
		ID:          cartItemIDCounter,
		UserID:      req.UserID,
		ProductID:   req.ProductID,
		ProductName: req.ProductName,
		Price:       req.Price,
		Quantity:    req.Quantity,
		Subtotal:    req.Price * float64(req.Quantity),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	cartItemIDCounter++
	userCarts[req.UserID] = append(items, newItem)

	cart := calculateCart(req.UserID, userCarts[req.UserID])
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Item added to cart", Cart: cart})
}

// UpdateCartItem handles PUT /api/cart and updates the quantity of an item
func UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "Invalid request body"})
		return
	}

	var req models.UpdateCartItemRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "Invalid JSON format"})
		return
	}

	if req.UserID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "userId is required"})
		return
	}

	if req.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "id is required"})
		return
	}

	if req.Quantity <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "quantity must be greater than 0"})
		return
	}

	cartMu.Lock()
	defer cartMu.Unlock()

	items := userCarts[req.UserID]
	found := false

	for i, item := range items {
		if item.ID == req.ID {
			items[i].Quantity = req.Quantity
			items[i].Subtotal = items[i].Price * float64(req.Quantity)
			items[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "Cart item not found"})
		return
	}

	userCarts[req.UserID] = items
	cart := calculateCart(req.UserID, items)
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Cart item updated", Cart: cart})
}

// DeleteCartItem handles DELETE /api/cart and removes items from the cart
func DeleteCartItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "Invalid request body"})
		return
	}

	var req models.DeleteCartItemRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "Invalid JSON format"})
		return
	}

	if req.UserID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "userId is required"})
		return
	}

	if len(req.IDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: "ids is required and must not be empty"})
		return
	}

	cartMu.Lock()
	defer cartMu.Unlock()

	items := userCarts[req.UserID]

	// Create a map for quick lookup
	idMap := make(map[int64]bool)
	for _, id := range req.IDs {
		idMap[id] = true
	}

	// Filter out items to delete
	filtered := make([]models.CartItem, 0, len(items))
	for _, item := range items {
		if !idMap[item.ID] {
			filtered = append(filtered, item)
		}
	}

	userCarts[req.UserID] = filtered
	cart := calculateCart(req.UserID, filtered)
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Items deleted", Cart: cart})
}

// ClearCart handles DELETE /api/cart/clear and removes all items from the user's cart
func ClearCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, err := parseUserID(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CartResponse{Success: false, Message: err.Error()})
		return
	}

	cartMu.Lock()
	defer cartMu.Unlock()

	delete(userCarts, userID)

	cart := calculateCart(userID, []models.CartItem{})
	json.NewEncoder(w).Encode(models.CartResponse{Success: true, Message: "Cart cleared", Cart: cart})
}

// calculateCart calculates total price and total items for a cart
func calculateCart(userID int64, items []models.CartItem) *models.Cart {
	var totalPrice float64
	var totalItems int

	for _, item := range items {
		totalPrice += item.Subtotal
		totalItems += item.Quantity
	}

	return &models.Cart{
		UserID:     userID,
		Items:      items,
		TotalPrice: totalPrice,
		TotalItems: totalItems,
		UpdatedAt:  time.Now(),
	}
}

// parseUserID extracts userID from query parameters
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