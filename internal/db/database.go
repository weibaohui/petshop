package db

import (
	"database/sql"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	// DB is the global database instance
	DB       *sql.DB
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

	DB, dbErr = sql.Open("sqlite3", dbPath)
	if dbErr != nil {
		dbInited = true
		return dbErr
	}

	dbErr = createTables()
	if dbErr != nil {
		DB.Close()
		DB = nil
		dbInited = true
		return dbErr
	}

	dbInited = true
	return nil
}

// GetDB returns the database instance
func GetDB() *sql.DB {
	return DB
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

	-- 用户表（用于OTP登录）
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		email TEXT,
		phone TEXT,
		status TEXT DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- OTP表
	CREATE TABLE IF NOT EXISTS user_otp (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL UNIQUE,
		otp_secret TEXT NOT NULL,
		otp_enabled BOOLEAN DEFAULT 0,
		otp_enabled_at TIMESTAMP,
		backup_codes TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
	CREATE INDEX IF NOT EXISTS idx_user_otp_user_id ON user_otp(user_id);
	`
	_, err := DB.Exec(schema)
	return err
}

// Close closes the database connection
func Close() error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if DB != nil {
		err := DB.Close()
		if err == nil {
			DB = nil
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
	if DB != nil {
		DB.Close()
	}
	DB = nil
	dbInited = false
	dbErr = nil
}