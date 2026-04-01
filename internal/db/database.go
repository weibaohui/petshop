package db

import (
	"database/sql"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db       *sql.DB
	once     sync.Once
	initErr  error
	initDone bool
)

// InitDB initializes the database connection
func InitDB(dbPath string) error {
	once.Do(func() {
		db, initErr = sql.Open("sqlite3", dbPath)
		if initErr != nil {
			initDone = true
			return
		}
		initErr = createTables()
		initDone = true
	})
	if !initDone || initErr != nil {
		return initErr
	}
	return nil
}

// GetDB returns the database instance
func GetDB() *sql.DB {
	return db
}

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
	`
	_, err := db.Exec(schema)
	return err
}

// Close closes the database connection
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}