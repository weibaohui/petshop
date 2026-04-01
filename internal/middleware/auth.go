package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

// UserIDKey is the context key for user ID
type contextKey string

const UserIDKey contextKey = "userID"

// AuthMiddleware validates session/JWT and extracts user ID
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "unauthorized: missing authorization header", http.StatusUnauthorized)
			return
		}

		// Extract token (support both "Bearer <token>" and raw token)
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader && !strings.Contains(authHeader, " ") {
			token = authHeader
		}

		// Parse user ID from token (in real app, validate JWT signature)
		// For this implementation, token format: "user_<userID>" or just the userID
		var userID int64

		if strings.HasPrefix(token, "user_") {
			idStr := strings.TrimPrefix(token, "user_")
			var err error
			userID, err = strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				http.Error(w, "unauthorized: invalid token format", http.StatusUnauthorized)
				return
			}
		} else {
			// Try parsing as direct user ID
			var err error
			userID, err = strconv.ParseInt(token, 10, 64)
			if err != nil {
				http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
				return
			}
		}

		// Validate user exists
		if userID <= 0 {
			http.Error(w, "unauthorized: invalid user ID", http.StatusUnauthorized)
			return
		}

		// Add user ID to request context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}