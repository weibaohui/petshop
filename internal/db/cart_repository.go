package db

import (
	"database/sql"
	"time"

	"petshop/internal/models"
)

// CartRepository handles cart database operations
type CartRepository struct {
	db *sql.DB
}

// NewCartRepository creates a new CartRepository
func NewCartRepository() *CartRepository {
	return &CartRepository{db: GetDB()}
}

// GetCartItems returns all cart items for a user with full data copy (concurrency safe)
func (r *CartRepository) GetCartItems(userID int64) ([]*models.CartItem, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, product_id, product_name, price, quantity, created_at, updated_at
		FROM cart_items WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.CartItem
	for rows.Next() {
		item := &models.CartItem{}
		if err := rows.Scan(&item.ID, &item.UserID, &item.ProductID, &item.ProductName,
			&item.Price, &item.Quantity, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Subtotal = item.Price * float64(item.Quantity)
		// Create independent copy for each item (concurrency safety)
		itemCopy := *item
		items = append(items, &itemCopy)
	}
	return items, rows.Err()
}

// AddCartItem adds a new cart item or updates quantity if exists
func (r *CartRepository) AddCartItem(userID, productID int64, productName string, price float64, quantity int) (*models.CartItem, error) {
	now := time.Now()

	// Check if item already exists
	var existingID int64
	err := r.db.QueryRow(`SELECT id FROM cart_items WHERE user_id = ? AND product_id = ?`, userID, productID).Scan(&existingID)

	if err == nil {
		// Update existing item quantity
		_, err = r.db.Exec(`
			UPDATE cart_items SET quantity = quantity + ?, updated_at = ? WHERE id = ?`,
			quantity, now, existingID)
		if err != nil {
			return nil, err
		}
		return r.GetCartItemByID(existingID)
	} else if err == sql.ErrNoRows {
		// Insert new item
		result, err := r.db.Exec(`
			INSERT INTO cart_items (user_id, product_id, product_name, price, quantity, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			userID, productID, productName, price, quantity, now, now)
		if err != nil {
			return nil, err
		}
		id, _ := result.LastInsertId()
		return r.GetCartItemByID(id)
	}

	return nil, err
}

// GetCartItemByID returns a cart item by ID
func (r *CartRepository) GetCartItemByID(id int64) (*models.CartItem, error) {
	item := &models.CartItem{}
	err := r.db.QueryRow(`
		SELECT id, user_id, product_id, product_name, price, quantity, created_at, updated_at
		FROM cart_items WHERE id = ?`, id).
		Scan(&item.ID, &item.UserID, &item.ProductID, &item.ProductName,
			&item.Price, &item.Quantity, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	item.Subtotal = item.Price * float64(item.Quantity)
	return item, nil
}

// UpdateCartItemQuantity updates the quantity of a cart item
func (r *CartRepository) UpdateCartItemQuantity(userID, itemID int64, quantity int) error {
	_, err := r.db.Exec(`UPDATE cart_items SET quantity = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		quantity, time.Now(), itemID, userID)
	return err
}

// RemoveCartItem removes a cart item
func (r *CartRepository) RemoveCartItem(userID, itemID int64) error {
	_, err := r.db.Exec(`DELETE FROM cart_items WHERE id = ? AND user_id = ?`, itemID, userID)
	return err
}

// ClearCart removes all cart items for a user
func (r *CartRepository) ClearCart(userID int64) error {
	_, err := r.db.Exec(`DELETE FROM cart_items WHERE user_id = ?`, userID)
	return err
}