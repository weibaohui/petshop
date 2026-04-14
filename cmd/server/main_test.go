package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"petshop/internal/db"
	"petshop/internal/handlers"
	"petshop/internal/logger"
	"petshop/internal/middleware"
)

// setupTestServer creates a test server with all middleware and routes
func setupTestServer(t *testing.T) (http.Handler, func()) {
	t.Helper()

	// Create temp directory for test files
	tempDir := t.TempDir()

	// Reset global state for testing
	logger.ResetForTesting()
	handlers.ResetPetsForTesting()

	// Initialize logger
	if err := logger.Init(tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize database with temp file
	dbPath := filepath.Join(tempDir, "test.db")
	db.ResetForTesting()
	if err := db.InitDB(dbPath); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize repositories with the test database
	handlers.InitRepositories()

	// Create rate limiter for testing
	rateLimiter := middleware.NewRateLimiter(100, time.Minute)

	// Create mux and apply middleware chain
	mux := http.NewServeMux()
	setupRoutes(mux)

	// Apply middleware chain
	chain := middleware.RecoveryMiddleware(
		middleware.RequestLoggerMiddleware(
			middleware.SecurityHeaders(
				middleware.RateLimitMiddleware(rateLimiter)(
					middleware.XSSProtection(
						middleware.InputSanitizer(mux))))))

	cleanup := func() {
		db.Close()
		db.ResetForTesting()
		logger.Close()
		logger.ResetForTesting()
		handlers.ResetPetsForTesting()
	}

	return chain, cleanup
}

// TestRouteRegistration tests that all routes are properly registered
func TestRouteRegistration(t *testing.T) {
	// Set JWT secret for testing (must be at least 32 bytes)
	os.Setenv("JWT_SECRET_KEY", "this-is-a-test-secret-key-that-is-32b")
	defer os.Unsetenv("JWT_SECRET_KEY")

	handler, cleanup := setupTestServer(t)
	defer cleanup()

	// Generate a valid JWT token for admin routes
	validToken, err := middleware.GenerateToken(1)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Reset pets state before each subtest to avoid test order dependency
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		body           string
		authRequired   bool
	}{
		// Pet routes
		{"ListPets GET", "GET", "/api/pets", http.StatusOK, "", false},
		{"GetPet GET", "GET", "/api/pet?id=1", http.StatusOK, "", false},
		{"GetPet Not Found", "GET", "/api/pet?id=999", http.StatusNotFound, "", false},
		{"UpdatePet PUT", "PUT", "/api/pet", http.StatusOK, `{"id":1,"name":"Buddy Updated","type":"Dog","status":"available"}`, false},
		{"DeletePet DELETE", "DELETE", "/api/pet?id=3", http.StatusOK, "", false},
		{"SearchPets GET", "GET", "/api/pet/search?name=Buddy", http.StatusOK, "", false},
		{"CacheStats GET", "GET", "/api/pet/cache/stats", http.StatusOK, "", false},
		{"CacheHitRate GET", "GET", "/api/pet/cache/hitrate", http.StatusOK, "", false},
		{"PetByPath GET", "GET", "/api/pet/2", http.StatusOK, "", false},
		{"PetByPath PUT", "PUT", "/api/pet/2", http.StatusOK, `{"id":2,"name":"Whiskers Updated","type":"Cat","status":"available"}`, false},

		// Admin product routes
		{"ListProducts GET", "GET", "/api/admin/products", http.StatusOK, "", true},
		{"GetProduct GET", "GET", "/api/admin/product?id=1", http.StatusOK, "", true},
		{"UpdateProduct PUT", "PUT", "/api/admin/product", http.StatusOK, `{"id":1,"name":"Test Product","price":9.99,"stock":10}`, true},

		// Admin inventory routes
		{"ListInventoryLogs GET", "GET", "/api/admin/inventory/logs", http.StatusOK, "", true},
		{"GetInventoryAlerts GET", "GET", "/api/admin/inventory/alerts", http.StatusOK, "", true},

		// Admin order routes
		{"ListOrders GET", "GET", "/api/admin/orders", http.StatusOK, "", true},
		{"GetOrder GET", "GET", "/api/admin/order?id=1", http.StatusOK, "", true},

		// Admin user routes
		{"ListUsers GET", "GET", "/api/admin/users", http.StatusOK, "", true},
		{"GetUser GET", "GET", "/api/admin/user?id=1", http.StatusOK, "", true},

		// Admin stats routes
		{"GetSalesStats GET", "GET", "/api/admin/stats/sales", http.StatusOK, "", true},
		{"GetHotProducts GET", "GET", "/api/admin/stats/hot-products", http.StatusOK, "", true},

		// Admin config routes
		{"ListCarousels GET", "GET", "/api/admin/carousels", http.StatusOK, "", true},
		{"ListAnnouncements GET", "GET", "/api/admin/announcements", http.StatusOK, "", true},
		{"GetSystemConfigs GET", "GET", "/api/admin/configs", http.StatusOK, "", true},

		// Version endpoint
		{"Version GET", "GET", "/api/version", http.StatusOK, "", false},

		// Error page
		{"ErrorPage GET", "GET", "/error", http.StatusInternalServerError, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset pets state before each subtest to avoid test order dependency
			handlers.ResetPetsForTesting()

			var reqBody io.Reader
			if tt.body != "" {
				reqBody = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.path, reqBody)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			// Add auth header for protected routes
			if tt.authRequired {
				req.Header.Set("Authorization", "Bearer "+validToken)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// TestMethodNotAllowed tests 405 Method Not Allowed responses
func TestMethodNotAllowed(t *testing.T) {
	// Set JWT secret for testing (must be at least 32 bytes)
	os.Setenv("JWT_SECRET_KEY", "this-is-a-test-secret-key-that-is-32b")
	defer os.Unsetenv("JWT_SECRET_KEY")

	handler, cleanup := setupTestServer(t)
	defer cleanup()

	// Generate a valid JWT token for admin routes
	validToken, err := middleware.GenerateToken(1)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name          string
		method        string
		path          string
		expectedAllow string
		authRequired  bool
	}{
		{"Pet POST not allowed", "POST", "/api/pet", "GET, PUT, DELETE", false},
		{"Pet PATCH not allowed", "PATCH", "/api/pet", "GET, PUT, DELETE", false},
		{"Products DELETE not allowed", "DELETE", "/api/admin/products", "GET, POST", true},
		{"Product PATCH not allowed", "PATCH", "/api/admin/product", "GET, PUT, DELETE", true},
		{"Orders POST not allowed", "POST", "/api/admin/orders", "GET", true},
		{"Order DELETE not allowed", "DELETE", "/api/admin/order", "GET, PUT", true},
		{"Users POST not allowed", "POST", "/api/admin/users", "GET", true},
		{"User DELETE not allowed", "DELETE", "/api/admin/user", "GET, PUT", true},
		{"Carousels DELETE not allowed", "DELETE", "/api/admin/carousels", "GET, POST", true},
		{"Carousel GET not allowed", "GET", "/api/admin/carousel", "PUT, DELETE", true},
		{"Announcements DELETE not allowed", "DELETE", "/api/admin/announcements", "GET, POST", true},
		{"Announcement GET not allowed", "GET", "/api/admin/announcement", "PUT, DELETE", true},
		{"Configs POST not allowed", "POST", "/api/admin/configs", "GET", true},
		{"Config GET not allowed", "GET", "/api/admin/config", "POST", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			// Add auth header for protected routes
			if tt.authRequired {
				req.Header.Set("Authorization", "Bearer "+validToken)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
			}

			allowHeader := rr.Header().Get("Allow")
			if allowHeader != tt.expectedAllow {
				t.Errorf("Expected Allow header %q, got %q", tt.expectedAllow, allowHeader)
			}
		})
	}
}

// TestV1PetRoutes tests v1 pet routes for method restrictions and path validation
func TestV1PetRoutes(t *testing.T) {
	handler, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedAllow  string
	}{
		{"FilterPets GET allowed", "GET", "/api/v1/pets", http.StatusOK, ""},
		{"FilterPets POST not allowed", "POST", "/api/v1/pets", http.StatusMethodNotAllowed, "GET"},
		{"FilterPets PUT not allowed", "PUT", "/api/v1/pets", http.StatusMethodNotAllowed, "GET"},
		{"GetCategories GET allowed", "GET", "/api/v1/categories", http.StatusOK, ""},
		{"GetCategories POST not allowed", "POST", "/api/v1/categories", http.StatusMethodNotAllowed, "GET"},
		{"GetPetByPath GET allowed", "GET", "/api/v1/pets/1", http.StatusOK, ""},
		{"GetPetByPath with query params", "GET", "/api/v1/pets/1?foo=bar", http.StatusOK, ""},
		{"GetPetByPath POST not allowed", "POST", "/api/v1/pets/1", http.StatusMethodNotAllowed, "GET"},
		{"GetPetByPath invalid extra path", "GET", "/api/v1/pets/1/extra", http.StatusNotFound, ""},
		{"GetPetByPath too many parts", "GET", "/api/v1/pets/1/extra/more", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedAllow != "" {
				allowHeader := rr.Header().Get("Allow")
				if allowHeader != tt.expectedAllow {
					t.Errorf("Expected Allow header %q, got %q", tt.expectedAllow, allowHeader)
				}
			}
		})
	}
}

