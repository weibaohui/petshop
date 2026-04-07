package models

import "time"

// CartItem represents a single item in the shopping cart
type CartItem struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"userId"`      // User ID for persistence
	ProductID   int64     `json:"productId"`   // Product ID
	ProductName string    `json:"productName"` // Product name
	Price       float64   `json:"price"`       // Unit price
	Quantity    int       `json:"quantity"`    // Quantity of this item
	Subtotal    float64   `json:"subtotal"`    // Calculated subtotal (price * quantity)
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Cart represents a user's shopping cart
type Cart struct {
	UserID     int64      `json:"userId"`
	Items      []CartItem `json:"items"`
	TotalPrice float64    `json:"totalPrice"` // Sum of all item subtotals
	TotalItems int        `json:"totalItems"` // Total number of items
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// AddToCartRequest represents the request to add an item to cart
type AddToCartRequest struct {
	UserID      int64   `json:"userId"`
	ProductID   int64   `json:"productId"`
	ProductName string  `json:"productName"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
}

// UpdateCartItemRequest represents the request to update item quantity
type UpdateCartItemRequest struct {
	UserID   int64 `json:"userId"`
	ID       int64 `json:"id"`
	Quantity int   `json:"quantity"`
}

// DeleteCartItemRequest represents the request to delete items
type DeleteCartItemRequest struct {
	UserID int64   `json:"userId"`
	IDs    []int64 `json:"ids"` // IDs of items to delete
}

// CartResponse represents the API response for cart operations
type CartResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Cart    *Cart  `json:"cart,omitempty"`
}
