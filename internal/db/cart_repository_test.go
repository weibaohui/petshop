package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

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
	_, err = db.Exec(schema)
	require.NoError(t, err)

	return db
}

// createTestRepository creates a CartRepository with the test database
func createTestRepository(t *testing.T) (*CartRepository, *sql.DB) {
	testDB := setupTestDB(t)
	repo := &CartRepository{db: testDB}
	return repo, testDB
}

func TestNewCartRepository(t *testing.T) {
	// This test just verifies the function doesn't panic
	// Since it depends on global db state, we can't fully test it
	assert.NotPanics(t, func() {
		_ = NewCartRepository()
	})
}

func TestNewCartRepositoryWithDB(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	repo := NewCartRepositoryWithDB(testDB)
	assert.NotNil(t, repo)

	// Verify the repository works by adding an item
	item, err := repo.AddCartItem(1, 1, "Test Product", 100.0, 2)
	assert.NoError(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, int64(1), item.UserID)
	assert.Equal(t, int64(1), item.ProductID)
}

func TestCartRepository_GetCartItems(t *testing.T) {
	repo, db := createTestRepository(t)
	defer db.Close()

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO cart_items (user_id, product_id, product_name, price, quantity)
		VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)`,
		1, 1, "Product 1", 100.0, 2,
		1, 2, "Product 2", 50.0, 1)
	require.NoError(t, err)

	tests := []struct {
		name           string
		userID         int64
		wantCount      int
		wantTotalPrice float64
	}{
		{
			name:           "get cart items for user with items",
			userID:         1,
			wantCount:      2,
			wantTotalPrice: 250.0, // 100*2 + 50*1
		},
		{
			name:           "get cart items for user without items",
			userID:         2,
			wantCount:      0,
			wantTotalPrice: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := repo.GetCartItems(tt.userID)
			require.NoError(t, err)
			assert.Len(t, items, tt.wantCount)

			var total float64
			for _, item := range items {
				total += item.Subtotal
				assert.Equal(t, item.Price*float64(item.Quantity), item.Subtotal)
			}
			assert.Equal(t, tt.wantTotalPrice, total)
		})
	}
}

func TestCartRepository_GetCartItems_Error(t *testing.T) {
	repo, db := createTestRepository(t)
	db.Close() // Close db to force error

	_, err := repo.GetCartItems(1)
	assert.Error(t, err)
}

func TestCartRepository_AddCartItem_NewItem(t *testing.T) {
	repo, db := createTestRepository(t)
	defer db.Close()

	item, err := repo.AddCartItem(1, 1, "Test Product", 99.99, 2)
	require.NoError(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, int64(1), item.UserID)
	assert.Equal(t, int64(1), item.ProductID)
	assert.Equal(t, "Test Product", item.ProductName)
	assert.Equal(t, 99.99, item.Price)
	assert.Equal(t, 2, item.Quantity)
	assert.Equal(t, 199.98, item.Subtotal)
}

func TestCartRepository_AddCartItem_UpdateExisting(t *testing.T) {
	repo, db := createTestRepository(t)
	defer db.Close()

	// First add an item
	_, err := repo.AddCartItem(1, 1, "Test Product", 100.0, 2)
	require.NoError(t, err)

	// Add the same product again - should update quantity
	item, err := repo.AddCartItem(1, 1, "Test Product", 100.0, 3)
	require.NoError(t, err)
	assert.Equal(t, 5, item.Quantity) // 2 + 3
	assert.Equal(t, 500.0, item.Subtotal)
}

func TestCartRepository_GetCartItemByProductID(t *testing.T) {
	repo, db := createTestRepository(t)
	defer db.Close()

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO cart_items (user_id, product_id, product_name, price, quantity)
		VALUES (?, ?, ?, ?, ?)`,
		1, 1, "Test Product", 99.99, 2)
	require.NoError(t, err)

	tests := []struct {
		name       string
		userID     int64
		productID  int64
		wantFound  bool
		wantName   string
		wantErrMsg string
	}{
		{
			name:      "get existing item",
			userID:    1,
			productID: 1,
			wantFound: true,
			wantName:  "Test Product",
		},
		{
			name:       "get non-existent item",
			userID:     1,
			productID:  999,
			wantFound:  false,
			wantErrMsg: "sql: no rows in result set",
		},
		{
			name:       "get item for wrong user",
			userID:     2,
			productID:  1,
			wantFound:  false,
			wantErrMsg: "sql: no rows in result set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := repo.GetCartItemByProductID(tt.userID, tt.productID)
			if tt.wantFound {
				require.NoError(t, err)
				assert.Equal(t, tt.wantName, item.ProductName)
				assert.Equal(t, item.Price*float64(item.Quantity), item.Subtotal)
			} else {
				assert.Error(t, err)
				assert.Nil(t, item)
			}
		})
	}
}

