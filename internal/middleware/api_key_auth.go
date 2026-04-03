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
type apiTokenKey string

const APITokenContextKey apiTokenKey = "apiToken"

// apiTokenRepo is the repository for API tokens
var apiTokenRepo = db.NewAPITokenRepository()

// APIKeyAuthMiddleware validates API key/token and extracts token info
func APIKeyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from header
		authHeader := r.Header.Get("X-API-Key")
		if authHeader == "" {
			authHeader = r.Header.Get("Authorization")
			// Support "Bearer <token>" format
			if strings.HasPrefix(authHeader, "Bearer ") {
				authHeader = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized: missing API key"}`))
			return
		}

		// Validate token
		token, err := apiTokenRepo.ValidateToken(authHeader)
		if err != nil {
			logger.Error("failed to validate api token", map[string]interface{}{
				"error": err.Error(),
			})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal server error"}`))
			return
		}

		if token == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized: invalid or expired API key"}`))
			return
		}

		// Add token info to request context
		ctx := context.WithValue(r.Context(), APITokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetAPIToken extracts API token from context
func GetAPIToken(ctx context.Context) (*models.APIToken, bool) {
	token, ok := ctx.Value(APITokenContextKey).(*models.APIToken)
	return token, ok
}

// OptionalAPIKeyAuthMiddleware optionally validates API key but doesn't require it
func OptionalAPIKeyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from header
		authHeader := r.Header.Get("X-API-Key")
		if authHeader == "" {
			authHeader = r.Header.Get("Authorization")
			// Support "Bearer <token>" format
			if strings.HasPrefix(authHeader, "Bearer ") {
				authHeader = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// If token provided, validate it
		if authHeader != "" {
			token, err := apiTokenRepo.ValidateToken(authHeader)
			if err != nil {
				logger.Error("failed to validate api token", map[string]interface{}{
					"error": err.Error(),
				})
			}

			if token != nil {
				// Add token info to request context
				ctx := context.WithValue(r.Context(), APITokenContextKey, token)
				r = r.WithContext(ctx)
			}
		}

		next.ServeHTTP(w, r)
	})
}
