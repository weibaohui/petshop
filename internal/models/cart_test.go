package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCartItemStruct(t *testing.T) {
	t.Run("CartItem JSON serialization and deserialization", func(t *testing.T) {
		item := CartItem{
			ID:          1,
			UserID:      100,
			ProductID:   200,
			ProductName: "Premium Dog Food",
			Price:       29.99,
			Quantity:    2,
			Subtotal:    59.98,
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(item)
		assert.NoError(t, err)

		var decoded CartItem
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, item.ID, decoded.ID)
		assert.Equal(t, item.UserID, decoded.UserID)
		assert.Equal(t, item.ProductID, decoded.ProductID)
		assert.Equal(t, item.ProductName, decoded.ProductName)
		assert.Equal(t, item.Price, decoded.Price)
		assert.Equal(t, item.Quantity, decoded.Quantity)
		assert.Equal(t, item.Subtotal, decoded.Subtotal)
		assert.Equal(t, item.CreatedAt, decoded.CreatedAt)
		assert.Equal(t, item.UpdatedAt, decoded.UpdatedAt)
	})

	t.Run("CartItem JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"id":2,"userId":101,"productId":201,"productName":"Cat Toy","price":9.99,"quantity":1,"subtotal":9.99,"createdAt":"2024-03-01T00:00:00Z","updatedAt":"2024-03-01T00:00:00Z"}`

		var item CartItem
		err := json.Unmarshal([]byte(jsonStr), &item)
		assert.NoError(t, err)

		assert.Equal(t, int64(2), item.ID)
		assert.Equal(t, int64(101), item.UserID)
		assert.Equal(t, int64(201), item.ProductID)
		assert.Equal(t, "Cat Toy", item.ProductName)
		assert.Equal(t, 9.99, item.Price)
		assert.Equal(t, 1, item.Quantity)
		assert.Equal(t, 9.99, item.Subtotal)
	})
}

func TestCartStruct(t *testing.T) {
	t.Run("Cart JSON serialization and deserialization", func(t *testing.T) {
		cart := Cart{
			UserID: 100,
			Items: []CartItem{
				{
					ID:          1,
					UserID:      100,
					ProductID:   200,
					ProductName: "Dog Food",
					Price:       29.99,
					Quantity:    2,
					Subtotal:    59.98,
					CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},
			TotalPrice: 59.98,
			TotalItems: 2,
			UpdatedAt:  time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(cart)
		assert.NoError(t, err)

		var decoded Cart
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, cart.UserID, decoded.UserID)
		assert.Len(t, decoded.Items, 1)
		assert.Equal(t, cart.Items[0].ProductID, decoded.Items[0].ProductID)
		assert.Equal(t, cart.TotalPrice, decoded.TotalPrice)
		assert.Equal(t, cart.TotalItems, decoded.TotalItems)
		assert.Equal(t, cart.UpdatedAt, decoded.UpdatedAt)
	})

	t.Run("Cart with empty items", func(t *testing.T) {
		cart := Cart{
			UserID:     100,
			Items:      []CartItem{},
			TotalPrice: 0,
			TotalItems: 0,
			UpdatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(cart)
		assert.NoError(t, err)

		var decoded Cart
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Empty(t, decoded.Items)
		assert.Equal(t, 0.0, decoded.TotalPrice)
		assert.Equal(t, 0, decoded.TotalItems)
	})

	t.Run("Cart JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"userId":102,"items":[{"id":3,"userId":102,"productId":203,"productName":"Fish Tank","price":49.99,"quantity":1,"subtotal":49.99,"createdAt":"2024-05-01T00:00:00Z","updatedAt":"2024-05-01T00:00:00Z"}],"totalPrice":49.99,"totalItems":1,"updatedAt":"2024-05-01T00:00:00Z"}`

		var cart Cart
		err := json.Unmarshal([]byte(jsonStr), &cart)
		assert.NoError(t, err)

		assert.Equal(t, int64(102), cart.UserID)
		assert.Len(t, cart.Items, 1)
		assert.Equal(t, "Fish Tank", cart.Items[0].ProductName)
		assert.Equal(t, 49.99, cart.TotalPrice)
		assert.Equal(t, 1, cart.TotalItems)
	})
}

func TestAddToCartRequestStruct(t *testing.T) {
	t.Run("AddToCartRequest JSON serialization and deserialization", func(t *testing.T) {
		req := AddToCartRequest{
			UserID:      100,
			ProductID:   200,
			ProductName: "Dog Treats",
			Price:       5.99,
			Quantity:    3,
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)

		var decoded AddToCartRequest
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, req.UserID, decoded.UserID)
		assert.Equal(t, req.ProductID, decoded.ProductID)
		assert.Equal(t, req.ProductName, decoded.ProductName)
		assert.Equal(t, req.Price, decoded.Price)
		assert.Equal(t, req.Quantity, decoded.Quantity)
	})

	t.Run("AddToCartRequest JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"userId":103,"productId":204,"productName":"Bird Cage","price":79.99,"quantity":1}`

		var req AddToCartRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		assert.NoError(t, err)

		assert.Equal(t, int64(103), req.UserID)
		assert.Equal(t, int64(204), req.ProductID)
		assert.Equal(t, "Bird Cage", req.ProductName)
		assert.Equal(t, 79.99, req.Price)
		assert.Equal(t, 1, req.Quantity)
	})
}

