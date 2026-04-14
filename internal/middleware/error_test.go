package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"petshop/internal/logger"
)

func init() {
	_ = logger.Init("")
}

func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	handler := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRecoveryMiddleware_WithPanic(t *testing.T) {
	handler := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Code != http.StatusInternalServerError {
		t.Errorf("expected code %d, got %d", http.StatusInternalServerError, resp.Code)
	}
	if resp.TraceID == "" {
		t.Error("expected trace ID to be set")
	}
	if resp.Message == "" {
		t.Error("expected message to be set")
	}
}

func TestRequestLoggerMiddleware(t *testing.T) {
	handler := RequestLoggerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("User-Agent", "test-agent")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestResponseWriterWrapper_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	wrapper.WriteHeader(http.StatusCreated)

	if wrapper.statusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, wrapper.statusCode)
	}
	if w.Code != http.StatusCreated {
		t.Errorf("expected underlying writer status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestGenerateTraceID(t *testing.T) {
	traceID := generateTraceID()
	if traceID == "" {
		t.Error("expected non-empty trace ID")
	}
	if len(traceID) < 10 {
		t.Errorf("expected trace ID to have reasonable length, got %s", traceID)
	}
}

func TestErrorResponse_JSON(t *testing.T) {
	resp := ErrorResponse{
		Code:    400,
		Message: "test error",
		TraceID: "trace123",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ErrorResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Code != 400 {
		t.Errorf("expected code 400, got %d", decoded.Code)
	}
	if decoded.Message != "test error" {
		t.Errorf("expected message 'test error', got %s", decoded.Message)
	}
	if decoded.TraceID != "trace123" {
		t.Errorf("expected trace ID 'trace123', got %s", decoded.TraceID)
	}
}
