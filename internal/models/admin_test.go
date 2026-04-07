package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProductStruct(t *testing.T) {
	t.Run("Product JSON serialization and deserialization", func(t *testing.T) {
		product := Product{
			ID:          1,
			Name:        "Premium Dog Food",
			Description: "High quality nutrition for dogs",
			Category:    "Food",
			Price:       29.99,
			Stock:       100,
			Status:      "on_sale",
			Images:      []string{"url1", "url2"},
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(product)
		assert.NoError(t, err)

		var decoded Product
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, product.ID, decoded.ID)
		assert.Equal(t, product.Name, decoded.Name)
		assert.Equal(t, product.Description, decoded.Description)
		assert.Equal(t, product.Category, decoded.Category)
		assert.Equal(t, product.Price, decoded.Price)
		assert.Equal(t, product.Stock, decoded.Stock)
		assert.Equal(t, product.Status, decoded.Status)
		assert.Equal(t, product.Images, decoded.Images)
		assert.Equal(t, product.CreatedAt, decoded.CreatedAt)
		assert.Equal(t, product.UpdatedAt, decoded.UpdatedAt)
	})

	t.Run("Product with empty images", func(t *testing.T) {
		product := Product{
			ID:        2,
			Name:      "Cat Toy",
			Category:  "Toys",
			Price:     9.99,
			Stock:     50,
			Status:    "on_sale",
			Images:    []string{},
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(product)
		assert.NoError(t, err)

		var decoded Product
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Empty(t, decoded.Images)
		assert.Equal(t, product.Name, decoded.Name)
	})

	t.Run("Product JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"id":3,"name":"Bird Cage","description":"Spacious cage for birds","category":"Housing","price":79.99,"stock":20,"status":"on_sale","images":["url3"],"createdAt":"2024-03-01T00:00:00Z","updatedAt":"2024-03-01T00:00:00Z"}`

		var product Product
		err := json.Unmarshal([]byte(jsonStr), &product)
		assert.NoError(t, err)

		assert.Equal(t, int64(3), product.ID)
		assert.Equal(t, "Bird Cage", product.Name)
		assert.Equal(t, "Housing", product.Category)
		assert.Equal(t, 79.99, product.Price)
		assert.Equal(t, 20, product.Stock)
		assert.Equal(t, "on_sale", product.Status)
	})
}

func TestInventoryStruct(t *testing.T) {
	t.Run("Inventory JSON serialization and deserialization", func(t *testing.T) {
		inv := Inventory{
			ID:          1,
			ProductID:   100,
			ChangeType:  "in",
			Quantity:    50,
			BeforeStock: 100,
			AfterStock:  150,
			Reason:      "Restock",
			Operator:    "admin",
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(inv)
		assert.NoError(t, err)

		var decoded Inventory
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, inv.ID, decoded.ID)
		assert.Equal(t, inv.ProductID, decoded.ProductID)
		assert.Equal(t, inv.ChangeType, decoded.ChangeType)
		assert.Equal(t, inv.Quantity, decoded.Quantity)
		assert.Equal(t, inv.BeforeStock, decoded.BeforeStock)
		assert.Equal(t, inv.AfterStock, decoded.AfterStock)
		assert.Equal(t, inv.Reason, decoded.Reason)
		assert.Equal(t, inv.Operator, decoded.Operator)
		assert.Equal(t, inv.CreatedAt, decoded.CreatedAt)
	})

	t.Run("Inventory JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"id":2,"productId":101,"changeType":"out","quantity":10,"beforeStock":50,"afterStock":40,"reason":"Sale","operator":"system","createdAt":"2024-04-01T00:00:00Z"}`

		var inv Inventory
		err := json.Unmarshal([]byte(jsonStr), &inv)
		assert.NoError(t, err)

		assert.Equal(t, int64(2), inv.ID)
		assert.Equal(t, int64(101), inv.ProductID)
		assert.Equal(t, "out", inv.ChangeType)
		assert.Equal(t, 10, inv.Quantity)
		assert.Equal(t, 50, inv.BeforeStock)
		assert.Equal(t, 40, inv.AfterStock)
	})
}