func TestUpdateCartItemRequestStruct(t *testing.T) {
	t.Run("UpdateCartItemRequest JSON serialization and deserialization", func(t *testing.T) {
		req := UpdateCartItemRequest{
			UserID:   100,
			ID:       1,
			Quantity: 5,
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)

		var decoded UpdateCartItemRequest
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, req.UserID, decoded.UserID)
		assert.Equal(t, req.ID, decoded.ID)
		assert.Equal(t, req.Quantity, decoded.Quantity)
	})

	t.Run("UpdateCartItemRequest JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"userId":104,"id":5,"quantity":10}`

		var req UpdateCartItemRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		assert.NoError(t, err)

		assert.Equal(t, int64(104), req.UserID)
		assert.Equal(t, int64(5), req.ID)
		assert.Equal(t, 10, req.Quantity)
	})
}

func TestDeleteCartItemRequestStruct(t *testing.T) {
	t.Run("DeleteCartItemRequest JSON serialization and deserialization", func(t *testing.T) {
		req := DeleteCartItemRequest{
			UserID: 100,
			IDs:    []int64{1, 2, 3},
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)

		var decoded DeleteCartItemRequest
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, req.UserID, decoded.UserID)
		assert.Equal(t, req.IDs, decoded.IDs)
	})

	t.Run("DeleteCartItemRequest with empty IDs", func(t *testing.T) {
		req := DeleteCartItemRequest{
			UserID: 100,
			IDs:    []int64{},
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)

		var decoded DeleteCartItemRequest
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, req.UserID, decoded.UserID)
		assert.Empty(t, decoded.IDs)
	})

	t.Run("DeleteCartItemRequest JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"userId":105,"ids":[4,5,6]}`

		var req DeleteCartItemRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		assert.NoError(t, err)

		assert.Equal(t, int64(105), req.UserID)
		assert.Equal(t, []int64{4, 5, 6}, req.IDs)
	})
}

func TestCartResponseStruct(t *testing.T) {
	t.Run("CartResponse JSON serialization and deserialization with cart", func(t *testing.T) {
		cart := &Cart{
			UserID:     100,
			Items:      []CartItem{},
			TotalPrice: 0,
			TotalItems: 0,
			UpdatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		resp := CartResponse{
			Success: true,
			Message: "Operation successful",
			Cart:    cart,
		}

		data, err := json.Marshal(resp)
		assert.NoError(t, err)

		var decoded CartResponse
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, resp.Success, decoded.Success)
		assert.Equal(t, resp.Message, decoded.Message)
		assert.NotNil(t, decoded.Cart)
		assert.Equal(t, cart.UserID, decoded.Cart.UserID)
	})

	t.Run("CartResponse JSON serialization and deserialization without cart", func(t *testing.T) {
		resp := CartResponse{
			Success: false,
			Message: "Cart not found",
			Cart:    nil,
		}

		data, err := json.Marshal(resp)
		assert.NoError(t, err)

		var decoded CartResponse
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, resp.Success, decoded.Success)
		assert.Equal(t, resp.Message, decoded.Message)
		assert.Nil(t, decoded.Cart)

		// Verify cart field is omitted when Cart is nil (omitempty behavior)
		var jsonMap map[string]json.RawMessage
		err = json.Unmarshal(data, &jsonMap)
		assert.NoError(t, err)
		_, exists := jsonMap["cart"]
		assert.False(t, exists, "cart should not exist in JSON when Cart is nil")
	})

	t.Run("CartResponse JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"success":true,"message":"Added to cart","cart":{"userId":106,"items":[],"totalPrice":0,"totalItems":0,"updatedAt":"2024-06-01T00:00:00Z"}}`

		var resp CartResponse
		err := json.Unmarshal([]byte(jsonStr), &resp)
		assert.NoError(t, err)

		assert.True(t, resp.Success)
		assert.Equal(t, "Added to cart", resp.Message)
		assert.NotNil(t, resp.Cart)
		assert.Equal(t, int64(106), resp.Cart.UserID)
	})
}
