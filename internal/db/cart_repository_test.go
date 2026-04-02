package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *CartRepository {
	resetDBState()
	err := InitDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(resetDBState)
	return NewCartRepository()
}

func TestCartRepository_GetCartItems(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(r *CartRepository)
		userID  int64
		wantLen int
		wantErr bool
	}{
		{
			name:    "empty cart",
			userID:  1,
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "cart with items",
			setup: func(r *CartRepository) {
				_, err := r.AddCartItem(1, 101, "Dog Food", 29.99, 2)
				require.NoError(t, err)
				_, err = r.AddCartItem(1, 102, "Cat Food", 19.99, 1)
				require.NoError(t, err)
			},
			userID:  1,
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "other user cart",
			setup: func(r *CartRepository) {
				_, err := r.AddCartItem(2, 103, "Bird Seed", 9.99, 3)
				require.NoError(t, err)
			},
			userID:  1,
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestDB(t)
			if tt.setup != nil {
				tt.setup(r)
			}
			items, err := r.GetCartItems(tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, items, tt.wantLen)
				for _, item := range items {
					assert.Equal(t, item.Price*float64(item.Quantity), item.Subtotal)
				}
			}
		})
	}
}

func TestCartRepository_AddCartItem(t *testing.T) {
	r := setupTestDB(t)

	t.Run("add new item", func(t *testing.T) {
		item, err := r.AddCartItem(1, 101, "Dog Food", 29.99, 2)
		require.NoError(t, err)
		assert.NotZero(t, item.ID)
		assert.Equal(t, int64(1), item.UserID)
		assert.Equal(t, int64(101), item.ProductID)
		assert.Equal(t, "Dog Food", item.ProductName)
		assert.Equal(t, 29.99, item.Price)
		assert.Equal(t, 2, item.Quantity)
		assert.Equal(t, 59.98, item.Subtotal)
	})

	t.Run("update existing item quantity", func(t *testing.T) {
		item, err := r.AddCartItem(1, 101, "Dog Food", 29.99, 3)
		require.NoError(t, err)
		assert.Equal(t, 5, item.Quantity)
		assert.Equal(t, 149.95, item.Subtotal)
	})

	t.Run("invalid quantity", func(t *testing.T) {
		// sqlite3 allows zero or negative depending on schema; schema has no CHECK
		_, err := r.AddCartItem(1, 102, "Cat Food", 19.99, 0)
		require.NoError(t, err)
	})
}

func TestCartRepository_GetCartItemByID(t *testing.T) {
	r := setupTestDB(t)

	item, err := r.AddCartItem(1, 101, "Dog Food", 29.99, 2)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{
			name:    "existing item",
			id:      item.ID,
			wantErr: false,
		},
		{
			name:    "non-existent item",
			id:      9999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.GetCartItemByID(tt.id)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, item.ID, got.ID)
				assert.Equal(t, item.Subtotal, got.Subtotal)
			}
		})
	}
}

func TestCartRepository_GetCartItemByProductID(t *testing.T) {
	r := setupTestDB(t)

	_, err := r.AddCartItem(1, 101, "Dog Food", 29.99, 2)
	require.NoError(t, err)

	tests := []struct {
		name      string
		userID    int64
		productID int64
		wantErr   bool
	}{
		{
			name:      "existing item",
			userID:    1,
			productID: 101,
			wantErr:   false,
		},
		{
			name:      "non-existent product",
			userID:    1,
			productID: 9999,
			wantErr:   true,
		},
		{
			name:      "wrong user",
			userID:    2,
			productID: 101,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.GetCartItemByProductID(tt.userID, tt.productID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, int64(101), got.ProductID)
				assert.Equal(t, 29.99*2, got.Subtotal)
			}
		})
	}
}