func TestCartRepository_GetCartItemByID(t *testing.T) {
	repo, db := createTestRepository(t)
	defer db.Close()

	// Insert test data
	result, err := db.Exec(`
		INSERT INTO cart_items (user_id, product_id, product_name, price, quantity)
		VALUES (?, ?, ?, ?, ?)`,
		1, 1, "Test Product", 99.99, 2)
	require.NoError(t, err)
	id, _ := result.LastInsertId()

	tests := []struct {
		name       string
		id         int64
		wantFound  bool
		wantName   string
	}{
		{
			name:      "get existing item",
			id:        id,
			wantFound: true,
			wantName:  "Test Product",
		},
		{
			name:      "get non-existent item",
			id:        999,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := repo.GetCartItemByID(tt.id)
			if tt.wantFound {
				require.NoError(t, err)
				assert.Equal(t, tt.wantName, item.ProductName)
				assert.Equal(t, item.Price*float64(item.Quantity), item.Subtotal)
			} else {
				assert.Error(t, err)
				assert.Nil(t, item)
			}
		})
	}
}

func TestCartRepository_UpdateCartItemQuantity(t *testing.T) {
	repo, db := createTestRepository(t)
	defer db.Close()

	// Insert test data
	result, err := db.Exec(`
		INSERT INTO cart_items (user_id, product_id, product_name, price, quantity)
		VALUES (?, ?, ?, ?, ?)`,
		1, 1, "Test Product", 100.0, 2)
	require.NoError(t, err)
	id, _ := result.LastInsertId()

	tests := []struct {
		name      string
		userID    int64
		itemID    int64
		quantity  int
		wantError bool
	}{
		{
			name:      "update existing item",
			userID:    1,
			itemID:    id,
			quantity:  5,
			wantError: false,
		},
		{
			name:      "update non-existent item",
			userID:    1,
			itemID:    999,
			quantity:  5,
			wantError: false, // No error returned even if no rows affected
		},
		{
			name:      "update item for wrong user",
			userID:    2,
			itemID:    id,
			quantity:  5,
			wantError: false, // No error returned even if no rows affected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateCartItemQuantity(tt.userID, tt.itemID, tt.quantity)
			assert.NoError(t, err)

			// Verify update for successful case
			if !tt.wantError && tt.userID == 1 && tt.itemID == id {
				item, err := repo.GetCartItemByID(id)
				require.NoError(t, err)
				assert.Equal(t, tt.quantity, item.Quantity)
			}
		})
	}
}

func TestCartRepository_RemoveCartItem(t *testing.T) {
	repo, db := createTestRepository(t)
	defer db.Close()

	// Insert test data for user 1
	result, err := db.Exec(`
		INSERT INTO cart_items (user_id, product_id, product_name, price, quantity)
		VALUES (?, ?, ?, ?, ?)`,
		1, 1, "Test Product", 100.0, 2)
	require.NoError(t, err)
	id, _ := result.LastInsertId()

	// Insert test data for user 2
	_, err = db.Exec(`
		INSERT INTO cart_items (user_id, product_id, product_name, price, quantity)
		VALUES (?, ?, ?, ?, ?)`,
		2, 2, "User2 Product", 50.0, 1)
	require.NoError(t, err)

	tests := []struct {
		name       string
		userID     int64
		itemID     int64
		shouldExist bool
	}{
		{
			name:       "remove existing item",
			userID:     1,
			itemID:     id,
			shouldExist: false,
		},
		{
			name:       "remove non-existent item",
			userID:     1,
			itemID:     999,
			shouldExist: false,
		},
		{
			name:       "remove item for wrong user",
			userID:     1,
			itemID:     id + 1, // This is user 2's item
			shouldExist: true,  // Should still exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.RemoveCartItem(tt.userID, tt.itemID)
			assert.NoError(t, err)

			// Verify removal
			_, err = repo.GetCartItemByID(tt.itemID)
			if tt.name == "remove existing item" {
				assert.Error(t, err) // Should be deleted
			} else if tt.name == "remove item for wrong user" {
				assert.NoError(t, err) // Should still exist
			}
		})
	}
}

