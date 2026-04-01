package models

import "time"

// Product 商品
type Product struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	Status      string    `json:"status"` // on_sale, off_sale, deleted
	Images      []string  `json:"images"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Inventory 库存记录
type Inventory struct {
	ID          int64     `json:"id"`
	ProductID   int64      `json:"productId"`
	ChangeType  string    `json:"changeType"` // in, out, adjust
	Quantity    int       `json:"quantity"`
	BeforeStock int       `json:"beforeStock"`
	AfterStock  int       `json:"afterStock"`
	Reason      string    `json:"reason"`
	Operator    string    `json:"operator"`
	CreatedAt   time.Time `json:"createdAt"`
}

// InventoryAlert 库存预警
type InventoryAlert struct {
	ID          int64   `json:"id"`
	ProductID   int64    `json:"productId"`
	ProductName string   `json:"productName"`
	Threshold   int      `json:"threshold"`
	CurrentStock int     `json:"currentStock"`
	IsRead      bool     `json:"isRead"`
}

// Order 订单
type Order struct {
	ID           int64         `json:"id"`
	UserID       int64         `json:"userId"`
	Products     []OrderItem   `json:"products"`
	TotalAmount  float64       `json:"totalAmount"`
	Status       string        `json:"status"` // pending, paid, shipped, delivered, cancelled, refunding, refunded
	RefundReason string        `json:"refundReason,omitempty"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

// OrderItem 订单项
type OrderItem struct {
	ProductID   int64   `json:"productId"`
	ProductName string  `json:"productName"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	Subtotal    float64 `json:"subtotal"`
}

// User 用户
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Status    string    `json:"status"` // active, disabled
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

// SalesStat 销售统计
type SalesStat struct {
	Date        string  `json:"date"`
	TotalSales  float64 `json:"totalSales"`
	OrderCount  int     `json:"orderCount"`
}

// HotProduct 热销商品
type HotProduct struct {
	ProductID   int64   `json:"productId"`
	ProductName string  `json:"productName"`
	SalesCount  int     `json:"salesCount"`
	SalesAmount float64 `json:"salesAmount"`
}

// Carousel 轮播图
type Carousel struct {
	ID       int64  `json:"id"`
	ImageURL string `json:"imageUrl"`
	LinkURL  string `json:"linkUrl"`
	SortOrder int   `json:"sortOrder"`
	Title    string `json:"title"`
	Status   string `json:"status"` // active, inactive
}

// Announcement 公告
type Announcement struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Status    string    `json:"status"` // active, inactive
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SystemConfig 系统配置
type SystemConfig struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