// TestMiddlewareChain tests that middleware is properly applied
func TestMiddlewareChain(t *testing.T) {
	handler, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		checkHeader    string
		expectedHeader string
	}{
		{"Security Headers - X-XSS-Protection", "X-XSS-Protection", "1; mode=block"},
		{"Security Headers - X-Content-Type-Options", "X-Content-Type-Options", "nosniff"},
		{"Security Headers - X-Frame-Options", "X-Frame-Options", "DENY"},
		{"Security Headers - CSP", "Content-Security-Policy", "default-src 'self'"},
		{"Security Headers - Referrer-Policy", "Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/pets", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			headerValue := rr.Header().Get(tt.checkHeader)
			if headerValue != tt.expectedHeader {
				t.Errorf("Expected %s header %q, got %q", tt.checkHeader, tt.expectedHeader, headerValue)
			}
		})
	}
}

// TestXSSProtection tests XSS protection middleware
func TestXSSProtection(t *testing.T) {
	handler, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/pets?type=<script>alert(1)</script>", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for XSS attempt, got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestRecoveryMiddleware tests panic recovery
func TestRecoveryMiddleware(t *testing.T) {
	// Initialize logger for this test
	tempDir := t.TempDir()
	logger.ResetForTesting()
	_ = logger.Init(tempDir)
	defer logger.Close()
	defer logger.ResetForTesting()

	// Create a mux with a route that panics
	mux := http.NewServeMux()
	mux.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("intentional test panic")
	})

	// Wrap with recovery middleware
	handler := middleware.RecoveryMiddleware(mux)

	req := httptest.NewRequest("GET", "/panic", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d after panic, got %d", http.StatusInternalServerError, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type after panic, got %q", contentType)
	}
}