func TestInventoryAlertStruct(t *testing.T) {
	t.Run("InventoryAlert JSON serialization and deserialization", func(t *testing.T) {
		alert := InventoryAlert{
			ID:           1,
			ProductID:    100,
			ProductName:  "Dog Food",
			Threshold:    20,
			CurrentStock: 15,
			IsRead:       false,
		}

		data, err := json.Marshal(alert)
		assert.NoError(t, err)

		var decoded InventoryAlert
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, alert.ID, decoded.ID)
		assert.Equal(t, alert.ProductID, decoded.ProductID)
		assert.Equal(t, alert.ProductName, decoded.ProductName)
		assert.Equal(t, alert.Threshold, decoded.Threshold)
		assert.Equal(t, alert.CurrentStock, decoded.CurrentStock)
		assert.Equal(t, alert.IsRead, decoded.IsRead)
	})

	t.Run("InventoryAlert JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"id":2,"productId":102,"productName":"Cat Litter","threshold":10,"currentStock":5,"isRead":true}`

		var alert InventoryAlert
		err := json.Unmarshal([]byte(jsonStr), &alert)
		assert.NoError(t, err)

		assert.Equal(t, int64(2), alert.ID)
		assert.Equal(t, int64(102), alert.ProductID)
		assert.Equal(t, "Cat Litter", alert.ProductName)
		assert.Equal(t, 10, alert.Threshold)
		assert.Equal(t, 5, alert.CurrentStock)
		assert.True(t, alert.IsRead)
	})
}

func TestOrderAndOrderItemStruct(t *testing.T) {
	t.Run("Order JSON serialization and deserialization", func(t *testing.T) {
		order := Order{
			ID:     1,
			UserID: 100,
			Products: []OrderItem{
				{
					ProductID:   200,
					ProductName: "Dog Food",
					Price:       29.99,
					Quantity:    2,
					Subtotal:    59.98,
				},
			},
			TotalAmount:  59.98,
			Status:       "pending",
			RefundReason: "",
			CreatedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(order)
		assert.NoError(t, err)

		var decoded Order
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, order.ID, decoded.ID)
		assert.Equal(t, order.UserID, decoded.UserID)
		assert.Len(t, decoded.Products, 1)
		assert.Equal(t, order.Products[0].ProductID, decoded.Products[0].ProductID)
		assert.Equal(t, order.TotalAmount, decoded.TotalAmount)
		assert.Equal(t, order.Status, decoded.Status)
		assert.Equal(t, order.CreatedAt, decoded.CreatedAt)
		assert.Equal(t, order.UpdatedAt, decoded.UpdatedAt)

		// Verify refundReason field is omitted when empty (omitempty behavior)
		var jsonMap map[string]json.RawMessage
		err = json.Unmarshal(data, &jsonMap)
		assert.NoError(t, err)
		_, exists := jsonMap["refundReason"]
		assert.False(t, exists, "refundReason should not exist in JSON when RefundReason is empty")
	})

	t.Run("Order with refund reason", func(t *testing.T) {
		order := Order{
			ID:           2,
			UserID:       101,
			Products:     []OrderItem{},
			TotalAmount:  0,
			Status:       "refunded",
			RefundReason: "Customer request",
			CreatedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(order)
		assert.NoError(t, err)

		var decoded Order
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, "refunded", decoded.Status)
		assert.Equal(t, "Customer request", decoded.RefundReason)

		// Verify refundReason field exists and has correct value when non-empty (omitempty behavior)
		var jsonMap map[string]json.RawMessage
		err = json.Unmarshal(data, &jsonMap)
		assert.NoError(t, err)
		refundReasonRaw, exists := jsonMap["refundReason"]
		assert.True(t, exists, "refundReason should exist in JSON when RefundReason is non-empty")
		var refundReasonValue string
		err = json.Unmarshal(refundReasonRaw, &refundReasonValue)
		assert.NoError(t, err)
		assert.Equal(t, "Customer request", refundReasonValue)
	})

	t.Run("Order JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"id":3,"userId":103,"products":[{"productId":201,"productName":"Cat Toy","price":9.99,"quantity":1,"subtotal":9.99}],"totalAmount":9.99,"status":"paid","createdAt":"2024-05-01T00:00:00Z","updatedAt":"2024-05-01T00:00:00Z"}`

		var order Order
		err := json.Unmarshal([]byte(jsonStr), &order)
		assert.NoError(t, err)

		assert.Equal(t, int64(3), order.ID)
		assert.Equal(t, int64(103), order.UserID)
		assert.Equal(t, 9.99, order.TotalAmount)
		assert.Equal(t, "paid", order.Status)
	})
}

