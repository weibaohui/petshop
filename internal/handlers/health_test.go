package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"petshop/internal/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	// Reset and initialize test database
	db.ResetForTesting()
	err := db.InitDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(db.ResetForTesting)

	tests := []struct {
		name               string
		wantStatusCode     int
		wantOverallStatus  string
		wantDatabaseStatus string
		wantServerStatus   string
		wantContentType    string
	}{
		{
			name:               "health check returns healthy when database is connected",
			wantStatusCode:     http.StatusOK,
			wantOverallStatus:  "healthy",
			wantDatabaseStatus: "connected",
			wantServerStatus:   "running",
			wantContentType:    "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			HealthCheck(w, req)

			// Verify HTTP status code
			assert.Equal(t, tt.wantStatusCode, w.Code)

			// Verify Content-Type header
			assert.Equal(t, tt.wantContentType, w.Header().Get("Content-Type"))

			// Parse response
			var response HealthResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)

			// Verify response structure
			assert.Equal(t, tt.wantOverallStatus, response.Status)
			assert.NotEmpty(t, response.Timestamp)
			assert.NotNil(t, response.Checks)

			// Verify checks
			assert.Equal(t, tt.wantDatabaseStatus, response.Checks["database"])
			assert.Equal(t, tt.wantServerStatus, response.Checks["server"])

			// Verify timestamp is valid RFC3339 format
			_, err = time.Parse(time.RFC3339, response.Timestamp)
			assert.NoError(t, err, "timestamp should be valid RFC3339 format")
		})
	}
}

func TestHealthCheckWithDBDisconnected(t *testing.T) {
	// Test with no database initialized (simulates DB failure)
	db.ResetForTesting()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	HealthCheck(w, req)

	// Verify HTTP status code is 503 Service Unavailable
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	// Parse response
	var response HealthResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Verify response indicates unhealthy status
	assert.Equal(t, "unhealthy", response.Status)
	assert.Equal(t, "disconnected", response.Checks["database"])
	assert.Equal(t, "running", response.Checks["server"])
}

func TestHealthCheckResponseStructure(t *testing.T) {
	// Reset and initialize test database
	db.ResetForTesting()
	err := db.InitDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(db.ResetForTesting)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	HealthCheck(w, req)

	// Verify response can be parsed as JSON
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify required fields exist
	assert.Contains(t, response, "status")
	assert.Contains(t, response, "timestamp")
	assert.Contains(t, response, "checks")

	// Verify checks field is a map with required keys
	checks, ok := response["checks"].(map[string]interface{})
	require.True(t, ok, "checks should be a map")
	assert.Contains(t, checks, "database")
	assert.Contains(t, checks, "server")
}

func TestCheckDatabase(t *testing.T) {
	t.Run("database is healthy when initialized", func(t *testing.T) {
		// Reset and initialize test database
		db.ResetForTesting()
		err := db.InitDB(":memory:")
		require.NoError(t, err)
		t.Cleanup(db.ResetForTesting)

		healthy := checkDatabase()
		assert.True(t, healthy)
	})

	t.Run("database is unhealthy when not initialized", func(t *testing.T) {
		// Reset database to simulate disconnected state
		db.ResetForTesting()

		healthy := checkDatabase()
		assert.False(t, healthy)
	})
}
