package middleware

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// UserIDKey is the context key for user ID
type contextKey string

const UserIDKey contextKey = "userID"

// ErrMissingJWTSecret is returned when JWT_SECRET_KEY environment variable is not set
var ErrMissingJWTSecret = errors.New("JWT_SECRET_KEY environment variable is not set")

// defaultJWTSecretKey is used only in development when JWT_SECRET_KEY is not set
const defaultJWTSecretKey = "dev-only-256-bit-secret-key-do-not-use-in-prod"

// GetJWTSecretKey returns the JWT secret key from environment variable
// Falls back to development default if not set, but enforces minimum 32 bytes length
func GetJWTSecretKey() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		secret = defaultJWTSecretKey
	}
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET_KEY must be at least 32 bytes long")
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
			http.Error(w, "internal server error: "+err.Error(), http.StatusInternalServerError)
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

// GenerateToken generates a JWT token for a user (utility for testing)
func GenerateToken(userID int64) (string, error) {
	secretKey, err := GetJWTSecretKey()
	if err != nil {
		return "", err
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