func TestUserStruct(t *testing.T) {
	t.Run("User JSON serialization and deserialization", func(t *testing.T) {
		user := User{
			ID:        1,
			Username:  "admin",
			Email:     "admin@example.com",
			Phone:     "13800138000",
			Status:    "active",
			Role:      "admin",
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(user)
		assert.NoError(t, err)

		var decoded User
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, user.ID, decoded.ID)
		assert.Equal(t, user.Username, decoded.Username)
		assert.Equal(t, user.Email, decoded.Email)
		assert.Equal(t, user.Phone, decoded.Phone)
		assert.Equal(t, user.Status, decoded.Status)
		assert.Equal(t, user.Role, decoded.Role)
		assert.Equal(t, user.CreatedAt, decoded.CreatedAt)
	})

	t.Run("User JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"id":2,"username":"john_doe","email":"john@example.com","phone":"13900139000","status":"disabled","role":"user","createdAt":"2024-02-01T00:00:00Z"}`

		var user User
		err := json.Unmarshal([]byte(jsonStr), &user)
		assert.NoError(t, err)

		assert.Equal(t, int64(2), user.ID)
		assert.Equal(t, "john_doe", user.Username)
		assert.Equal(t, "disabled", user.Status)
		assert.Equal(t, "user", user.Role)
	})
}

func TestSalesStatStruct(t *testing.T) {
	t.Run("SalesStat JSON serialization and deserialization", func(t *testing.T) {
		stat := SalesStat{
			Date:       "2024-01-01",
			TotalSales: 999.99,
			OrderCount: 10,
		}

		data, err := json.Marshal(stat)
		assert.NoError(t, err)

		var decoded SalesStat
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, stat.Date, decoded.Date)
		assert.Equal(t, stat.TotalSales, decoded.TotalSales)
		assert.Equal(t, stat.OrderCount, decoded.OrderCount)
	})

	t.Run("SalesStat JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"date":"2024-02-01","totalSales":500.5,"orderCount":5}`

		var stat SalesStat
		err := json.Unmarshal([]byte(jsonStr), &stat)
		assert.NoError(t, err)

		assert.Equal(t, "2024-02-01", stat.Date)
		assert.Equal(t, 500.5, stat.TotalSales)
		assert.Equal(t, 5, stat.OrderCount)
	})
}

func TestHotProductStruct(t *testing.T) {
	t.Run("HotProduct JSON serialization and deserialization", func(t *testing.T) {
		hp := HotProduct{
			ProductID:   1,
			ProductName: "Dog Food",
			SalesCount:  100,
			SalesAmount: 2999.99,
		}

		data, err := json.Marshal(hp)
		assert.NoError(t, err)

		var decoded HotProduct
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, hp.ProductID, decoded.ProductID)
		assert.Equal(t, hp.ProductName, decoded.ProductName)
		assert.Equal(t, hp.SalesCount, decoded.SalesCount)
		assert.Equal(t, hp.SalesAmount, decoded.SalesAmount)
	})

	t.Run("HotProduct JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"productId":2,"productName":"Cat Food","salesCount":50,"salesAmount":499.5}`

		var hp HotProduct
		err := json.Unmarshal([]byte(jsonStr), &hp)
		assert.NoError(t, err)

		assert.Equal(t, int64(2), hp.ProductID)
		assert.Equal(t, "Cat Food", hp.ProductName)
		assert.Equal(t, 50, hp.SalesCount)
		assert.Equal(t, 499.5, hp.SalesAmount)
	})
}

