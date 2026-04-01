package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// UserIDKey is the context key for user ID
type contextKey string

const UserIDKey contextKey = "userID"

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

		// Parse and validate JWT token
		// In production, use a proper secret key from config/environment
		secretKey := []byte("your-256-bit-secret-key-here")

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
	secretKey := []byte("your-256-bit-secret-key-here")
	claims := &JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "petshop",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}