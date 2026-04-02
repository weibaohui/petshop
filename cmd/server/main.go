package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"petshop/internal/db"
	"petshop/internal/handlers"
	"petshop/internal/logger"
	"petshop/internal/middleware"
)

// main is the entry point for the petshop server application.
// It initializes logging, database, middleware, and all API routes.
func main() {
	fmt.Println("Project: petshop")

	// Initialize logger
	logger.Init("logs")

	// Initialize database for cart persistence
	if err := db.InitDB("./cart.db"); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create rate limiter (100 requests per minute)
	rateLimiter := middleware.NewRateLimiter(100, time.Minute)

	// Create mux and apply global middleware
	mux := http.NewServeMux()

	// Apply middleware chain
	chain := middleware.RecoveryMiddleware(
		middleware.RequestLoggerMiddleware(
			middleware.SecurityHeaders(
				middleware.RateLimitMiddleware(rateLimiter)(
					middleware.XSSProtection(
						middleware.InputSanitizer(mux))))))

	// Register routes
	mux.HandleFunc("/api/pets", handlers.ListPets)
	mux.HandleFunc("/api/pet", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetPet(w, r)
		case http.MethodPut:
			handlers.UpdatePet(w, r)
		case http.MethodDelete:
			handlers.DeletePet(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Allow", "GET, PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/pet/search", handlers.SearchPets)
	mux.HandleFunc("/api/pet/cache/stats", handlers.GetCacheStats)
	mux.HandleFunc("/api/pet/cache/hitrate", handlers.GetCacheHitRate)
	mux.HandleFunc("/api/pet/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
		// parts should be like ["", "api", "pet", "1"] or ["", "api", "pet", "1", "photos"]
		if len(parts) >= 5 && parts[4] == "photos" {
			handlers.PetPhotoHandler(w, r)
			return
		}
		if len(parts) != 4 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			handlers.GetPet(w, r)
		case http.MethodPut:
			handlers.UpdatePet(w, r)
		case http.MethodDelete:
			handlers.DeletePet(w, r)
		default:
			w.Header().Set("Allow", "GET, PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Error page handler for non-API routes
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Error</title></head>
<body>
<h1>Something went wrong</h1>
<p>We apologize for the inconvenience. Please try again later.</p>
</body>
</html>`))
	})

	// ==================== 商品管理 ====================
	mux.HandleFunc("/api/admin/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListProducts(w, r)
		case http.MethodPost:
			handlers.CreateProduct(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/admin/product", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetProduct(w, r)
		case http.MethodPut:
			handlers.UpdateProduct(w, r)
		case http.MethodDelete:
			handlers.DeleteProduct(w, r)
		default:
			w.Header().Set("Allow", "GET, PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// ==================== 库存管理 ====================
	mux.HandleFunc("/api/admin/inventory/logs", handlers.ListInventoryLogs)
	mux.HandleFunc("/api/admin/inventory/alerts", handlers.GetInventoryAlerts)
	mux.HandleFunc("/api/admin/inventory/adjust", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.AdjustInventory(w, r)
		} else {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// ==================== 订单管理 ====================
	mux.HandleFunc("/api/admin/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListOrders(w, r)
		} else {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/admin/order", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetOrder(w, r)
		case http.MethodPut:
			handlers.UpdateOrderStatus(w, r)
		default:
			w.Header().Set("Allow", "GET, PUT")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/admin/order/refund", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.ProcessRefund(w, r)
		} else {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// ==================== 用户管理 ====================
	mux.HandleFunc("/api/admin/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListUsers(w, r)
		} else {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/admin/user", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetUser(w, r)
		case http.MethodPut:
			handlers.UpdateUserStatus(w, r)
		default:
			w.Header().Set("Allow", "GET, PUT")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/admin/user/reset-password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.ResetUserPassword(w, r)
		} else {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// ==================== 销售统计 ====================
	mux.HandleFunc("/api/admin/stats/sales", handlers.GetSalesStats)
	mux.HandleFunc("/api/admin/stats/hot-products", handlers.GetHotProducts)

	// ==================== 购物车管理 ====================
	// Apply auth middleware to protect cart endpoints (issue #1)
	cartHandler := middleware.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetCart(w, r)
		case http.MethodPost:
			handlers.AddToCart(w, r)
		case http.MethodPut:
			handlers.UpdateCartItem(w, r)
		case http.MethodDelete:
			handlers.DeleteCartItem(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Allow", "GET, POST, PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	http.Handle("/api/cart", cartHandler)
	http.Handle("/api/cart/", cartHandler)

	// Clear cart endpoint
	http.HandleFunc("/api/cart/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			cartHandler.ServeHTTP(w, r)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Allow", "DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// ==================== 系统配置 ====================
	// 轮播图管理
	mux.HandleFunc("/api/admin/carousels", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListCarousels(w, r)
		case http.MethodPost:
			handlers.CreateCarousel(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/admin/carousel", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateCarousel(w, r)
		case http.MethodDelete:
			handlers.DeleteCarousel(w, r)
		default:
			w.Header().Set("Allow", "PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 公告管理
	mux.HandleFunc("/api/admin/announcements", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListAnnouncements(w, r)
		case http.MethodPost:
			handlers.CreateAnnouncement(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/admin/announcement", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateAnnouncement(w, r)
		case http.MethodDelete:
			handlers.DeleteAnnouncement(w, r)
		default:
			w.Header().Set("Allow", "PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 系统参数配置
	mux.HandleFunc("/api/admin/configs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetSystemConfigs(w, r)
		} else {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/admin/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.SetSystemConfig(w, r)
		} else {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", chain))
}

// run initializes and executes the application.
// It sets up required resources and runs the main event loop.
func run() error {
	// Application initialization
	return nil
}
