package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"petshop/internal/models"
)

// ProductRepository handles product database operations
type ProductRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new ProductRepository
func NewProductRepository() *ProductRepository {
	return &ProductRepository{db: GetDB()}
}

// NewProductRepositoryWithDB creates a new ProductRepository with a specific database instance
func NewProductRepositoryWithDB(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// GetAll returns all non-deleted products
func (r *ProductRepository) GetAll() ([]*models.Product, error) {
	rows, err := r.db.Query(`
		SELECT id, name, description, category, price, stock, status, images, created_at, updated_at
		FROM products WHERE status != 'deleted' ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanProducts(rows)
}

// GetByID returns a product by ID
func (r *ProductRepository) GetByID(id int64) (*models.Product, error) {
	p := &models.Product{}
	var imagesJSON string

	err := r.db.QueryRow(`
		SELECT id, name, description, category, price, stock, status, images, created_at, updated_at
		FROM products WHERE id = ?`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Category, &p.Price, &p.Stock,
		&p.Status, &imagesJSON, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if imagesJSON != "" {
		json.Unmarshal([]byte(imagesJSON), &p.Images)
	}
	return p, nil
}

// Create creates a new product
func (r *ProductRepository) Create(p *models.Product) error {
	imagesJSON, _ := json.Marshal(p.Images)
	now := time.Now()

	result, err := r.db.Exec(`
		INSERT INTO products (name, description, category, price, stock, status, images, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Description, p.Category, p.Price, p.Stock, p.Status,
		string(imagesJSON), now, now)
	if err != nil {
		return err
	}

	p.ID, _ = result.LastInsertId()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

// Update updates a product
func (r *ProductRepository) Update(p *models.Product) error {
	imagesJSON, _ := json.Marshal(p.Images)
	now := time.Now()

	_, err := r.db.Exec(`
		UPDATE products SET name = ?, description = ?, category = ?, price = ?,
		stock = ?, status = ?, images = ?, updated_at = ?
		WHERE id = ?`,
		p.Name, p.Description, p.Category, p.Price, p.Stock, p.Status,
		string(imagesJSON), now, p.ID)
	if err != nil {
		return err
	}

	p.UpdatedAt = now
	return nil
}

// Delete marks a product as deleted
func (r *ProductRepository) Delete(id int64) error {
	_, err := r.db.Exec(`
		UPDATE products SET status = 'deleted', updated_at = ? WHERE id = ?`,
		time.Now(), id)
	return err
}

// UpdateStock updates product stock quantity
func (r *ProductRepository) UpdateStock(id int64, newStock int) error {
	_, err := r.db.Exec(`
		UPDATE products SET stock = ?, updated_at = ? WHERE id = ?`,
		newStock, time.Now(), id)
	return err
}

// GetLowStock returns products with stock below threshold
func (r *ProductRepository) GetLowStock(threshold int) ([]*models.Product, error) {
	rows, err := r.db.Query(`
		SELECT id, name, description, category, price, stock, status, images, created_at, updated_at
		FROM products WHERE status != 'deleted' AND stock <= ? ORDER BY stock ASC`, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanProducts(rows)
}

// scanProducts scans product rows
func (r *ProductRepository) scanProducts(rows *sql.Rows) ([]*models.Product, error) {
	var products []*models.Product
	for rows.Next() {
		p := &models.Product{}
		var imagesJSON string

		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.Price,
			&p.Stock, &p.Status, &imagesJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}

		if imagesJSON != "" {
			json.Unmarshal([]byte(imagesJSON), &p.Images)
		}
		products = append(products, p)
	}
	return products, rows.Err()
}
