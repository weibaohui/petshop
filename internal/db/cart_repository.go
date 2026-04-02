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

// NewCartRepositoryWithDB creates a new CartRepository with a specific database instance
// This is useful for testing with isolated database instances
func NewCartRepositoryWithDB(db *sql.DB) *CartRepository {
	return &CartRepository{db: db}
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

// AddCartItem adds a new cart item or updates quantity if exists (atomic upsert)
func (r *CartRepository) AddCartItem(userID, productID int64, productName string, price float64, quantity int) (*models.CartItem, error) {
	now := time.Now()

	// Use atomic upsert with INSERT ... ON CONFLICT
	result, err := r.db.Exec(`
		INSERT INTO cart_items (user_id, product_id, product_name, price, quantity, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, product_id) DO UPDATE SET
			quantity = quantity + excluded.quantity,
			updated_at = excluded.updated_at`,
		userID, productID, productName, price, quantity, now, now)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	// If LastInsertId returns 0 (some drivers don't support it), query by user_id + product_id
	if id == 0 {
		return r.GetCartItemByProductID(userID, productID)
	}
	return r.GetCartItemByID(id)
}

// GetCartItemByProductID returns a cart item by user ID and product ID
func (r *CartRepository) GetCartItemByProductID(userID, productID int64) (*models.CartItem, error) {
	item := &models.CartItem{}
	err := r.db.QueryRow(`
		SELECT id, user_id, product_id, product_name, price, quantity, created_at, updated_at
		FROM cart_items WHERE user_id = ? AND product_id = ?`, userID, productID).
		Scan(&item.ID, &item.UserID, &item.ProductID, &item.ProductName,
			&item.Price, &item.Quantity, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	item.Subtotal = item.Price * float64(item.Quantity)
	return item, nil
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

// RemoveCartItems atomically removes multiple cart items by their IDs
func (r *CartRepository) RemoveCartItems(userID int64, itemIDs []int64) error {
	if len(itemIDs) == 0 {
		return nil
	}
	query := `DELETE FROM cart_items WHERE user_id = ? AND id IN (`
	args := []interface{}{userID}
	for i, id := range itemIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, id)
	}
	query += ")"
	_, err := r.db.Exec(query, args...)
	return err
}