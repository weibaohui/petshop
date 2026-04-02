package middleware

import (
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"petshop/internal/logger"
	"petshop/internal/validator"
)

// trustedProxies contains the list of trusted proxy IPs/CIDRs.
// Only requests from these proxies will have X-Forwarded-For and X-Real-IP headers trusted.
var trustedProxies []net.IPNet

// SetTrustedProxies configures the list of trusted proxy CIDR ranges.
// This should be called during server initialization with the proxy network ranges.
func SetTrustedProxies(cidrs []string) {
	trustedProxies = nil
	for _, cidr := range cidrs {
		if _, ipNet, err := net.ParseCIDR(cidr); err == nil {
			trustedProxies = append(trustedProxies, *ipNet)
		}
	}
}

// isTrustedProxy checks if the given IP is from a trusted proxy.
func isTrustedProxy(ip string) bool {
	if ip == "" {
		return false
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	for _, ipNet := range trustedProxies {
		if ipNet.Contains(parsedIP) {
			return true
		}
	}
	return len(trustedProxies) == 0 // If no proxies configured, be permissive (backward compatible)
}

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

// getClientIP extracts the client IP from request.
// It only trusts X-Forwarded-For and X-Real-IP headers when the request
// comes from a trusted proxy. Otherwise, it falls back to RemoteAddr.
func getClientIP(r *http.Request) string {
	// Extract the proxy IP from RemoteAddr
	proxyIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		proxyIP = r.RemoteAddr
	}

	// Only trust headers from trusted proxies
	if !isTrustedProxy(proxyIP) {
		return proxyIP
	}

	// Check X-Forwarded-For header (trusted proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs: client, proxy1, proxy2, ...
		// We only take the first IP (original client)
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
		return xff
	}

	// Check X-Real-IP header (trusted proxy)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	return proxyIP
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
