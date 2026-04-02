package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"petshop/internal/db"
	"petshop/internal/handlers"
	"petshop/internal/logger"
	"petshop/internal/middleware"
)

// serverConfig holds server configuration for testing
type serverConfig struct {
	addr       string
	logDir     string
	dbPath     string
	shutdownTimeout time.Duration
}

// defaultConfig returns default server configuration
func defaultConfig() *serverConfig {
	return &serverConfig{
		addr:       ":8080",
		logDir:     "logs",
		dbPath:     "./cart.db",
		shutdownTimeout: 10 * time.Second,
	}
}

// main is the entry point for the petshop server application.
// It initializes logging, database, middleware, and all API routes.
func main() {
	fmt.Println("Project: petshop")

	if err := run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// run initializes and executes the application.
// It sets up required resources and runs the main event loop.
func run() error {
	config := defaultConfig()
	return runWithConfig(config)
}

// runWithConfig runs the server with the given configuration
func runWithConfig(config *serverConfig) error {
	// Initialize logger
	logger.Init(config.logDir)

	// Initialize database for cart persistence
	if err := db.InitDB(config.dbPath); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
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

	// Setup routes
	setupRoutes(mux)

	// Create HTTP server
	srv := &http.Server{
		Addr:    config.addr,
		Handler: chain,
	}

	// Start server in goroutine with error channel
	errCh := make(chan error, 1)
	go func() {
		log.Printf("Server starting on %s", config.addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		log.Println("Shutting down server...")
	case err := <-errCh:
		return fmt.Errorf("server failed to start: %w", err)
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Stop cache cleanup goroutines
	handlers.GetPetCache().Stop()

	log.Println("Server exited")
	return nil
}

// setupRoutes registers all API routes to the given mux
func setupRoutes(mux *http.ServeMux) {
	// Pet routes
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
		if len(parts) == 5 && parts[4] == "photos" {
			handlers.PetPhotoHandler(w, r)
			return
		}
		if len(parts) != 4 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Extract pet ID from path and set as query param for handlers
		r.URL.RawQuery = "id=" + parts[3]
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

	// Setup admin routes
	setupAdminRoutes(mux)

	// Setup cart routes
	setupCartRoutes(mux)

	// Setup system config routes
	setupConfigRoutes(mux)
}

// setupAdminRoutes registers admin API routes
func setupAdminRoutes(mux *http.ServeMux) {
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
}

// setupCartRoutes registers cart API routes
func setupCartRoutes(mux *http.ServeMux) {
	// Apply auth middleware to protect cart endpoints
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
	mux.Handle("/api/cart", cartHandler)
	mux.Handle("/api/cart/", cartHandler)

	// Clear cart endpoint
	mux.HandleFunc("/api/cart/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			cartHandler.ServeHTTP(w, r)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Allow", "DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// setupConfigRoutes registers system config routes
func setupConfigRoutes(mux *http.ServeMux) {
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
}