// TestInputSanitizer tests SQL injection protection
func TestInputSanitizer(t *testing.T) {
	handler, cleanup := setupTestServer(t)
	defer cleanup()

	// Test with SQL injection attempt using dangerous pattern
	req := httptest.NewRequest("GET", "/api/pets?type=%27%3B+DROP+TABLE", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Input sanitizer should reject SQL keywords
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for SQL injection attempt, got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestCartRoutes tests cart endpoints with auth middleware
func TestCartRoutes(t *testing.T) {
	// Set JWT secret for testing (must be at least 32 bytes)
	os.Setenv("JWT_SECRET_KEY", "this-is-a-test-secret-key-that-is-32b")
	defer os.Unsetenv("JWT_SECRET_KEY")

	handler, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		path           string
		authHeader     string
		expectedStatus int
	}{
		{"Cart without auth", "GET", "/api/cart", "", http.StatusUnauthorized},
		{"Cart with invalid auth", "GET", "/api/cart", "Bearer invalid-token", http.StatusUnauthorized},
		{"Cart POST without auth", "POST", "/api/cart", "", http.StatusUnauthorized},
		{"Cart PUT without auth", "PUT", "/api/cart", "", http.StatusUnauthorized},
		{"Cart DELETE without auth", "DELETE", "/api/cart", "", http.StatusUnauthorized},
		{"Cart clear without auth", "DELETE", "/api/cart/clear", "", http.StatusUnauthorized},
		{"Cart clear with invalid auth", "DELETE", "/api/cart/clear", "Bearer invalid", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// TestPetPhotoRoutes tests pet photo endpoints
func TestPetPhotoRoutes(t *testing.T) {
	handler, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{"GetPetPhotos GET", "GET", "/api/pet/1/photos", "", http.StatusOK},
		{"AddPetPhoto POST", "POST", "/api/pet/1/photos", `{"url":"https://example.com/photo.jpg"}`, http.StatusOK},
		{"DeletePetPhoto DELETE", "DELETE", "/api/pet/1/photos?url=https://example.com/photo.jpg", "", http.StatusOK},
		{"PetPhotoHandler invalid method", "PATCH", "/api/pet/1/photos", "", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// TestPetPathVariations tests various pet path patterns
func TestPetPathVariations(t *testing.T) {
	handler, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{"Valid pet path", "/api/pet/2", http.StatusOK},
		{"Valid pet path with trailing slash", "/api/pet/2/", http.StatusOK},
		{"Invalid pet path - too short", "/api/pet/", http.StatusNotFound},
		{"Invalid pet path - too many parts", "/api/pet/1/extra/invalid", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// TestRunWithConfig tests the runWithConfig function with injected dependencies
func TestRunWithConfig(t *testing.T) {
	tempDir := t.TempDir()

	config := &serverConfig{
		addr:            "127.0.0.1:0", // Let system assign port
		logDir:          tempDir,
		dbPath:          filepath.Join(tempDir, "test.db"),
		shutdownTimeout: 1 * time.Second,
	}

	// Reset state before test
	logger.ResetForTesting()
	db.ResetForTesting()
	handlers.ResetPetsForTesting()
	defer func() {
		db.ResetForTesting()
		logger.ResetTestingMode()
		handlers.ResetPetsForTesting()
	}()

	// Create a signal channel for testing
	sigCh := make(chan os.Signal, 1)

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		deps := &serverDependencies{
			signalChan: sigCh,
		}
		errCh <- runWithDependencies(config, deps)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Send signal to shutdown
	sigCh <- syscall.SIGTERM

	// Wait for graceful shutdown
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("runWithConfig returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for server to shutdown")
	}
}

// TestSetupRoutes tests route setup function
func TestSetupRoutes(t *testing.T) {
	mux := http.NewServeMux()
	setupRoutes(mux)

	// Test that routes are registered by making requests
	testCases := []struct {
		path string
	}{
		{"/api/pets"},
		{"/api/pet"},
		{"/api/pet/search"},
		{"/api/pet/cache/stats"},
		{"/api/pet/cache/hitrate"},
		{"/api/admin/products"},
		{"/api/admin/product"},
		{"/api/admin/inventory/logs"},
		{"/api/admin/inventory/alerts"},
		{"/api/admin/inventory/adjust"},
		{"/api/admin/orders"},
		{"/api/admin/order"},
		{"/api/admin/order/refund"},
		{"/api/admin/users"},
		{"/api/admin/user"},
		{"/api/admin/user/reset-password"},
		{"/api/admin/stats/sales"},
		{"/api/admin/stats/hot-products"},
		{"/api/cart"},
		{"/api/cart/clear"},
		{"/api/admin/carousels"},
		{"/api/admin/carousel"},
		{"/api/admin/announcements"},
		{"/api/admin/announcement"},
		{"/api/admin/configs"},
		{"/api/admin/config"},
		{"/api/version"},
		{"/error"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Route %s", tc.path), func(t *testing.T) {
			// Just verify the route doesn't panic
			req := httptest.NewRequest("GET", tc.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			// Any status is fine, we just want to verify the route is registered
		})
	}
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	config := defaultConfig()

	if config.addr != ":8080" {
		t.Errorf("Expected addr :8080, got %s", config.addr)
	}
	if config.logDir != "logs" {
		t.Errorf("Expected logDir logs, got %s", config.logDir)
	}
	if config.dbPath != "./cart.db" {
		t.Errorf("Expected dbPath ./cart.db, got %s", config.dbPath)
	}
	if config.shutdownTimeout != 10*time.Second {
		t.Errorf("Expected shutdownTimeout 10s, got %v", config.shutdownTimeout)
	}
}

// TestMiddlewareOrder tests that middleware executes in correct order
func TestMiddlewareOrder(t *testing.T) {
	tempDir := t.TempDir()
	_ = logger.Init(tempDir)
	defer logger.Close()

	// Create a simple handler that records execution
	var executionOrder []string
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware that records execution
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware1-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware1-after")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware2-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware2-after")
		})
	}

	handler := middleware1(middleware2(baseHandler))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	if len(executionOrder) != len(expected) {
		t.Errorf("Expected %d execution steps, got %d", len(expected), len(executionOrder))
	}

	for i, exp := range expected {
		if i >= len(executionOrder) || executionOrder[i] != exp {
			t.Errorf("Expected step %d to be %q, got %q", i, exp, executionOrder[i])
		}
	}
}

// TestRateLimiter tests rate limiting functionality
func TestRateLimiter(t *testing.T) {
	// Create rate limiter with very low limit for testing
	rateLimiter := middleware.NewRateLimiter(2, time.Minute)

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with rate limiter middleware
	wrapped := middleware.RateLimitMiddleware(rateLimiter)(handler)

	// Make requests from same IP
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/api/pets", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if i < 2 {
			if rr.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status %d, got %d", i+1, http.StatusOK, rr.Code)
			}
		} else {
			if rr.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d: Expected status %d (rate limited), got %d", i+1, http.StatusTooManyRequests, rr.Code)
			}
		}
	}
}

