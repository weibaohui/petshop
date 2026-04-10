// @title PetShop API
// @version 1.0
// @description PetShop RESTful API 文档 - 宠物商店后端服务
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token authentication, format: Bearer <token>

// @securityDefinitions.apikey ApiTokenAuth
// @in header
// @name X-API-Token
// @description API Token authentication for Open API access

package main

import (
	"context"
	"encoding/json"
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

	httpSwagger "github.com/swaggo/http-swagger"
	_ "petshop/docs"
)

// serverConfig holds server configuration for testing
type serverConfig struct {
	addr            string
	logDir          string
	dbPath          string
	shutdownTimeout time.Duration
}

// defaultConfig returns default server configuration
func defaultConfig() *serverConfig {
	return &serverConfig{
		addr:            ":8080",
		logDir:          "logs",
		dbPath:          "./cart.db",
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
	return runWithDependencies(config, nil)
}

// serverDependencies holds injectable dependencies for testing the server
type serverDependencies struct {
	signalChan         <-chan os.Signal
	serverErrorHandler func(error)
}

// runWithDependencies runs the server with injectable dependencies for testing
// config: server configuration
// deps: optional dependencies for testing (signalChan and serverErrorHandler)
func runWithDependencies(config *serverConfig, deps *serverDependencies) error {
	// Initialize logger
	if err := logger.Init(config.logDir); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize database for cart persistence
	if err := db.InitDB(config.dbPath); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Initialize repositories
	handlers.InitRepositories()

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
			if deps != nil && deps.serverErrorHandler != nil {
				deps.serverErrorHandler(err)
			}
		}
	}()

	// Wait for interrupt signal or server error
	var quit <-chan os.Signal
	if deps != nil && deps.signalChan != nil {
		quit = deps.signalChan
	} else {
		quitCh := make(chan os.Signal, 1)
		signal.Notify(quitCh, syscall.SIGINT, syscall.SIGTERM)
		quit = quitCh
	}

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

// methodNotAllowed writes a 405 Method Not Allowed response with the given allowed methods
func methodNotAllowed(w http.ResponseWriter, allowedMethods string) {
	w.Header().Set("Allow", allowedMethods)
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// methodNotAllowedJSON writes a 405 response with JSON content type
func methodNotAllowedJSON(w http.ResponseWriter, allowedMethods string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Allow", allowedMethods)
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// requireMethod checks if the request method matches, otherwise sends 405 response
// Returns true if method matches, false otherwise
func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		methodNotAllowed(w, method)
		return false
	}
	return true
}

// requireMethodJSON checks if the request method matches with JSON content type
func requireMethodJSON(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		methodNotAllowedJSON(w, method)
		return false
	}
	return true
}

// routeHandler is a function that handles a specific route
type routeHandler func(w http.ResponseWriter, r *http.Request)

// registerRoute registers a simple GET route
func registerRoute(mux *http.ServeMux, path string, handler routeHandler) {
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		handler(w, r)
	})
}

// registerAuthRoute registers an authenticated route with single method
func registerAuthRoute(mux *http.ServeMux, path string, method string, handler routeHandler) {
	mux.Handle(path, middleware.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			methodNotAllowed(w, method)
			return
		}
		handler(w, r)
	})))
}

// registerAuthRoutes registers an authenticated route with multiple methods
func registerAuthRoutes(mux *http.ServeMux, path string, methodHandlers map[string]routeHandler) {
	mux.Handle(path, middleware.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, ok := methodHandlers[r.Method]
		if !ok {
			// Build allowed methods string with deterministic order
			methods := make([]string, 0, len(methodHandlers))
			for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
				if _, exists := methodHandlers[m]; exists {
					methods = append(methods, m)
				}
			}
			methodNotAllowed(w, strings.Join(methods, ", "))
			return
		}
		handler(w, r)
	})))
}

