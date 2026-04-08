package handlers

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"petshop/internal/db"
	"petshop/internal/models"
)

// Admin data storage - repository instances
var (
	productRepo      *db.ProductRepository
	orderRepo        *db.OrderRepository
	announcementRepo *db.AnnouncementRepository
	carouselRepo     *db.CarouselRepository
	inventoryRepo    *db.InventoryRepository

	dataMu             sync.RWMutex
	inventoryThreshold = 10
)

// InitRepositories initializes all repositories with the given database
func InitRepositories() {
	productRepo = db.NewProductRepository()
	orderRepo = db.NewOrderRepository()
	announcementRepo = db.NewAnnouncementRepository()
	carouselRepo = db.NewCarouselRepository()
	inventoryRepo = db.NewInventoryRepository()

	// Initialize sample data if database is empty
	initSampleData()
}

// InitRepositoriesWithDB initializes repositories with a specific database instance (for testing)
func InitRepositoriesWithDB(database *sql.DB) {
	productRepo = db.NewProductRepositoryWithDB(database)
	orderRepo = db.NewOrderRepositoryWithDB(database)
	announcementRepo = db.NewAnnouncementRepositoryWithDB(database)
	carouselRepo = db.NewCarouselRepositoryWithDB(database)
	inventoryRepo = db.NewInventoryRepositoryWithDB(database)
}

func initSampleData() {
	// Check if products exist
	products, _ := productRepo.GetAll()
	if len(products) > 0 {
		return // Data already exists
	}

	now := time.Now()

	// Initialize sample products
	if err := productRepo.Create(&models.Product{
		Name:        "狗粮 10kg",
		Description: "优质狗粮",
		Category:    "狗粮",
		Price:       299.00,
		Stock:       50,
		Status:      "on_sale",
		Images:      []string{"/static/images/dog_food.jpg"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		// Log error but continue - sample data init failure is not fatal
		fmt.Printf("Failed to create sample product: %v\n", err)
	}
	if err := productRepo.Create(&models.Product{
		Name:        "猫粮 5kg",
		Description: "天然猫粮",
		Category:    "猫粮",
		Price:       199.00,
		Stock:       8,
		Status:      "on_sale",
		Images:      []string{"/static/images/cat_food.jpg"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		fmt.Printf("Failed to create sample product: %v\n", err)
	}

	// Initialize sample orders
	if err := orderRepo.Create(&models.Order{
		UserID: 1,
		Products: []models.OrderItem{
			{ProductID: 1, ProductName: "狗粮 10kg", Price: 299.00, Quantity: 1, Subtotal: 299.00},
		},
		TotalAmount: 299.00,
		Status:      "paid",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		fmt.Printf("Failed to create sample order: %v\n", err)
	}

	// Initialize sample carousels
	if err := carouselRepo.Create(&models.Carousel{
		ImageURL:  "/static/carousel/banner1.jpg",
		LinkURL:   "/product/1",
		SortOrder: 1,
		Title:     "春季大促",
		Status:    "active",
	}); err != nil {
		fmt.Printf("Failed to create sample carousel: %v\n", err)
	}

	// Initialize sample announcements
	if err := announcementRepo.Create(&models.Announcement{
		Title:     "春节放假通知",
		Content:   "春节期间客服工作时间调整",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		fmt.Printf("Failed to create sample announcement: %v\n", err)
	}
}
