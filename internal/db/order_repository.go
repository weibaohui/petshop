package db

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"petshop/internal/models"
)

var errOrderRepositoryDBNotInitialized = errors.New("order repository database is not initialized")

// OrderRepository handles order database operations
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new OrderRepository
func NewOrderRepository() *OrderRepository {
	return &OrderRepository{db: GetDB()}
}

// NewOrderRepositoryWithDB creates a new OrderRepository with a specific database instance
func NewOrderRepositoryWithDB(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) ensureDB() error {
	if r == nil || r.db == nil {
		return errOrderRepositoryDBNotInitialized
	}
	return nil
}

// GetAll returns all orders
func (r *OrderRepository) GetAll() ([]*models.Order, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(`
		SELECT id, user_id, total_amount, status, COALESCE(refund_reason, ''), created_at, updated_at
		FROM orders ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

// GetByID returns an order by ID with its items
func (r *OrderRepository) GetByID(id int64) (*models.Order, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	o := &models.Order{}

	err := r.db.QueryRow(`
		SELECT id, user_id, total_amount, status, COALESCE(refund_reason, ''), created_at, updated_at
		FROM orders WHERE id = ?`, id).Scan(
		&o.ID, &o.UserID, &o.TotalAmount, &o.Status, &o.RefundReason, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Get order items
	items, err := r.getOrderItems(id)
	if err != nil {
		return nil, err
	}
	o.Products = items

	return o, nil
}

// GetByStatus returns orders filtered by status
func (r *OrderRepository) GetByStatus(status string) ([]*models.Order, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(`
		SELECT id, user_id, total_amount, status, COALESCE(refund_reason, ''), created_at, updated_at
		FROM orders WHERE status = ? ORDER BY id DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

// Create creates a new order with items in a transaction
func (r *OrderRepository) Create(o *models.Order) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	// Insert order
	result, err := tx.Exec(`
		INSERT INTO orders (user_id, total_amount, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		o.UserID, o.TotalAmount, o.Status, now, now)
	if err != nil {
		return err
	}

	o.ID, _ = result.LastInsertId()
	o.CreatedAt = now
	o.UpdatedAt = now

	// Insert order items
	for i := range o.Products {
		item := &o.Products[i]
		_, err := tx.Exec(`
			INSERT INTO order_items (order_id, product_id, product_name, price, quantity, subtotal)
			VALUES (?, ?, ?, ?, ?, ?)`,
			o.ID, item.ProductID, item.ProductName, item.Price, item.Quantity, item.Subtotal)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateStatus updates order status
func (r *OrderRepository) UpdateStatus(id int64, status string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	_, err := r.db.Exec(`
		UPDATE orders SET status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now(), id)
	return err
}

// UpdateRefund updates order refund status and reason
func (r *OrderRepository) UpdateRefund(id int64, reason string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	_, err := r.db.Exec(`
		UPDATE orders SET status = 'refunded', refund_reason = ?, updated_at = ? WHERE id = ?`,
		reason, time.Now(), id)
	return err
}

// HasPendingOrdersByProduct checks if a product has pending orders
func (r *OrderRepository) HasPendingOrdersByProduct(productID int64) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM order_items oi
		JOIN orders o ON oi.order_id = o.id
		WHERE oi.product_id = ? AND o.status NOT IN ('delivered', 'cancelled', 'refunded')`,
		productID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetOrderItemsByProduct returns order items for a product
func (r *OrderRepository) GetOrderItemsByProduct(productID int64) ([]models.OrderItem, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(`
		SELECT product_id, product_name, price, quantity, subtotal
		FROM order_items WHERE product_id = ?`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.Price, &item.Quantity, &item.Subtotal); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// getOrderItems returns all items for an order
func (r *OrderRepository) getOrderItems(orderID int64) ([]models.OrderItem, error) {
	rows, err := r.db.Query(`
		SELECT product_id, product_name, price, quantity, subtotal
		FROM order_items WHERE order_id = ?`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.Price, &item.Quantity, &item.Subtotal); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// getOrderItemsBatch returns all items for multiple orders in a single query
func (r *OrderRepository) getOrderItemsBatch(orderIDs []int64) (map[int64][]models.OrderItem, error) {
	if len(orderIDs) == 0 {
		return make(map[int64][]models.OrderItem), nil
	}

	// Build IN clause placeholders
	placeholders := make([]string, len(orderIDs))
	args := make([]interface{}, len(orderIDs))
	for i, id := range orderIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := "SELECT order_id, product_id, product_name, price, quantity, subtotal FROM order_items WHERE order_id IN (" +
		strings.Join(placeholders, ",") + ")"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	itemsMap := make(map[int64][]models.OrderItem)
	for rows.Next() {
		var orderID int64
		var item models.OrderItem
		if err := rows.Scan(&orderID, &item.ProductID, &item.ProductName, &item.Price, &item.Quantity, &item.Subtotal); err != nil {
			return nil, err
		}
		itemsMap[orderID] = append(itemsMap[orderID], item)
	}
	return itemsMap, rows.Err()
}

// scanOrders scans order rows
func (r *OrderRepository) scanOrders(rows *sql.Rows) ([]*models.Order, error) {
	var orders []*models.Order
	var orderIDs []int64

	// First, collect all orders
	for rows.Next() {
		o := &models.Order{}
		if err := rows.Scan(&o.ID, &o.UserID, &o.TotalAmount, &o.Status, &o.RefundReason, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
		orderIDs = append(orderIDs, o.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Close rows before executing another query
	rows.Close()

	// Batch fetch all order items in a single query
	itemsMap, err := r.getOrderItemsBatch(orderIDs)
	if err != nil {
		return nil, err
	}

	// Assign items to corresponding orders
	for i, orderID := range orderIDs {
		orders[i].Products = itemsMap[orderID]
	}

	return orders, nil
}