// handleV1PetPath handles /api/v1/pets/:id routes
func handleV1PetPath(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/pets/")
	parts := strings.Split(path, "/")
	if len(parts) == 1 && parts[0] != "" {
		query := r.URL.Query()
		query.Set("id", parts[0])
		r.URL.RawQuery = query.Encode()
		handlers.GetPet(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// handlePetPath handles /api/pet/:id routes with GET/PUT/DELETE
func handlePetPath(w http.ResponseWriter, r *http.Request) {
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
		methodNotAllowed(w, "GET, PUT, DELETE")
	}
}

// handleOpenAPIPetPath handles /api/open/pets/:id routes
func handleOpenAPIPetPath(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/open/pets/")
	parts := strings.Split(path, "/")
	if len(parts) == 1 && parts[0] != "" {
		query := r.URL.Query()
		query.Set("id", parts[0])
		r.URL.RawQuery = query.Encode()
		handlers.GetPet(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// setupRoutes registers all API routes to the given mux
func setupRoutes(mux *http.ServeMux) {
	// Pet routes
	mux.HandleFunc("/api/pets", handlers.ListPets)
	registerRoute(mux, "/api/v1/pets", handlers.FilterPets)
	registerRoute(mux, "/api/v1/categories", handlers.GetCategories)
	// Handle /api/v1/pets/:id
	mux.HandleFunc("/api/v1/pets/", func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		handleV1PetPath(w, r)
	})
	mux.HandleFunc("/api/pet", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetPet(w, r)
		case http.MethodPut:
			handlers.UpdatePet(w, r)
		case http.MethodDelete:
			handlers.DeletePet(w, r)
		default:
			methodNotAllowedJSON(w, "GET, PUT, DELETE")
		}
	})
	mux.HandleFunc("/api/pet/search", handlers.SearchPets)
	mux.HandleFunc("/api/pet/cache/stats", handlers.GetCacheStats)
	mux.HandleFunc("/api/pet/cache/hitrate", handlers.GetCacheHitRate)
	mux.HandleFunc("/api/pet/", handlePetPath)

	// Error page handler for non-API routes
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<!DOCTYPE html>
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

	// Setup API token management routes
	setupAPITokenRoutes(mux)

	// Setup open API routes (public API with token auth)
	setupOpenAPIRoutes(mux)

	// Register health check endpoint
	registerRoute(mux, "/health", handlers.HealthCheck)

	// Register version endpoint
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"version": "1.0.0",
			"name":    "PetShop API",
		})
	})

	// Swagger UI route
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)
}

// setupAdminRoutes registers admin API routes
func setupAdminRoutes(mux *http.ServeMux) {
	// ==================== 商品管理 ====================
	registerAuthRoutes(mux, "/api/admin/products", map[string]routeHandler{
		http.MethodGet:  handlers.ListProducts,
		http.MethodPost: handlers.CreateProduct,
	})
	registerAuthRoutes(mux, "/api/admin/product", map[string]routeHandler{
		http.MethodGet:    handlers.GetProduct,
		http.MethodPut:    handlers.UpdateProduct,
		http.MethodDelete: handlers.DeleteProduct,
	})

	// ==================== 库存管理 ====================
	mux.Handle("/api/admin/inventory/logs", middleware.AuthMiddleware(http.HandlerFunc(handlers.ListInventoryLogs)))
	mux.Handle("/api/admin/inventory/alerts", middleware.AuthMiddleware(http.HandlerFunc(handlers.GetInventoryAlerts)))
	registerAuthRoute(mux, "/api/admin/inventory/adjust", http.MethodPost, handlers.AdjustInventory)

	// ==================== 订单管理 ====================
	registerAuthRoute(mux, "/api/admin/orders", http.MethodGet, handlers.ListOrders)
	registerAuthRoutes(mux, "/api/admin/order", map[string]routeHandler{
		http.MethodGet: handlers.GetOrder,
		http.MethodPut: handlers.UpdateOrderStatus,
	})
	registerAuthRoute(mux, "/api/admin/order/refund", http.MethodPost, handlers.ProcessRefund)

	// ==================== 用户管理 ====================
	registerAuthRoute(mux, "/api/admin/users", http.MethodGet, handlers.ListUsers)
	registerAuthRoutes(mux, "/api/admin/user", map[string]routeHandler{
		http.MethodGet: handlers.GetUser,
		http.MethodPut: handlers.UpdateUserStatus,
	})
	registerAuthRoute(mux, "/api/admin/user/reset-password", http.MethodPost, handlers.ResetUserPassword)

	// ==================== 销售统计 ====================
	mux.Handle("/api/admin/stats/sales", middleware.AuthMiddleware(http.HandlerFunc(handlers.GetSalesStats)))
	mux.Handle("/api/admin/stats/hot-products", middleware.AuthMiddleware(http.HandlerFunc(handlers.GetHotProducts)))
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
			methodNotAllowedJSON(w, "GET, POST, PUT, DELETE")
		}
	}))
	mux.Handle("/api/cart", cartHandler)
	mux.Handle("/api/cart/", cartHandler)

	// Clear cart endpoint
	mux.HandleFunc("/api/cart/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			cartHandler.ServeHTTP(w, r)
		} else {
			methodNotAllowedJSON(w, "DELETE")
		}
	})
}