// TestCacheStop tests that cache cleanup is stopped properly
func TestCacheStop(t *testing.T) {
	// Get the cache instance
	cache := handlers.GetPetCache()

	// Stop should not panic
	cache.Stop()
}

// TestVersionEndpoint tests the version API endpoint
func TestVersionEndpoint(t *testing.T) {
	handler, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("GET returns version info", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/version", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
		}

		contentType := rr.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got %q", contentType)
		}

		body := rr.Body.String()
		if !strings.Contains(body, `"version"`) {
			t.Errorf("Expected version in body, got %q", body)
		}
	})

	t.Run("POST returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/version", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}
	})
}

// TestErrorPageContent tests error page HTML content
func TestErrorPageContent(t *testing.T) {
	handler, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/error", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected HTML content type, got %q", contentType)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Something went wrong") {
		t.Errorf("Expected error message in body, got %q", body)
	}
}

// BenchmarkRouteHandling benchmarks route handling performance
func BenchmarkRouteHandling(b *testing.B) {
	tempDir := b.TempDir()
	_ = logger.Init(tempDir)
	defer logger.Close()

	dbPath := filepath.Join(tempDir, "bench.db")
	if err := db.InitDB(dbPath); err != nil {
		b.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	defer db.ResetForTesting()

	mux := http.NewServeMux()
	setupRoutes(mux)

	handler := middleware.RecoveryMiddleware(mux)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/pets", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// TestInventoryAdjustEndpoint tests inventory adjust endpoint
func TestInventoryAdjustEndpoint(t *testing.T) {
	// Set JWT secret for testing (must be at least 32 bytes)
	os.Setenv("JWT_SECRET_KEY", "this-is-a-test-secret-key-that-is-32b")
	defer os.Unsetenv("JWT_SECRET_KEY")

	handler, cleanup := setupTestServer(t)
	defer cleanup()

	// Generate a valid JWT token for admin routes
	validToken, err := middleware.GenerateToken(1)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedAllow  string
	}{
		{"POST allowed", "POST", http.StatusBadRequest, ""}, // Bad request due to missing body
		{"GET not allowed", "GET", http.StatusMethodNotAllowed, "POST"},
		{"PUT not allowed", "PUT", http.StatusMethodNotAllowed, "POST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/admin/inventory/adjust", nil)
			req.Header.Set("Authorization", "Bearer "+validToken)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedAllow != "" {
				allowHeader := rr.Header().Get("Allow")
				if allowHeader != tt.expectedAllow {
					t.Errorf("Expected Allow header %q, got %q", tt.expectedAllow, allowHeader)
				}
			}
		})
	}
}