func TestCartRepository_ClearCart(t *testing.T) {
	repo, db := createTestRepository(t)
	defer db.Close()

	// Insert test data for user 1
	_, err := db.Exec(`
		INSERT INTO cart_items (user_id, product_id, product_name, price, quantity)
		VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)`,
		1, 1, "Product 1", 100.0, 2,
		1, 2, "Product 2", 50.0, 1)
	require.NoError(t, err)

	// Insert test data for user 2
	_, err = db.Exec(`
		INSERT INTO cart_items (user_id, product_id, product_name, price, quantity)
		VALUES (?, ?, ?, ?, ?)`,
		2, 3, "User2 Product", 75.0, 3)
	require.NoError(t, err)

	// Clear cart for user 1
	err = repo.ClearCart(1)
	require.NoError(t, err)

	// Verify user 1's cart is empty
	items, err := repo.GetCartItems(1)
	require.NoError(t, err)
	assert.Len(t, items, 0)

	// Verify user 2's cart still has items
	items, err = repo.GetCartItems(2)
	require.NoError(t, err)
	assert.Len(t, items, 1)
}

func TestCartRepository_RemoveCartItems(t *testing.T) {
	repo, db := createTestRepository(t)
	defer db.Close()

	// Insert test data for user 1
	result1, _ := db.Exec(`INSERT INTO cart_items (user_id, product_id, product_name, price, quantity) VALUES (?, ?, ?, ?, ?)`,
		1, 1, "Product 1", 100.0, 2)
	id1, _ := result1.LastInsertId()

	result2, _ := db.Exec(`INSERT INTO cart_items (user_id, product_id, product_name, price, quantity) VALUES (?, ?, ?, ?, ?)`,
		1, 2, "Product 2", 50.0, 1)
	id2, _ := result2.LastInsertId()

	// Insert test data for user 2
	result3, _ := db.Exec(`INSERT INTO cart_items (user_id, product_id, product_name, price, quantity) VALUES (?, ?, ?, ?, ?)`,
		2, 3, "User2 Product", 75.0, 3)
	id3, _ := result3.LastInsertId()

	tests := []struct {
		name       string
		userID     int64
		itemIDs    []int64
		wantErr    bool
		remainingCount int
	}{
		{
			name:       "remove multiple items",
			userID:     1,
			itemIDs:    []int64{id1, id2},
			wantErr:    false,
			remainingCount: 0,
		},
		{
			name:       "remove with empty slice",
			userID:     1,
			itemIDs:    []int64{},
			wantErr:    false,
			remainingCount: 2, // Items still there from previous test run
		},
		{
			name:       "remove single item",
			userID:     2,
			itemIDs:    []int64{id3},
			wantErr:    false,
			remainingCount: 0,
		},
		{
			name:       "remove non-existent items",
			userID:     1,
			itemIDs:    []int64{999, 1000},
			wantErr:    false,
			remainingCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Re-setup data for the "empty slice" test
			if tt.name == "remove with empty slice" {
				db.Exec(`DELETE FROM cart_items WHERE user_id = 1`)
				db.Exec(`INSERT INTO cart_items (user_id, product_id, product_name, price, quantity) VALUES (?, ?, ?, ?, ?)`,
					1, 1, "Product 1", 100.0, 2)
				db.Exec(`INSERT INTO cart_items (user_id, product_id, product_name, price, quantity) VALUES (?, ?, ?, ?, ?)`,
					1, 2, "Product 2", 50.0, 1)
			}

			err := repo.RemoveCartItems(tt.userID, tt.itemIDs)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCartRepository_RemoveCartItems_NilSlice(t *testing.T) {
	repo, db := createTestRepository(t)
	defer db.Close()

	// Insert test data
	_, err := db.Exec(`INSERT INTO cart_items (user_id, product_id, product_name, price, quantity) VALUES (?, ?, ?, ?, ?)`,
		1, 1, "Product 1", 100.0, 2)
	require.NoError(t, err)

	// Call with nil slice - should not error and not remove anything
	err = repo.RemoveCartItems(1, nil)
	assert.NoError(t, err)

	// Verify item still exists
	items, err := repo.GetCartItems(1)
	require.NoError(t, err)
	assert.Len(t, items, 1)
}
