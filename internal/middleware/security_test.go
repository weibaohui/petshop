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
	// Set a specific trusted proxy range that does NOT include our test proxy
	// This ensures we're in "strict mode" where untrusted proxies are rejected
	SetTrustedProxies([]string{"10.0.0.0/8"})

	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
		comment    string
	}{
		{
			name:       "X-Forwarded-For from untrusted proxy",
			xff:        "10.0.0.1",
			xri:        "",
			remoteAddr: "192.168.1.1:1234",
			expected:   "192.168.1.1",
			comment:    "Should fallback to RemoteAddr when proxy is not trusted",
		},
		{
			name:       "X-Real-IP from untrusted proxy",
			xff:        "",
			xri:        "10.0.0.2",
			remoteAddr: "192.168.1.1:1234",
			expected:   "192.168.1.1",
			comment:    "Should ignore X-Real-IP when proxy is not trusted",
		},
		{
			name:       "Both headers from untrusted proxy",
			xff:        "10.0.0.1",
			xri:        "10.0.0.2",
			remoteAddr: "192.168.1.1:1234",
			expected:   "192.168.1.1",
			comment:    "Should ignore both headers when proxy is not trusted",
		},
		{
			name:       "Multi-hop XFF from untrusted proxy",
			xff:        "10.0.0.1, 10.0.0.2, 10.0.0.3",
			xri:        "",
			remoteAddr: "192.168.1.1:1234",
			expected:   "192.168.1.1",
			comment:    "Should ignore multi-hop XFF when proxy is not trusted",
		},
		{
			name:       "RemoteAddr only",
			xff:        "",
			xri:        "",
			remoteAddr: "192.168.1.1:1234",
			expected:   "192.168.1.1",
			comment:    "Should return RemoteAddr when no headers present",
		},
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
				t.Errorf("%s: expected %s, got %s", tt.comment, tt.expected, ip)
			}
		})
	}

	// Reset to no trusted proxies
	SetTrustedProxies(nil)
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
		comment    string
	}{
		{
			name:       "X-Forwarded-For from trusted proxy",
			xff:        "10.0.0.1",
			xri:        "",
			remoteAddr: "192.168.1.100:1234",
			expected:   "10.0.0.1",
			comment:    "Should extract client IP from XFF when proxy is trusted",
		},
		{
			name:       "X-Forwarded-For with multiple IPs - selects leftmost",
			xff:        "10.0.0.1, 10.0.0.2, 10.0.0.3",
			xri:        "",
			remoteAddr: "192.168.1.100:1234",
			expected:   "10.0.0.1",
			comment:    "Should select the leftmost (original client) IP from multi-hop XFF",
		},
		{
			name:       "X-Forwarded-For with many hops",
			xff:        "203.0.113.1, 198.51.100.2, 192.168.50.3, 10.0.0.4",
			xri:        "",
			remoteAddr: "192.168.1.100:1234",
			expected:   "203.0.113.1",
			comment:    "Should select original client IP from complex multi-hop chain",
		},
		{
			name:       "X-Forwarded-For with spaces",
			xff:        "  10.0.0.1  ,  10.0.0.2  ,  10.0.0.3  ",
			xri:        "",
			remoteAddr: "192.168.1.100:1234",
			expected:   "10.0.0.1",
			comment:    "Should handle XFF with extra whitespace",
		},
		{
			name:       "X-Real-IP from trusted proxy when XFF not present",
			xff:        "",
			xri:        "10.0.0.2",
			remoteAddr: "192.168.1.100:1234",
			expected:   "10.0.0.2",
			comment:    "Should use X-Real-IP when XFF is not present and proxy is trusted",
		},
		{
			name:       "XFF takes precedence over X-Real-IP",
			xff:        "10.0.0.1",
			xri:        "10.0.0.2",
			remoteAddr: "192.168.1.100:1234",
			expected:   "10.0.0.1",
			comment:    "X-Forwarded-For should take precedence over X-Real-IP",
		},
		{
			name:       "No headers from trusted proxy",
			xff:        "",
			xri:        "",
			remoteAddr: "192.168.1.100:1234",
			expected:   "192.168.1.100",
			comment:    "Should return proxy IP when no forwarding headers present",
		},
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
				t.Errorf("%s: expected %s, got %s", tt.comment, tt.expected, ip)
			}
		})
	}

	// Reset to no trusted proxies
	SetTrustedProxies(nil)
}

func TestGetClientIP_ProxyNotInTrustedList(t *testing.T) {
	// Set trusted proxies to specific ranges
	SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12"})

	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
		comment    string
	}{
		{
			name:       "Request from untrusted public IP with XFF",
			xff:        "203.0.113.1",
			xri:        "",
			remoteAddr: "203.0.113.50:1234",
			expected:   "203.0.113.50",
			comment:    "Public proxy not in trusted list should not be trusted",
		},
		{
			name:       "Request from untrusted 192.168.x with XFF",
			xff:        "10.0.0.1",
			xri:        "10.0.0.2",
			remoteAddr: "192.168.1.1:1234",
			expected:   "192.168.1.1",
			comment:    "192.168.x.x not in trusted list (10.0.0.0/8 or 172.16.0.0/12)",
		},
		{
			name:       "Request from trusted 10.x.x.x with XFF",
			xff:        "203.0.113.1, 198.51.100.1",
			xri:        "",
			remoteAddr: "10.0.0.5:1234",
			expected:   "203.0.113.1",
			comment:    "10.x.x.x is in trusted list, should extract leftmost XFF IP",
		},
		{
			name:       "Request from trusted 172.16.x.x with XFF",
			xff:        "192.0.2.1",
			xri:        "",
			remoteAddr: "172.16.5.10:1234",
			expected:   "192.0.2.1",
			comment:    "172.16.x.x is in trusted list, should trust XFF",
		},
		{
			name:       "Spoofed XFF from untrusted proxy",
			xff:        "1.2.3.4, 5.6.7.8, 9.10.11.12",
			xri:        "99.99.99.99",
			remoteAddr: "203.0.113.99:1234",
			expected:   "203.0.113.99",
			comment:    "Spoofed XFF from untrusted proxy should be ignored",
		},
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
				t.Errorf("%s: expected %s, got %s", tt.comment, tt.expected, ip)
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
		name       string
		query      string
		expectCode int
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