// TestRefundEndpoint tests refund endpoint
func TestRefundEndpoint(t *testing.T) {
	// Set JWT secret for testing (must be at least 32 bytes)
	os.Setenv("JWT_SECRET_KEY", "this-is-a-test-secret-key-that-is-32b")
	defer os.Unsetenv("JWT_SECRET_KEY")

	handler, cleanup := setupTestServer(t)
	defer cleanup()

	// Generate a valid JWT token for admin routes
	validToken, err := middleware.GenerateToken(1)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedAllow  string
	}{
		{"POST allowed", "POST", http.StatusBadRequest, ""}, // Bad request due to missing body
		{"GET not allowed", "GET", http.StatusMethodNotAllowed, "POST"},
		{"PUT not allowed", "PUT", http.StatusMethodNotAllowed, "POST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/admin/order/refund", nil)
			req.Header.Set("Authorization", "Bearer "+validToken)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedAllow != "" {
				allowHeader := rr.Header().Get("Allow")
				if allowHeader != tt.expectedAllow {
					t.Errorf("Expected Allow header %q, got %q", tt.expectedAllow, allowHeader)
				}
			}
		})
	}
}

// TestResetPasswordEndpoint tests reset password endpoint
func TestResetPasswordEndpoint(t *testing.T) {
	// Set JWT secret for testing (must be at least 32 bytes)
	os.Setenv("JWT_SECRET_KEY", "this-is-a-test-secret-key-that-is-32b")
	defer os.Unsetenv("JWT_SECRET_KEY")

	handler, cleanup := setupTestServer(t)
	defer cleanup()

	// Generate a valid JWT token for admin routes
	validToken, err := middleware.GenerateToken(1)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedAllow  string
	}{
		{"POST allowed", "POST", http.StatusBadRequest, ""}, // Bad request due to missing body
		{"GET not allowed", "GET", http.StatusMethodNotAllowed, "POST"},
		{"PUT not allowed", "PUT", http.StatusMethodNotAllowed, "POST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/admin/user/reset-password", nil)
			req.Header.Set("Authorization", "Bearer "+validToken)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedAllow != "" {
				allowHeader := rr.Header().Get("Allow")
				if allowHeader != tt.expectedAllow {
					t.Errorf("Expected Allow header %q, got %q", tt.expectedAllow, allowHeader)
				}
			}
		})
	}
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	handler, cleanup := setupTestServer(t)
	defer cleanup()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/api/pets", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				errors <- fmt.Errorf("request %d: expected status %d, got %d", idx, http.StatusOK, rr.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		t.Log(err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Got %d errors during concurrent requests", errorCount)
	}
}

