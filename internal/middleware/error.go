package middleware

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"time"

	"petshop/internal/logger"
)

// ErrorResponse is the structure for friendly error responses
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"traceId,omitempty"`
}

// RecoveryMiddleware catches panics and returns a friendly error page
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				traceID := generateTraceID()
				logger.Error("panic recovered", map[string]interface{}{
					"trace_id": traceID,
					"error":    err,
					"stack":    string(debug.Stack()),
					"path":     r.URL.Path,
					"method":   r.Method,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				resp := ErrorResponse{
					Code:    http.StatusInternalServerError,
					Message: "An internal error occurred. Please try again later.",
					TraceID: traceID,
				}
				_ = json.NewEncoder(w).Encode(resp)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestLoggerMiddleware logs all incoming requests
func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapper, r)

		logger.Access("http request", map[string]interface{}{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      wrapper.statusCode,
			"duration":    time.Since(start).String(),
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
		})
	})
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func generateTraceID() string {
	return time.Now().Format("20060102150405.000000")
}
