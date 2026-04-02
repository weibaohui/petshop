package middleware

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"petshop/internal/logger"

	"github.com/golang-jwt/jwt/v5"
)

// UserIDKey is the context key for user ID
type contextKey string

const UserIDKey contextKey = "userID"

// ErrMissingJWTSecret is returned when JWT_SECRET_KEY environment variable is not set
var ErrMissingJWTSecret = errors.New("JWT_SECRET_KEY environment variable is not set")

// defaultJWTSecretKey is used only in development when JWT_SECRET_KEY is not set
const defaultJWTSecretKey = "dev-only-256-bit-secret-key-do-not-use-in-prod"

// jwtSecret holds the injected JWT secret key for testing or custom configuration
var jwtSecret []byte

// SetJWTSecret allows injecting a JWT secret key programmatically.
// This is primarily used for testing to ensure tests use an isolated test key.
// When set, this takes precedence over environment variables.
func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}

// ClearJWTSecret clears the injected JWT secret key.
// After calling this, GetJWTSecretKey will fall back to environment variables.
func ClearJWTSecret() {
	jwtSecret = nil
}

// isDevelopment returns true if APP_ENV is set to "development"
func isDevelopment() bool {
	return os.Getenv("APP_ENV") == "development"
}

// GetJWTSecretKey returns the JWT secret key.
// Priority:
// 1. Injected secret via SetJWTSecret (for testing)
// 2. JWT_SECRET_KEY environment variable
// 3. Development default only if APP_ENV is "development"
// In production, returns error if JWT_SECRET_KEY is not set or is too short
func GetJWTSecretKey() ([]byte, error) {
	// Return injected secret if available (allows test isolation)
	if jwtSecret != nil {
		return jwtSecret, nil
	}

	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		if isDevelopment() {
			secret = defaultJWTSecretKey
			logger.Warn("using default JWT secret key in development mode", map[string]interface{}{
				"warning": "do not use default key in production",
			})
		} else {
			return nil, ErrMissingJWTSecret
		}
	}
	if len(secret) < 32 {
		if isDevelopment() {
			logger.Warn("JWT_SECRET_KEY is too short, using default key in development mode", map[string]interface{}{
				"warning":    "provided key length too short",
				"key_length": len(secret),
			})
			secret = defaultJWTSecretKey
		} else {
			return nil, errors.New("JWT_SECRET_KEY must be at least 32 bytes long")
		}
	}
	return []byte(secret), nil
}

// JWTClaims represents the JWT claims structure
type JWTClaims struct {
	UserID int64 `json:"userId"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT token and extracts user ID
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "unauthorized: missing authorization header", http.StatusUnauthorized)
			return
		}

		// Extract token (support both "Bearer <token>" and raw token)
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader && !strings.Contains(authHeader, " ") {
			tokenString = authHeader
		}

		secretKey, err := GetJWTSecretKey()
		if err != nil {
			logger.Error("failed to get JWT secret key", map[string]interface{}{
				"error": err.Error(),
			})
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secretKey, nil
		})

		if err != nil {
			http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok || !token.Valid {
			http.Error(w, "unauthorized: invalid token claims", http.StatusUnauthorized)
			return
		}

		// Validate user ID
		if claims.UserID <= 0 {
			http.Error(w, "unauthorized: invalid user ID", http.StatusUnauthorized)
			return
		}

		// Add user ID to request context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}

// GenerateTokenWithSecret generates a JWT token with the provided secret key.
// If secretKey is nil or empty, it falls back to GetJWTSecretKey().
func GenerateTokenWithSecret(userID int64, secretKey []byte) (string, error) {
	if len(secretKey) == 0 {
		var err error
		secretKey, err = GetJWTSecretKey()
		if err != nil {
			return "", err
		}
	}
	claims := &JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "petshop",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// GenerateToken generates a JWT token for a user (utility for testing).
// Uses the injected secret (SetJWTSecret) if available, otherwise uses GetJWTSecretKey().
func GenerateToken(userID int64) (string, error) {
	return GenerateTokenWithSecret(userID, nil)
}