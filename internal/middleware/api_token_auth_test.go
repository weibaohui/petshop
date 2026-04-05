package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"petshop/internal/db"
	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
)

func setupTestDBForMiddleware(t *testing.T) func() {
	// Reset database state
	db.ResetForTesting()

	// Use test database file
	testDBPath := "./test_middleware_api_tokens.db"

	// Remove existing test database
	os.Remove(testDBPath)

	// Initialize database
	err := db.InitDB(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Cleanup function
	return func() {
		db.Close()
		os.Remove(testDBPath)
		db.ResetForTesting()
	}
}

func TestAPITokenAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	cleanup := setupTestDBForMiddleware(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with missing authorization header")
	})

	middleware := APITokenAuthMiddleware(testHandler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing authorization header")
}

func TestAPITokenAuthMiddleware_InvalidAuthorizationFormat(t *testing.T) {
	cleanup := setupTestDBForMiddleware(t)
	defer cleanup()

	tests := []struct {
		name          string
		authHeader    string
		expectedError string
	}{
		{
			name:          "no prefix",
			authHeader:    "just_token_value",
			expectedError: "invalid authorization format",
		},
		{
			name:          "wrong prefix",
			authHeader:    "Basic dGVzdA==",
			expectedError: "invalid authorization format",
		},
		{
			name:          "empty token after Bearer",
			authHeader:    "Bearer ",
			expectedError: "invalid token format",
		},
		{
			name:          "empty token after Token",
			authHeader:    "Token ",
			expectedError: "invalid token format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
			req.Header.Set("Authorization", tt.authHeader)
			w := httptest.NewRecorder()

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("handler should not be called with invalid authorization format")
			})

			middleware := APITokenAuthMiddleware(testHandler)
			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedError)
		})
	}
}

func TestAPITokenAuthMiddleware_BearerTokenSuccess(t *testing.T) {
	cleanup := setupTestDBForMiddleware(t)
	defer cleanup()

	// Create a valid token in the database
	repo := db.NewAPITokenRepository()
	rawToken := "psk_test_bearer_token_123"
	tokenHash := db.HashToken(rawToken)
	token := &models.APIToken{
		Name:        "Test Bearer Token",
		TokenHash:   tokenHash,
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read,write",
	}
	err := repo.Create(token)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	w := httptest.NewRecorder()

	var capturedContext context.Context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContext = r.Context()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	})

	middleware := APITokenAuthMiddleware(testHandler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedContext)

	// Verify token is in context
	apiToken, ok := GetAPIToken(capturedContext)
	assert.True(t, ok)
	assert.NotNil(t, apiToken)
	assert.Equal(t, token.Name, apiToken.Name)
	assert.Equal(t, token.Permissions, apiToken.Permissions)
}

func TestAPITokenAuthMiddleware_TokenPrefixSuccess(t *testing.T) {
	cleanup := setupTestDBForMiddleware(t)
	defer cleanup()

	// Create a valid token in the database
	repo := db.NewAPITokenRepository()
	rawToken := "psk_test_token_prefix_456"
	tokenHash := db.HashToken(rawToken)
	token := &models.APIToken{
		Name:        "Test Token Prefix",
		TokenHash:   tokenHash,
		Status:      "active",
		CreatedBy:   1,
		Permissions: "admin",
	}
	err := repo.Create(token)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Token "+rawToken)
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := APITokenAuthMiddleware(testHandler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPITokenAuthMiddleware_InvalidToken(t *testing.T) {
	cleanup := setupTestDBForMiddleware(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid_token_value")
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid token")
	})

	middleware := APITokenAuthMiddleware(testHandler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired token")
}

func TestAPITokenAuthMiddleware_DisabledToken(t *testing.T) {
	cleanup := setupTestDBForMiddleware(t)
	defer cleanup()

	// Create a disabled token
	repo := db.NewAPITokenRepository()
	rawToken := "psk_disabled_token_789"
	tokenHash := db.HashToken(rawToken)
	token := &models.APIToken{
		Name:        "Disabled Token",
		TokenHash:   tokenHash,
		Status:      "disabled",
		CreatedBy:   1,
		Permissions: "read",
	}
	err := repo.Create(token)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with disabled token")
	})

	middleware := APITokenAuthMiddleware(testHandler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired token")
}

func TestAPITokenAuthMiddleware_ExpiredToken(t *testing.T) {
	cleanup := setupTestDBForMiddleware(t)
	defer cleanup()

	// Create an expired token
	repo := db.NewAPITokenRepository()
	rawToken := "psk_expired_token_abc"
	tokenHash := db.HashToken(rawToken)
	pastTime := time.Now().Add(-24 * time.Hour)
	token := &models.APIToken{
		Name:        "Expired Token",
		TokenHash:   tokenHash,
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
		ExpiresAt:   &pastTime,
	}
	err := repo.Create(token)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with expired token")
	})

	middleware := APITokenAuthMiddleware(testHandler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired token")
}