// setupAPITokenRoutes registers API token management routes (admin only)
func setupAPITokenRoutes(mux *http.ServeMux) {
	// Token management - requires JWT auth (admin)
	registerAuthRoutes(mux, "/api/admin/tokens", map[string]routeHandler{
		http.MethodGet:  handlers.ListAPITokens,
		http.MethodPost: handlers.CreateAPIToken,
	})
	registerAuthRoutes(mux, "/api/admin/token", map[string]routeHandler{
		http.MethodPut:    handlers.UpdateAPITokenStatus,
		http.MethodDelete: handlers.DeleteAPIToken,
	})
}

// setupOpenAPIRoutes registers public API routes with API token authentication
func setupOpenAPIRoutes(mux *http.ServeMux) {
	// Open API endpoints - requires API token auth
	openAPIHandler := middleware.APITokenAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return basic info about the API
		response := map[string]interface{}{
			"message": "Welcome to PetShop Open API",
			"version": "v1",
			"endpoints": []string{
				"GET /api/open/pets - List all pets",
				"GET /api/open/pets/{id} - Get pet by ID",
				"GET /api/open/categories - Get all categories",
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	mux.Handle("/api/open", openAPIHandler)

	// List pets - open API
	mux.Handle("/api/open/pets", middleware.APITokenAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		handlers.ListPets(w, r)
	})))

	// Get pet by ID - open API
	mux.Handle("/api/open/pets/", middleware.APITokenAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		handleOpenAPIPetPath(w, r)
	})))

	// Get categories - open API
	mux.Handle("/api/open/categories", middleware.APITokenAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		handlers.GetCategories(w, r)
	})))
}

// setupConfigRoutes registers system config routes
func setupConfigRoutes(mux *http.ServeMux) {
	// 轮播图管理
	registerAuthRoutes(mux, "/api/admin/carousels", map[string]routeHandler{
		http.MethodGet:  handlers.ListCarousels,
		http.MethodPost: handlers.CreateCarousel,
	})
	registerAuthRoutes(mux, "/api/admin/carousel", map[string]routeHandler{
		http.MethodPut:    handlers.UpdateCarousel,
		http.MethodDelete: handlers.DeleteCarousel,
	})

	// 公告管理
	registerAuthRoutes(mux, "/api/admin/announcements", map[string]routeHandler{
		http.MethodGet:  handlers.ListAnnouncements,
		http.MethodPost: handlers.CreateAnnouncement,
	})
	registerAuthRoutes(mux, "/api/admin/announcement", map[string]routeHandler{
		http.MethodPut:    handlers.UpdateAnnouncement,
		http.MethodDelete: handlers.DeleteAnnouncement,
	})

	// 系统参数配置
	registerAuthRoute(mux, "/api/admin/configs", http.MethodGet, handlers.GetSystemConfigs)
	registerAuthRoute(mux, "/api/admin/config", http.MethodPost, handlers.SetSystemConfig)
}
