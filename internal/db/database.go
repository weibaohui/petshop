package db

import (
	"database/sql"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db       *sql.DB
	dbMu     sync.Mutex
	dbInited bool
	dbErr    error
)

// InitDB initializes the database connection
func InitDB(dbPath string) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if dbInited {
		return dbErr
	}

	db, dbErr = sql.Open("sqlite3", dbPath)
	if dbErr != nil {
		dbInited = true
		return dbErr
	}

	dbErr = createTables()
	if dbErr != nil {
		db.Close()
		db = nil
		dbInited = true
		return dbErr
	}

	dbInited = true
	return nil
}

// GetDB returns the database instance
func GetDB() *sql.DB {
	return db
}

// createTables creates all database tables including cart_items and api_tokens
func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS cart_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		product_id INTEGER NOT NULL,
		product_name TEXT NOT NULL,
		price REAL NOT NULL,
		quantity INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, product_id)
	);
	CREATE INDEX IF NOT EXISTS idx_cart_user_id ON cart_items(user_id);

	CREATE TABLE IF NOT EXISTS api_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		description TEXT,
		status TEXT NOT NULL DEFAULT 'active',
		last_used_at DATETIME,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by INTEGER NOT NULL DEFAULT 0,
		permissions TEXT DEFAULT 'read'
	);
	CREATE INDEX IF NOT EXISTS idx_api_token_hash ON api_tokens(token_hash);
	CREATE INDEX IF NOT EXISTS idx_api_token_status ON api_tokens(status);

	-- Products table
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		category TEXT NOT NULL,
		price REAL NOT NULL,
		stock INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'on_sale',
		images TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_products_status ON products(status);
	CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);

	-- Orders table
	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		total_amount REAL NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		refund_reason TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
	CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

	-- Order items table
	CREATE TABLE IF NOT EXISTS order_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id INTEGER NOT NULL,
		product_id INTEGER NOT NULL,
		product_name TEXT NOT NULL,
		price REAL NOT NULL,
		quantity INTEGER NOT NULL,
		subtotal REAL NOT NULL,
		FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);

	-- Announcements table
	CREATE TABLE IF NOT EXISTS announcements (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_announcements_status ON announcements(status);

	-- Carousels table
	CREATE TABLE IF NOT EXISTS carousels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		image_url TEXT NOT NULL,
		link_url TEXT,
		sort_order INTEGER NOT NULL DEFAULT 0,
		title TEXT,
		status TEXT NOT NULL DEFAULT 'active'
	);
	CREATE INDEX IF NOT EXISTS idx_carousels_status ON carousels(status);

	-- Inventory logs table
	CREATE TABLE IF NOT EXISTS inventory_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER NOT NULL,
		change_type TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		before_stock INTEGER NOT NULL,
		after_stock INTEGER NOT NULL,
		reason TEXT,
		operator TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_inventory_logs_product_id ON inventory_logs(product_id);
	`
	_, err := db.Exec(schema)
	return err
}

// Close closes the database connection
func Close() error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if db != nil {
		err := db.Close()
		if err == nil {
			db = nil
			dbInited = false
		}
		return err
	}
	return nil
}

// ResetForTesting resets the database state for testing
// This should only be used in tests
func ResetForTesting() {
	dbMu.Lock()
	defer dbMu.Unlock()
	if db != nil {
		db.Close()
	}
	db = nil
	dbInited = false
	dbErr = nil
}