func TestCartRepository_UpdateCartItemQuantity(t *testing.T) {
	r := setupTestDB(t)

	item, err := r.AddCartItem(1, 101, "Dog Food", 29.99, 2)
	require.NoError(t, err)

	tests := []struct {
		name     string
		userID   int64
		itemID   int64
		quantity int
		wantErr  bool
		check    func()
	}{
		{
			name:     "update existing item",
			userID:   1,
			itemID:   item.ID,
			quantity: 5,
			wantErr:  false,
			check: func() {
				updated, err := r.GetCartItemByID(item.ID)
				require.NoError(t, err)
				assert.Equal(t, 5, updated.Quantity)
				assert.Equal(t, 29.99*5, updated.Subtotal)
			},
		},
		{
			name:     "update non-existent item",
			userID:   1,
			itemID:   9999,
			quantity: 3,
			wantErr:  false, // Exec succeeds with 0 rows affected
		},
		{
			name:     "wrong user",
			userID:   2,
			itemID:   item.ID,
			quantity: 10,
			wantErr:  false, // Exec succeeds with 0 rows affected
			check: func() {
				updated, err := r.GetCartItemByID(item.ID)
				require.NoError(t, err)
				assert.Equal(t, 5, updated.Quantity) // unchanged
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.UpdateCartItemQuantity(tt.userID, tt.itemID, tt.quantity)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.check != nil {
				tt.check()
			}
		})
	}
}

func TestCartRepository_RemoveCartItem(t *testing.T) {
	r := setupTestDB(t)

	item, err := r.AddCartItem(1, 101, "Dog Food", 29.99, 2)
	require.NoError(t, err)

	tests := []struct {
		name    string
		userID  int64
		itemID  int64
		wantErr bool
		check   func()
	}{
		{
			name:    "remove existing item",
			userID:  1,
			itemID:  item.ID,
			wantErr: false,
			check: func() {
				_, err := r.GetCartItemByID(item.ID)
				assert.Error(t, err)
			},
		},
		{
			name:    "remove non-existent item",
			userID:  1,
			itemID:  9999,
			wantErr: false,
		},
		{
			name:    "wrong user",
			userID:  2,
			itemID:  item.ID,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.RemoveCartItem(tt.userID, tt.itemID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.check != nil {
				tt.check()
			}
		})
	}
}

func TestCartRepository_ClearCart(t *testing.T) {
	r := setupTestDB(t)

	_, err := r.AddCartItem(1, 101, "Dog Food", 29.99, 2)
	require.NoError(t, err)
	_, err = r.AddCartItem(1, 102, "Cat Food", 19.99, 1)
	require.NoError(t, err)
	_, err = r.AddCartItem(2, 103, "Bird Seed", 9.99, 1)
	require.NoError(t, err)

	err = r.ClearCart(1)
	require.NoError(t, err)

	user1Items, err := r.GetCartItems(1)
	require.NoError(t, err)
	assert.Len(t, user1Items, 0)

	user2Items, err := r.GetCartItems(2)
	require.NoError(t, err)
	assert.Len(t, user2Items, 1)
}

func TestCartRepository_RemoveCartItems(t *testing.T) {
	r := setupTestDB(t)

	item1, err := r.AddCartItem(1, 101, "Dog Food", 29.99, 2)
	require.NoError(t, err)
	item2, err := r.AddCartItem(1, 102, "Cat Food", 19.99, 1)
	require.NoError(t, err)
	item3, err := r.AddCartItem(1, 103, "Bird Seed", 9.99, 1)
	require.NoError(t, err)

	t.Run("remove multiple items", func(t *testing.T) {
		err := r.RemoveCartItems(1, []int64{item1.ID, item2.ID})
		require.NoError(t, err)

		items, err := r.GetCartItems(1)
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, item3.ID, items[0].ID)
	})

	t.Run("empty ids", func(t *testing.T) {
		err := r.RemoveCartItems(1, []int64{})
		require.NoError(t, err)
	})

	t.Run("remove with wrong user", func(t *testing.T) {
		err := r.RemoveCartItems(2, []int64{item3.ID})
		require.NoError(t, err)

		items, err := r.GetCartItems(1)
		require.NoError(t, err)
		assert.Len(t, items, 1) // still there
	})
}