func TestCarouselStruct(t *testing.T) {
	t.Run("Carousel JSON serialization and deserialization", func(t *testing.T) {
		carousel := Carousel{
			ID:        1,
			ImageURL:  "https://example.com/image.jpg",
			LinkURL:   "https://example.com/link",
			SortOrder: 1,
			Title:     "New Arrivals",
			Status:    "active",
		}

		data, err := json.Marshal(carousel)
		assert.NoError(t, err)

		var decoded Carousel
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, carousel.ID, decoded.ID)
		assert.Equal(t, carousel.ImageURL, decoded.ImageURL)
		assert.Equal(t, carousel.LinkURL, decoded.LinkURL)
		assert.Equal(t, carousel.SortOrder, decoded.SortOrder)
		assert.Equal(t, carousel.Title, decoded.Title)
		assert.Equal(t, carousel.Status, decoded.Status)
	})

	t.Run("Carousel JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"id":2,"imageUrl":"https://example.com/banner.jpg","linkUrl":"https://example.com/sale","sortOrder":2,"title":"Summer Sale","status":"inactive"}`

		var carousel Carousel
		err := json.Unmarshal([]byte(jsonStr), &carousel)
		assert.NoError(t, err)

		assert.Equal(t, int64(2), carousel.ID)
		assert.Equal(t, "Summer Sale", carousel.Title)
		assert.Equal(t, "inactive", carousel.Status)
	})
}

func TestAnnouncementStruct(t *testing.T) {
	t.Run("Announcement JSON serialization and deserialization", func(t *testing.T) {
		announcement := Announcement{
			ID:        1,
			Title:     "Holiday Notice",
			Content:   "We will be closed on New Year's Day.",
			Status:    "active",
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(announcement)
		assert.NoError(t, err)

		var decoded Announcement
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, announcement.ID, decoded.ID)
		assert.Equal(t, announcement.Title, decoded.Title)
		assert.Equal(t, announcement.Content, decoded.Content)
		assert.Equal(t, announcement.Status, decoded.Status)
		assert.Equal(t, announcement.CreatedAt, decoded.CreatedAt)
		assert.Equal(t, announcement.UpdatedAt, decoded.UpdatedAt)
	})

	t.Run("Announcement JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"id":2,"title":"System Maintenance","content":"Scheduled maintenance at midnight.","status":"active","createdAt":"2024-03-01T00:00:00Z","updatedAt":"2024-03-01T00:00:00Z"}`

		var announcement Announcement
		err := json.Unmarshal([]byte(jsonStr), &announcement)
		assert.NoError(t, err)

		assert.Equal(t, int64(2), announcement.ID)
		assert.Equal(t, "System Maintenance", announcement.Title)
		assert.Equal(t, "Scheduled maintenance at midnight.", announcement.Content)
	})
}

func TestSystemConfigStruct(t *testing.T) {
	t.Run("SystemConfig JSON serialization and deserialization", func(t *testing.T) {
		config := SystemConfig{
			Key:   "site_name",
			Value: "PetShop",
		}

		data, err := json.Marshal(config)
		assert.NoError(t, err)

		var decoded SystemConfig
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, config.Key, decoded.Key)
		assert.Equal(t, config.Value, decoded.Value)
	})

	t.Run("SystemConfig JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"key":"contact_email","value":"support@petshop.com"}`

		var config SystemConfig
		err := json.Unmarshal([]byte(jsonStr), &config)
		assert.NoError(t, err)

		assert.Equal(t, "contact_email", config.Key)
		assert.Equal(t, "support@petshop.com", config.Value)
	})
}
