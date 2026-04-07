package db

import (
	"database/sql"
	"time"

	"petshop/internal/models"
)

// InventoryRepository handles inventory log database operations
type InventoryRepository struct {
	db *sql.DB
}

// NewInventoryRepository creates a new InventoryRepository
func NewInventoryRepository() *InventoryRepository {
	return &InventoryRepository{db: GetDB()}
}

// NewInventoryRepositoryWithDB creates a new InventoryRepository with a specific database instance
func NewInventoryRepositoryWithDB(db *sql.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

// GetAll returns all inventory logs
func (r *InventoryRepository) GetAll() ([]models.Inventory, error) {
	rows, err := r.db.Query(`
		SELECT id, product_id, change_type, quantity, before_stock, after_stock, reason, operator, created_at
		FROM inventory_logs ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanInventoryLogs(rows)
}

// GetByProductID returns inventory logs for a product
func (r *InventoryRepository) GetByProductID(productID int64) ([]models.Inventory, error) {
	rows, err := r.db.Query(`
		SELECT id, product_id, change_type, quantity, before_stock, after_stock, reason, operator, created_at
		FROM inventory_logs WHERE product_id = ? ORDER BY id DESC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanInventoryLogs(rows)
}

// Create creates a new inventory log
func (r *InventoryRepository) Create(i *models.Inventory) error {
	now := time.Now()

	result, err := r.db.Exec(`
		INSERT INTO inventory_logs (product_id, change_type, quantity, before_stock, after_stock, reason, operator, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		i.ProductID, i.ChangeType, i.Quantity, i.BeforeStock, i.AfterStock, i.Reason, i.Operator, now)
	if err != nil {
		return err
	}

	i.ID, _ = result.LastInsertId()
	i.CreatedAt = now
	return nil
}

// CreateInTransaction creates an inventory log within a transaction
func (r *InventoryRepository) CreateInTransaction(tx *sql.Tx, i *models.Inventory) error {
	now := time.Now()

	result, err := tx.Exec(`
		INSERT INTO inventory_logs (product_id, change_type, quantity, before_stock, after_stock, reason, operator, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		i.ProductID, i.ChangeType, i.Quantity, i.BeforeStock, i.AfterStock, i.Reason, i.Operator, now)
	if err != nil {
		return err
	}

	i.ID, _ = result.LastInsertId()
	i.CreatedAt = now
	return nil
}

// scanInventoryLogs scans inventory log rows
func (r *InventoryRepository) scanInventoryLogs(rows *sql.Rows) ([]models.Inventory, error) {
	var logs []models.Inventory
	for rows.Next() {
		var i models.Inventory
		if err := rows.Scan(
			&i.ID, &i.ProductID, &i.ChangeType, &i.Quantity, &i.BeforeStock, &i.AfterStock,
			&i.Reason, &i.Operator, &i.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, i)
	}
	return logs, rows.Err()
}
