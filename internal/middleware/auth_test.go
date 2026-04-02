package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// setupIsolatedTestEnvironment sets up an isolated test environment
// This should be used when tests need to control the environment completely
func setupIsolatedTestEnvironment(t *testing.T) {
	t.Helper()
	// Store original values
	origSecret := os.Getenv("JWT_SECRET_KEY")
	origEnv := os.Getenv("APP_ENV")

	// Clear the override before each test
	ResetJWTSecret()
	// Clear environment variables
	os.Unsetenv("JWT_SECRET_KEY")
	os.Unsetenv("APP_ENV")

	// Restore original values after test
	t.Cleanup(func() {
		if origSecret != "" {
			os.Setenv("JWT_SECRET_KEY", origSecret)
		} else {
			os.Unsetenv("JWT_SECRET_KEY")
		}
		if origEnv != "" {
			os.Setenv("APP_ENV", origEnv)
		} else {
			os.Unsetenv("APP_ENV")
		}
		// Restore test secret from TestMain
		SetJWTSecret(testJWTSecret)
	})
}

// ========== GetJWTSecretKey Tests ==========

// TestGetJWTSecretKey_WithEnvVar tests getting JWT secret from environment variable
func TestGetJWTSecretKey_WithEnvVar(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	expectedSecret := "this-is-a-secure-256-bit-secret-key"
	os.Setenv("JWT_SECRET_KEY", expectedSecret)
	os.Setenv("APP_ENV", "production")

	secret, err := GetJWTSecretKey()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if string(secret) != expectedSecret {
		t.Errorf("expected secret to be %q, got %q", expectedSecret, string(secret))
	}
}

// TestGetJWTSecretKey_DevelopmentDefault tests development environment uses default key
func TestGetJWTSecretKey_DevelopmentDefault(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	os.Setenv("APP_ENV", "development")

	secret, err := GetJWTSecretKey()
	if err != nil {
		t.Fatalf("expected no error in development mode, got: %v", err)
	}

	if string(secret) != defaultJWTSecretKey {
		t.Errorf("expected default secret in development mode, got %q", string(secret))
	}
}

// TestGetJWTSecretKey_ProductionMissingSecret tests production returns error when secret is missing
func TestGetJWTSecretKey_ProductionMissingSecret(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	os.Setenv("APP_ENV", "production")

	_, err := GetJWTSecretKey()
	if err == nil {
		t.Fatal("expected error in production when JWT_SECRET_KEY is missing, got nil")
	}

	if err != ErrMissingJWTSecret {
		t.Errorf("expected ErrMissingJWTSecret, got: %v", err)
	}
}

// TestGetJWTSecretKey_ShortKeyInProduction tests production returns error for short key
func TestGetJWTSecretKey_ShortKeyInProduction(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	os.Setenv("JWT_SECRET_KEY", "short")
	os.Setenv("APP_ENV", "production")

	_, err := GetJWTSecretKey()
	if err == nil {
		t.Fatal("expected error for short key in production, got nil")
	}

	if !strings.Contains(err.Error(), "at least 32 bytes") {
		t.Errorf("expected error about key length, got: %v", err)
	}
}

// TestGetJWTSecretKey_ShortKeyInDevelopment tests development falls back to default for short key
func TestGetJWTSecretKey_ShortKeyInDevelopment(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	os.Setenv("JWT_SECRET_KEY", "short")
	os.Setenv("APP_ENV", "development")

	secret, err := GetJWTSecretKey()
	if err != nil {
		t.Fatalf("expected no error in development mode, got: %v", err)
	}

	if string(secret) != defaultJWTSecretKey {
		t.Errorf("expected default secret for short key in development, got %q", string(secret))
	}
}

// TestGetJWTSecretKey_SetJWTSecretOverride tests SetJWTSecret override functionality
func TestGetJWTSecretKey_SetJWTSecretOverride(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	overrideSecret := "override-secret-key-256-bit-for-testing-only-now"
	SetJWTSecret(overrideSecret)

	// Even in production with no env var, override should work
	os.Setenv("APP_ENV", "production")

	secret, err := GetJWTSecretKey()
	if err != nil {
		t.Fatalf("expected no error with override, got: %v", err)
	}

	if string(secret) != overrideSecret {
		t.Errorf("expected override secret %q, got %q", overrideSecret, string(secret))
	}
}

// TestGetJWTSecretKey_ResetJWTSecret tests ResetJWTSecret restores default behavior
func TestGetJWTSecretKey_ResetJWTSecret(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	// Set an override
	SetJWTSecret("override-secret")

	// Reset it
	ResetJWTSecret()

	// Should now use default in development
	os.Setenv("APP_ENV", "development")

	secret, err := GetJWTSecretKey()
	if err != nil {
		t.Fatalf("expected no error after reset, got: %v", err)
	}

	if string(secret) != defaultJWTSecretKey {
		t.Errorf("expected default secret after reset, got %q", string(secret))
	}
}

// TestGetJWTSecretKey_OverrideTakesPrecedence tests that override takes precedence over env var
func TestGetJWTSecretKey_OverrideTakesPrecedence(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	envSecret := "env-secret-key-256-bit-secure-key"
	overrideSecret := "override-secret-key-for-testing-only"

	os.Setenv("JWT_SECRET_KEY", envSecret)
	os.Setenv("APP_ENV", "production")

	SetJWTSecret(overrideSecret)

	secret, err := GetJWTSecretKey()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if string(secret) != overrideSecret {
		t.Errorf("expected override secret %q, got %q", overrideSecret, string(secret))
	}
}

