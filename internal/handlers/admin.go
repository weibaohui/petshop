package handlers

import (
	"sync"
	"time"

	"petshop/internal/models"
)

// Admin data storage
var (
	products      = make(map[int64]*models.Product)
	orders        = make(map[int64]*models.Order)
	users         = make(map[int64]*models.User)
	inventoryLogs = make([]models.Inventory, 0)
	carousels     = make(map[int64]*models.Carousel)
	announcements = make(map[int64]*models.Announcement)
	systemConfigs = make(map[string]string)

	dataMu sync.RWMutex
	nextProductID      int64 = 1
	nextOrderID        int64 = 1
	nextUserID         int64 = 1
	nextInventoryID    int64 = 1
	nextCarouselID     int64 = 1
	nextAnnouncementID int64 = 1

	// 库存预警阈值
	inventoryThreshold = 10
)

func init() {
	// 初始化示例数据
	products[1] = &models.Product{
		ID:          1,
		Name:        "狗粮 10kg",
		Description: "优质狗粮",
		Category:    "狗粮",
		Price:       299.00,
		Stock:       50,
		Status:      "on_sale",
		Images:      []string{"/static/images/dog_food.jpg"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	products[2] = &models.Product{
		ID:          2,
		Name:        "猫粮 5kg",
		Description: "天然猫粮",
		Category:    "猫粮",
		Price:       199.00,
		Stock:       8,
		Status:      "on_sale",
		Images:      []string{"/static/images/cat_food.jpg"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	nextProductID = 3

	users[1] = &models.User{
		ID:        1,
		Username:  "user1",
		Email:     "user1@example.com",
		Phone:     "13800138000",
		Status:    "active",
		Role:      "user",
		CreatedAt: time.Now(),
	}
	users[2] = &models.User{
		ID:        2,
		Username:  "user2",
		Email:     "user2@example.com",
		Phone:     "13800138001",
		Status:    "active",
		Role:      "user",
		CreatedAt: time.Now(),
	}
	nextUserID = 3

	orders[1] = &models.Order{
		ID:      1,
		UserID:  1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "paid",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	nextOrderID = 2

	carousels[1] = &models.Carousel{
		ID:        1,
		ImageURL:  "/static/carousel/banner1.jpg",
		LinkURL:   "/product/1",
		SortOrder: 1,
		Title:     "春季大促",
		Status:    "active",
	}
	nextCarouselID = 2

	announcements[1] = &models.Announcement{
		ID:        1,
		Title:     "春节放假通知",
		Content:   "春节期间客服工作时间调整",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	nextAnnouncementID = 2

	systemConfigs["site_name"] = "宠物商店"
	systemConfigs["inventory_threshold"] = "10"
}
