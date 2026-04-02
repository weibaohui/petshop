package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// testJWTSecret is a dedicated secret key for test isolation
const testJWTSecret = "test-only-secret-key-do-not-use-in-production-32bytes"

func TestMain(m *testing.M) {
	os.Setenv("APP_ENV", "development")
	// Set isolated test JWT secret
	SetJWTSecret(testJWTSecret)
	code := m.Run()
	// Clean up after tests
	ResetJWTSecret()
	os.Exit(code)
}

func TestAuthMiddleware_NoAuthHeader(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	if !strings.Contains(w.Body.String(), "missing authorization header") {
		t.Errorf("expected 'missing authorization header' in body, got %s", w.Body.String())
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	token, err := GenerateToken(123)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	var capturedUserID int64
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		if ok {
			capturedUserID = userID
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if capturedUserID != 123 {
		t.Errorf("expected userID %d, got %d", 123, capturedUserID)
	}
}

func TestAuthMiddleware_RawToken(t *testing.T) {
	token, err := GenerateToken(456)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	var capturedUserID int64
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		if ok {
			capturedUserID = userID
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if capturedUserID != 456 {
		t.Errorf("expected userID %d, got %d", 456, capturedUserID)
	}
}

// TestAuthMiddleware_InvalidUserID 测试无效userID分支的覆盖
// 创建一个语法有效但包含无效userID的JWT，确保触发"invalid userID"分支
func TestAuthMiddleware_InvalidUserID(t *testing.T) {
	testCases := []struct {
		name   string
		userID int64
	}{
		{
			name:   "zero_userID",
			userID: 0,
		},
		{
			name:   "negative_userID",
			userID: -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 生成一个语法有效的JWT，但claims中包含无效userID
			token, err := GenerateToken(tc.userID)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}

			handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			// 设置Authorization为"Bearer <token>"格式
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// 确保AuthMiddleware执行"invalid userID"分支并返回http.StatusUnauthorized
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
			}
		})
	}
}

func TestGetUserID_NotPresent(t *testing.T) {
	ctx := context.Background()
	_, ok := GetUserID(ctx)
	if ok {
		t.Error("expected GetUserID to return false for missing key")
	}
}

func TestGetUserID_Present(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		if !ok || userID != 789 {
			t.Errorf("expected userID %d and ok=true, got %d and ok=%v", 789, userID, ok)
		}
		w.WriteHeader(http.StatusOK)
	}))

	token, _ := GenerateToken(789)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken(100)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}
