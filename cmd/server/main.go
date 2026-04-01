package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"petshop/internal/handlers"
	"petshop/internal/logger"
	"petshop/internal/middleware"
)

func main() {
	fmt.Println("Project: petshop")

	// Initialize logger
	logger.Init("logs")

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
			w.Header().Set("Allow", "GET, PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/pet/search", handlers.SearchPets)
	mux.HandleFunc("/api/pet/cache/stats", handlers.GetCacheStats)
	mux.HandleFunc("/api/pet/cache/hitrate", handlers.GetCacheHitRate)
	mux.HandleFunc("/api/pet/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/photos") {
			handlers.PetPhotoHandler(w, r)
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

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", chain))
}

func run() error {
	// Application initialization
	return nil
}
