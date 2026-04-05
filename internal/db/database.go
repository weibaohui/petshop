package db

import (
	"database/sql"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db      *sql.DB
	dbMu    sync.Mutex
	dbInited bool
	dbErr   error
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