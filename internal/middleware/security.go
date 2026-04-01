package middleware

import (
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"petshop/internal/logger"
	"petshop/internal/validator"
)

// SecurityHeaders adds security headers to responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent XSS
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		// Content Security Policy
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	// Start cleanup goroutine
	go rl.cleanup()
	return rl
}

// cleanup removes old entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, times := range rl.requests {
			var valid []time.Time
			for _, t := range times {
				if now.Sub(t) < rl.window {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = valid
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	times := rl.requests[ip]

	// Filter to only requests within the window
	var valid []time.Time
	for _, t := range times {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.requests[ip] = valid
		return false
	}

	rl.requests[ip] = append(valid, now)
	return true
}

// RateLimitMiddleware limits request rate per IP
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			if !limiter.Allow(ip) {
				logger.Warn("rate limit exceeded", map[string]interface{}{
					"ip": ip,
					"path": r.URL.Path,
				})
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// InputSanitizer sanitizes user input to prevent injection attacks
func InputSanitizer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sanitize query parameters
		query := r.URL.Query()
		for key, values := range query {
			for i, value := range values {
				if validator.ContainsSQLKeywords(value) {
					logger.Warn("potential SQL injection detected", map[string]interface{}{
						"parameter": key,
						"value":     value,
					})
					http.Error(w, "Invalid input detected.", http.StatusBadRequest)
					return
				}
				query.Set(key, validator.SanitizeString(value))
				values[i] = validator.SanitizeString(value)
			}
		}
		r.URL.RawQuery = query.Encode()

		next.ServeHTTP(w, r)
	})
}

// CSRFProtection provides CSRF token validation
type CSRFProtection struct {
	tokens map[string]string
	mu     sync.RWMutex
}

// NewCSRFProtection creates a new CSRF protector
func NewCSRFProtection() *CSRFProtection {
	return &CSRFProtection{
		tokens: make(map[string]string),
	}
}

// GenerateToken generates a CSRF token for a session
func (c *CSRFProtection) GenerateToken(sessionID string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple token generation (in production, use crypto/rand)
	token := strconv.FormatInt(time.Now().UnixNano(), 36)
	c.tokens[sessionID] = token
	return token
}

// ValidateToken validates a CSRF token
func (c *CSRFProtection) ValidateToken(sessionID, token string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	expected, ok := c.tokens[sessionID]
	return ok && expected == token
}

// XSSProtection middleware prevents XSS attacks
func XSSProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for script tags in parameters
		scriptPattern := regexp.MustCompile(`(?i)<script|javascript:|on\w+=`)
		for key, values := range r.URL.Query() {
			for _, value := range values {
				if scriptPattern.MatchString(value) {
					logger.Warn("potential XSS attack detected", map[string]interface{}{
						"parameter": key,
						"value":     value,
					})
					http.Error(w, "Invalid input detected.", http.StatusBadRequest)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
