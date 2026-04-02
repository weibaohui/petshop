package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("X-XSS-Protection") != "1; mode=block" {
		t.Error("expected X-XSS-Protection header")
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected X-Content-Type-Options header")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("expected X-Frame-Options header")
	}
	if w.Header().Get("Content-Security-Policy") != "default-src 'self'" {
		t.Error("expected Content-Security-Policy header")
	}
	if w.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Error("expected Referrer-Policy header")
	}
}

func TestNewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(10, time.Minute)
	if limiter == nil {
		t.Fatal("expected non-nil rate limiter")
	}
	if limiter.limit != 10 {
		t.Errorf("expected limit %d, got %d", 10, limiter.limit)
	}
	if limiter.window != time.Minute {
		t.Errorf("expected window %v, got %v", time.Minute, limiter.window)
	}
}

func TestRateLimiter_Allow_UnderLimit(t *testing.T) {
	limiter := NewRateLimiter(5, time.Minute)
	limiter.mu.Lock()
	limiter.requests = make(map[string][]time.Time)
	limiter.mu.Unlock()

	for i := 0; i < 5; i++ {
		if !limiter.Allow("192.168.1.1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiter_Allow_OverLimit(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)
	limiter.mu.Lock()
	limiter.requests = make(map[string][]time.Time)
	limiter.mu.Unlock()

	for i := 0; i < 3; i++ {
		limiter.Allow("192.168.1.2")
	}

	if limiter.Allow("192.168.1.2") {
		t.Error("request over limit should be denied")
	}
}

func TestRateLimiter_Allow_DifferentIPs(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)
	limiter.mu.Lock()
	limiter.requests = make(map[string][]time.Time)
	limiter.mu.Unlock()

	if !limiter.Allow("192.168.1.1") {
		t.Error("first request should be allowed")
	}
	if !limiter.Allow("192.168.1.2") {
		t.Error("request from different IP should be allowed")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	limiter.mu.Lock()
	limiter.requests = make(map[string][]time.Time)
	limiter.mu.Unlock()

	middleware := RateLimitMiddleware(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("first request: expected %d, got %d", http.StatusOK, w.Code)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("second request: expected %d, got %d", http.StatusOK, w.Code)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("third request: expected %d, got %d", http.StatusTooManyRequests, w.Code)
	}
}

func TestRateLimitMiddleware_WithXForwardedFor(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)
	limiter.mu.Lock()
	limiter.requests = make(map[string][]time.Time)
	limiter.mu.Unlock()

	middleware := RateLimitMiddleware(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("first request: expected %d, got %d", http.StatusOK, w.Code)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("second request: expected %d, got %d", http.StatusTooManyRequests, w.Code)
	}
}

func TestGetClientIP_UntrustedProxy(t *testing.T) {
	// Without trusted proxies set, X-Forwarded-For should be ignored
	// as no proxies are configured (backward compatible - permissive mode)
	SetTrustedProxies(nil)

	tests := []struct {
		name      string
		xff       string
		xri       string
		remoteAddr string
		expected  string
	}{
		{"X-Forwarded-For from untrusted", "10.0.0.1", "", "192.168.1.1:1234", "10.0.0.1"},
		{"X-Real-IP from untrusted", "", "10.0.0.2", "192.168.1.1:1234", "10.0.0.2"},
		{"RemoteAddr only", "", "", "192.168.1.1:1234", "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			req.RemoteAddr = tt.remoteAddr

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, ip)
			}
		})
	}
}

func TestGetClientIP_TrustedProxy(t *testing.T) {
	// Set trusted proxy to 192.168.1.0/24
	SetTrustedProxies([]string{"192.168.1.0/24"})

	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
	}{
		{"X-Forwarded-For from trusted proxy", "10.0.0.1", "", "192.168.1.100:1234", "10.0.0.1"},
		{"X-Forwarded-For with multiple IPs", "10.0.0.1, 10.0.0.2, 10.0.0.3", "", "192.168.1.100:1234", "10.0.0.1"},
		{"X-Real-IP from trusted proxy", "", "10.0.0.2", "192.168.1.100:1234", "10.0.0.2"},
		{"No headers, trusted proxy", "", "", "192.168.1.100:1234", "192.168.1.100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			req.RemoteAddr = tt.remoteAddr

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, ip)
			}
		})
	}

	// Reset to no trusted proxies
	SetTrustedProxies(nil)
}

func TestGetClientIP_UntrustedProxyRejectsXFF(t *testing.T) {
	// Set trusted proxy to 10.0.0.0/8, so 192.168.1.x is NOT trusted
	SetTrustedProxies([]string{"10.0.0.0/8"})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.RemoteAddr = "192.168.1.1:1234" // Not in 10.0.0.0/8

	ip := getClientIP(req)
	// Should return proxy IP (from RemoteAddr) not the spoofed XFF
	if ip != "192.168.1.1" {
		t.Errorf("expected untrusted proxy to return RemoteAddr IP '192.168.1.1', got '%s'", ip)
	}

	// Reset to no trusted proxies
	SetTrustedProxies(nil)
}

func TestNewCSRFProtection(t *testing.T) {
	csrf := NewCSRFProtection()
	if csrf == nil {
		t.Fatal("expected non-nil CSRF protection")
	}
	if csrf.tokens == nil {
		t.Error("expected tokens map to be initialized")
	}
}

func TestCSRFProtection_GenerateToken(t *testing.T) {
	csrf := NewCSRFProtection()
	token := csrf.GenerateToken("session123")
	if token == "" {
		t.Error("expected non-empty token")
	}

	csrf.mu.RLock()
	defer csrf.mu.RUnlock()
	if _, ok := csrf.tokens["session123"]; !ok {
		t.Error("expected token to be stored")
	}
}

func TestCSRFProtection_ValidateToken_Valid(t *testing.T) {
	csrf := NewCSRFProtection()
	token := csrf.GenerateToken("session123")

	if !csrf.ValidateToken("session123", token) {
		t.Error("expected valid token to pass validation")
	}
}

func TestCSRFProtection_ValidateToken_Invalid(t *testing.T) {
	csrf := NewCSRFProtection()
	csrf.GenerateToken("session123")

	if csrf.ValidateToken("session123", "wrong-token") {
		t.Error("expected invalid token to fail validation")
	}
}

func TestCSRFProtection_ValidateToken_WrongSession(t *testing.T) {
	csrf := NewCSRFProtection()
	token := csrf.GenerateToken("session123")

	if csrf.ValidateToken("session456", token) {
		t.Error("expected token from different session to fail validation")
	}
}

func TestXSSProtection(t *testing.T) {
	handler := XSSProtection(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name        string
		query       string
		expectCode  int
	}{
		{"Normal query", "/test?name=test", http.StatusOK},
		{"Script tag", "/test?name=<script>alert(1)</script>", http.StatusBadRequest},
		{"Javascript protocol", "/test?name=javascript:alert(1)", http.StatusBadRequest},
		{"onload handler", "/test?name=test%20onload%3Dalert(1)", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.query, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tt.expectCode {
				t.Errorf("expected %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}