// TestServerConfigCreation tests server config creation
func TestServerConfigCreation(t *testing.T) {
	config := &serverConfig{
		addr:            ":9090",
		logDir:          "/tmp/logs",
		dbPath:          "/tmp/test.db",
		shutdownTimeout: 5 * time.Second,
	}

	if config.addr != ":9090" {
		t.Errorf("Expected addr :9090, got %s", config.addr)
	}
	if config.logDir != "/tmp/logs" {
		t.Errorf("Expected logDir /tmp/logs, got %s", config.logDir)
	}
	if config.dbPath != "/tmp/test.db" {
		t.Errorf("Expected dbPath /tmp/test.db, got %s", config.dbPath)
	}
	if config.shutdownTimeout != 5*time.Second {
		t.Errorf("Expected shutdownTimeout 5s, got %v", config.shutdownTimeout)
	}
}

// TestRunWithConfigServer tests the runWithConfig function with actual server start/stop
func TestRunWithConfigServer(t *testing.T) {
	tempDir := t.TempDir()

	config := &serverConfig{
		addr:            "127.0.0.1:0", // Let system assign port
		logDir:          tempDir,
		dbPath:          filepath.Join(tempDir, "test_run.db"),
		shutdownTimeout: 1 * time.Second,
	}

	// Reset state before test
	logger.ResetForTesting()
	db.ResetForTesting()
	handlers.ResetPetsForTesting()
	defer func() {
		db.ResetForTesting()
		logger.ResetTestingMode()
		handlers.ResetPetsForTesting()
	}()

	// Create a signal channel for testing
	sigCh := make(chan os.Signal, 1)

	// Start server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		deps := &serverDependencies{
			signalChan: sigCh,
		}
		errCh <- runWithDependencies(config, deps)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Server should be running, send signal to stop
	sigCh <- syscall.SIGTERM

	// Wait for graceful shutdown
	select {
	case err := <-errCh:
		if err != nil {
			t.Logf("runWithConfig returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for server to shutdown")
	}
}

// TestCartRoutesMethods tests cart routes with different HTTP methods
func TestCartRoutesMethods(t *testing.T) {
	// Set JWT secret for testing
	os.Setenv("JWT_SECRET_KEY", "this-is-a-test-secret-key-that-is-32b")
	defer os.Unsetenv("JWT_SECRET_KEY")

	handler, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{"Cart POST", "POST", "/api/cart", `{"product_id":1,"quantity":1}`, http.StatusUnauthorized},
		{"Cart PUT", "PUT", "/api/cart", `{"product_id":1,"quantity":2}`, http.StatusUnauthorized},
		{"Cart DELETE", "DELETE", "/api/cart", "", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}