func TestAPITokenAuthMiddleware_NonExpiredToken(t *testing.T) {
	cleanup := setupTestDBForMiddleware(t)
	defer cleanup()

	// Create a token with future expiration
	repo := db.NewAPITokenRepository()
	rawToken := "psk_valid_expiring_token"
	tokenHash := db.HashToken(rawToken)
	futureTime := time.Now().Add(24 * time.Hour)
	token := &models.APIToken{
		Name:        "Valid Expiring Token",
		TokenHash:   tokenHash,
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
		ExpiresAt:   &futureTime,
	}
	err := repo.Create(token)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := APITokenAuthMiddleware(testHandler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetAPIToken(t *testing.T) {
	t.Run("get token from context with valid token", func(t *testing.T) {
		token := &models.APIToken{
			ID:          1,
			Name:        "Test Token",
			Permissions: "read,write",
		}
		ctx := context.WithValue(context.Background(), APITokenKey, token)

		retrieved, ok := GetAPIToken(ctx)
		assert.True(t, ok)
		assert.NotNil(t, retrieved)
		assert.Equal(t, token.ID, retrieved.ID)
		assert.Equal(t, token.Name, retrieved.Name)
	})

	t.Run("get token from empty context", func(t *testing.T) {
		ctx := context.Background()

		retrieved, ok := GetAPIToken(ctx)
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("get token with wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), APITokenKey, "not a token")

		retrieved, ok := GetAPIToken(ctx)
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		permissions string
		checkPerm  string
		expected   bool
	}{
		{
			name:       "has read permission",
			permissions: "read",
			checkPerm:  "read",
			expected:   true,
		},
		{
			name:       "has write permission in list",
			permissions: "read,write,delete",
			checkPerm:  "write",
			expected:   true,
		},
		{
			name:       "admin permission allows all",
			permissions: "admin",
			checkPerm:  "delete",
			expected:   true,
		},
		{
			name:       "admin in list allows all",
			permissions: "read,admin",
			checkPerm:  "write",
			expected:   true,
		},
		{
			name:       "missing permission",
			permissions: "read",
			checkPerm:  "write",
			expected:   false,
		},
		{
			name:       "empty permissions defaults to read only",
			permissions: "",
			checkPerm:  "read",
			expected:   true,
		},
		{
			name:       "empty permissions deny write",
			permissions: "",
			checkPerm:  "write",
			expected:   false,
		},
		{
			name:       "permissions with spaces",
			permissions: "read, write , delete",
			checkPerm:  "write",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &models.APIToken{
				ID:          1,
				Name:        "Test Token",
				Permissions: tt.permissions,
			}
			ctx := context.WithValue(context.Background(), APITokenKey, token)

			result := HasPermission(ctx, tt.checkPerm)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasPermission_NoTokenInContext(t *testing.T) {
	ctx := context.Background()
	result := HasPermission(ctx, "read")
	assert.False(t, result)
}

func TestHasPermission_NilToken(t *testing.T) {
	ctx := context.WithValue(context.Background(), APITokenKey, (*models.APIToken)(nil))
	result := HasPermission(ctx, "read")
	assert.False(t, result)
}

func TestAPITokenKey_Constant(t *testing.T) {
	assert.Equal(t, apiTokenContextKey("apiToken"), APITokenKey)
}

func TestAPITokenAuthMiddleware_UpdatesLastUsedAt(t *testing.T) {
	cleanup := setupTestDBForMiddleware(t)
	defer cleanup()

	// Create a valid token
	repo := db.NewAPITokenRepository()
	rawToken := "psk_test_last_used"
	tokenHash := db.HashToken(rawToken)
	token := &models.APIToken{
		Name:        "Test Last Used",
		TokenHash:   tokenHash,
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
	}
	err := repo.Create(token)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := APITokenAuthMiddleware(testHandler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Give the goroutine time to update
	time.Sleep(100 * time.Millisecond)

	// Verify last used was updated
	updated, _ := repo.GetByTokenHash(tokenHash)
	assert.NotNil(t, updated.LastUsedAt)
}

func TestAPITokenAuthMiddleware_ContentTypeHeader(t *testing.T) {
	cleanup := setupTestDBForMiddleware(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	middleware := APITokenAuthMiddleware(testHandler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	contentType := w.Header().Get("Content-Type")
	assert.Equal(t, "application/json", contentType)
	assert.True(t, strings.Contains(w.Body.String(), "{"))
}