// ========== GenerateToken Additional Tests ==========

// TestGenerateToken_MissingSecret tests token generation fails without secret
func TestGenerateToken_MissingSecret(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	os.Setenv("APP_ENV", "production")

	_, err := GenerateToken(12345)
	if err == nil {
		t.Fatal("expected error when JWT secret is missing, got nil")
	}
}

// ========== AuthMiddleware Additional Tests ==========

// TestAuthMiddleware_InvalidTokenSignature tests middleware with token signed with wrong secret
func TestAuthMiddleware_InvalidTokenSignature(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	SetJWTSecret("correct-secret-key-256-bit-for-signing")

	// Generate token with different secret
	wrongSecret := "wrong-secret-key-256-bit-for-signing"
	SetJWTSecret(wrongSecret)
	token, _ := GenerateToken(12345)
	ResetJWTSecret()

	// Now set the correct secret for verification
	SetJWTSecret("correct-secret-key-256-bit-for-signing")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid signature")
	})

	authMiddleware := AuthMiddleware(testHandler)
	rr := httptest.NewRecorder()

	authMiddleware.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

// TestAuthMiddleware_MissingSecretKey tests middleware returns 500 when secret key is missing
func TestAuthMiddleware_MissingSecretKey(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	os.Setenv("APP_ENV", "production")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer some-token")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when secret is missing")
	})

	authMiddleware := AuthMiddleware(testHandler)
	rr := httptest.NewRecorder()

	authMiddleware.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

// TestAuthMiddleware_EmptyBearerToken tests middleware with empty Bearer token
func TestAuthMiddleware_EmptyBearerToken(t *testing.T) {
	setupIsolatedTestEnvironment(t)

	SetJWTSecret("test-secret-key-256-bit-for-jwt-signing")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with empty token")
	})

	authMiddleware := AuthMiddleware(testHandler)
	rr := httptest.NewRecorder()

	authMiddleware.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

// ========== GetUserID Additional Tests ==========

// TestGetUserID_ContextWithoutUserID tests GetUserID with context that has no user ID
func TestGetUserID_ContextWithoutUserID(t *testing.T) {
	ctx := context.Background()

	userID, ok := GetUserID(ctx)

	if ok {
		t.Error("expected ok to be false")
	}

	if userID != 0 {
		t.Errorf("expected user ID to be 0, got %d", userID)
	}
}

// TestGetUserID_WrongType tests GetUserID returns false when value is wrong type
func TestGetUserID_WrongType(t *testing.T) {
	// Create context with int instead of int64
	ctx := context.WithValue(context.Background(), UserIDKey, 123)

	_, ok := GetUserID(ctx)

	if ok {
		t.Error("expected ok to be false when user ID is wrong type (int instead of int64)")
	}
}

// TestGetUserID_StringType tests GetUserID returns false when value is string instead of int64
func TestGetUserID_StringType(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserIDKey, "123")

	_, ok := GetUserID(ctx)

	if ok {
		t.Error("expected ok to be false when user ID is string type")
	}
}

// ========== isDevelopment Tests ==========

// TestIsDevelopment tests the isDevelopment function
func TestIsDevelopment(t *testing.T) {
	// Store original value
	origEnv := os.Getenv("APP_ENV")
	defer func() {
		if origEnv != "" {
			os.Setenv("APP_ENV", origEnv)
		} else {
			os.Unsetenv("APP_ENV")
		}
	}()

	tests := []struct {
		name     string
		appEnv   string
		expected bool
	}{
		{"development mode", "development", true},
		{"production mode", "production", false},
		{"staging mode", "staging", false},
		{"empty env", "", false},
		{"test mode", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("APP_ENV", tt.appEnv)

			result := isDevelopment()
			if result != tt.expected {
				t.Errorf("isDevelopment() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ========== Constants and Variables Tests ==========

// TestErrMissingJWTSecret_Variable tests that ErrMissingJWTSecret is defined correctly
func TestErrMissingJWTSecret_Variable(t *testing.T) {
	if ErrMissingJWTSecret == nil {
		t.Error("ErrMissingJWTSecret should not be nil")
	}

	expectedMsg := "JWT_SECRET_KEY environment variable is not set"
	if ErrMissingJWTSecret.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, ErrMissingJWTSecret.Error())
	}
}

// TestDefaultJWTSecretKey_Length tests default key is at least 32 bytes
func TestDefaultJWTSecretKey_Length(t *testing.T) {
	if len(defaultJWTSecretKey) < 32 {
		t.Errorf("defaultJWTSecretKey should be at least 32 bytes, got %d", len(defaultJWTSecretKey))
	}
}

// TestUserIDKey_Constant tests UserIDKey constant
func TestUserIDKey_Constant(t *testing.T) {
	if string(UserIDKey) != "userID" {
		t.Errorf("expected UserIDKey to be 'userID', got %q", string(UserIDKey))
	}
}

// TestJWTClaims_Type tests JWTClaims struct type
func TestJWTClaims_Type(t *testing.T) {
	claims := JWTClaims{
		UserID: 123,
	}

	if claims.UserID != 123 {
		t.Errorf("expected UserID to be 123, got %d", claims.UserID)
	}
}
