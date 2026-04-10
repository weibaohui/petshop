package middleware

import (
	"context"
	"net/http"
	"strings"

	"petshop/internal/db"
	"petshop/internal/logger"
	"petshop/internal/models"
)

// APITokenKey is the context key for API token
type apiTokenContextKey string

// APITokenKey is the context key used to store and retrieve the API token from context
const APITokenKey apiTokenContextKey = "apiToken"

// APITokenAuthMiddleware validates API token for open API endpoints
func APITokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "unauthorized: missing authorization header"}`))
			return
		}

		// Extract token (support both "Bearer <token>" and "Token <token>")
		var tokenString string
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else if strings.HasPrefix(authHeader, "Token ") {
			tokenString = strings.TrimPrefix(authHeader, "Token ")
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "unauthorized: invalid authorization format, use 'Bearer token' or 'Token token'"}`))
			return
		}

		if tokenString == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "unauthorized: invalid token format"}`))
			return
		}

		// Hash the token and validate
		tokenHash := db.HashToken(tokenString)
		repo := db.NewAPITokenRepository()

		valid, token := repo.IsTokenValid(tokenHash)
		if !valid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "unauthorized: invalid or expired token"}`))
			return
		}

		// Update last used time asynchronously
		go func(tokenID int64) {
			if err := repo.UpdateLastUsedAt(tokenID); err != nil {
				logger.Error("failed to update token last used time", map[string]interface{}{
					"error":    err.Error(),
					"token_id": tokenID,
				})
			}
		}(token.ID)

		// Add token info to request context
		ctx := context.WithValue(r.Context(), APITokenKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetAPIToken extracts API token from context
func GetAPIToken(ctx context.Context) (*models.APIToken, bool) {
	token, ok := ctx.Value(APITokenKey).(*models.APIToken)
	return token, ok
}

// HasPermission checks if the API token has the required permission
func HasPermission(ctx context.Context, permission string) bool {
	token, ok := ctx.Value(APITokenKey).(*models.APIToken)
	if !ok || token == nil {
		return false
	}

	// If permissions is empty, only allow read
	if token.Permissions == "" {
		return permission == "read"
	}

	// Check if permission is in the list
	permissions := strings.Split(token.Permissions, ",")
	for _, p := range permissions {
		if strings.TrimSpace(p) == permission || strings.TrimSpace(p) == "admin" {
			return true
		}
	}
	return false
}
