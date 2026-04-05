// Package handlers provides HTTP handlers for the petshop API.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"petshop/internal/db"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// HealthCheck handles the /health endpoint
// Returns the health status of the service including database connectivity
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check database connectivity
	dbStatus := "connected"
	dbHealthy := checkDatabase()

	if !dbHealthy {
		dbStatus = "disconnected"
	}

	// Determine overall health status
	overallStatus := "healthy"
	httpStatus := http.StatusOK
	if !dbHealthy {
		overallStatus = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks: map[string]string{
			"database": dbStatus,
			"server":   "running",
		},
	}

	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(response)
}

// checkDatabase verifies database connectivity by executing a simple query
// Returns true if database is healthy, false otherwise
func checkDatabase() bool {
	database := db.GetDB()
	if database == nil {
		return false
	}

	// Execute simple query with timeout to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var result int
	err := database.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	return err == nil && result == 1
}
