package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"petshop/internal/db"
	"petshop/internal/logger"
	"petshop/internal/middleware"
	"petshop/internal/models"
)

func setupOTPTest(t *testing.T) (*OTPHandler, func()) {
	// 设置 JWT 密钥用于测试
	os.Setenv("JWT_SECRET_KEY", "test-jwt-secret-key-that-is-long-enough-for-hmac256-usage")
	os.Setenv("APP_ENV", "test")
	middleware.ResetJWTSecret()

	// 设置测试数据库
	tempFile, err := os.CreateTemp("", "otp_test_*.db")
	require.NoError(t, err)
	tempFile.Close()

	db.ResetForTesting()
	err = db.InitDB(tempFile.Name())
	require.NoError(t, err)

	// 初始化日志
	tempLogDir, _ := os.MkdirTemp("", "otp_log_*")
	logger.Init(tempLogDir)

	handler := NewOTPHandler()

	cleanup := func() {
		db.Close()
		os.Remove(tempFile.Name())
		os.RemoveAll(tempLogDir)
		db.ResetForTesting()
		os.Unsetenv("JWT_SECRET_KEY")
		os.Unsetenv("APP_ENV")
	}

	return handler, cleanup
}

func createTestUser(t *testing.T, username string) int64 {
	result, err := db.DB.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, "test_hash",
	)
	require.NoError(t, err)
	userID, _ := result.LastInsertId()
	return userID
}

// createAuthenticatedRequest 创建带有认证上下文的请求
func createAuthenticatedRequest(t *testing.T, method, url string, body []byte, userID int64) *http.Request {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, url, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// 添加用户 ID 到上下文
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	return req.WithContext(ctx)
}

func TestNewOTPHandler(t *testing.T) {
	handler := NewOTPHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.otpService)
}

func TestOTPHandler_SetupOTP_Unauthorized(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/otp/setup", nil)
	rr := httptest.NewRecorder()

	handler.SetupOTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestOTPHandler_SetupOTP_Success(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	// 创建测试用户
	userID := createTestUser(t, "testuser")

	req := createAuthenticatedRequest(t, http.MethodPost, "/api/otp/setup?username=testuser", nil, userID)
	rr := httptest.NewRecorder()

	handler.SetupOTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response models.OTPSetupResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Secret)
	assert.NotEmpty(t, response.QRCodeURL)
	assert.True(t, strings.HasPrefix(response.QRCodeURL, "data:image/png;base64,"))
	assert.Equal(t, 8, len(response.BackupCodes))
}

func TestOTPHandler_BindOTP_Unauthorized(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/otp/bind", nil)
	rr := httptest.NewRecorder()

	handler.BindOTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestOTPHandler_BindOTP_InvalidBody(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	userID := createTestUser(t, "testuser")

	req := createAuthenticatedRequest(t, http.MethodPost, "/api/otp/bind", []byte("invalid json"), userID)
	rr := httptest.NewRecorder()

	handler.BindOTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOTPHandler_BindOTP_EmptyFields(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	userID := createTestUser(t, "testuser")

	reqBody := models.OTPBindRequest{
		OTPCode: "",
		Secret:  "",
	}
	body, _ := json.Marshal(reqBody)

	req := createAuthenticatedRequest(t, http.MethodPost, "/api/otp/bind", body, userID)
	rr := httptest.NewRecorder()

	handler.BindOTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOTPHandler_GetOTPStatus_Unauthorized(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/otp/status", nil)
	rr := httptest.NewRecorder()

	handler.GetOTPStatus(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestOTPHandler_GetOTPStatus_NotEnabled(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	userID := createTestUser(t, "testuser")

	req := createAuthenticatedRequest(t, http.MethodGet, "/api/otp/status", nil, userID)
	rr := httptest.NewRecorder()

	handler.GetOTPStatus(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var status models.OTPStatus
	err := json.Unmarshal(rr.Body.Bytes(), &status)
	require.NoError(t, err)

	assert.False(t, status.Enabled)
}

func TestOTPHandler_UnbindOTP_Unauthorized(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/otp/unbind", nil)
	rr := httptest.NewRecorder()

	handler.UnbindOTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestOTPHandler_VerifyOTP_InvalidBody(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/otp/verify", strings.NewReader("invalid json"))
	rr := httptest.NewRecorder()

	handler.VerifyOTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOTPHandler_VerifyOTP_EmptyFields(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	reqBody := models.OTPVerifyRequest{
		UserID:  0,
		OTPCode: "",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/otp/verify", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.VerifyOTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOTPHandler_VerifyOTP_NotEnabled(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	userID := createTestUser(t, "testuser")

	reqBody := models.OTPVerifyRequest{
		UserID:  userID,
		OTPCode: "123456",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/otp/verify", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.VerifyOTP(rr, req)

	// OTP 未启用时返回 500 因为服务返回了错误
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestOTPHandler_FullFlow(t *testing.T) {
	handler, cleanup := setupOTPTest(t)
	defer cleanup()

	// 创建测试用户
	userID := createTestUser(t, "testuser")

	// 1. 设置 OTP
	req1 := createAuthenticatedRequest(t, http.MethodPost, "/api/otp/setup?username=testuser", nil, userID)
	rr1 := httptest.NewRecorder()
	handler.SetupOTP(rr1, req1)

	require.Equal(t, http.StatusOK, rr1.Code)

	var setupResp models.OTPSetupResponse
	err := json.Unmarshal(rr1.Body.Bytes(), &setupResp)
	require.NoError(t, err)

	// 2. 生成有效的 TOTP 码
	otpCode, err := totp.GenerateCode(setupResp.Secret, time.Now())
	require.NoError(t, err)

	// 3. 绑定 OTP
	bindReq := models.OTPBindRequest{
		Secret:  setupResp.Secret,
		OTPCode: otpCode,
	}
	bindBody, _ := json.Marshal(bindReq)

	req2 := createAuthenticatedRequest(t, http.MethodPost, "/api/otp/bind", bindBody, userID)
	rr2 := httptest.NewRecorder()
	handler.BindOTP(rr2, req2)

	assert.Equal(t, http.StatusOK, rr2.Code)

	// 4. 检查状态 - 应该已启用
	req3 := createAuthenticatedRequest(t, http.MethodGet, "/api/otp/status", nil, userID)
	rr3 := httptest.NewRecorder()
	handler.GetOTPStatus(rr3, req3)

	assert.Equal(t, http.StatusOK, rr3.Code)

	var status models.OTPStatus
	err = json.Unmarshal(rr3.Body.Bytes(), &status)
	require.NoError(t, err)
	assert.True(t, status.Enabled)

	// 5. 验证 OTP 码（模拟登录验证）
	newOTPCode, _ := totp.GenerateCode(setupResp.Secret, time.Now())

	verifyReq := models.OTPVerifyRequest{
		UserID:  userID,
		OTPCode: newOTPCode,
	}
	verifyBody, _ := json.Marshal(verifyReq)

	req4 := httptest.NewRequest(http.MethodPost, "/api/otp/verify", bytes.NewReader(verifyBody))
	rr4 := httptest.NewRecorder()
	handler.VerifyOTP(rr4, req4)

	assert.Equal(t, http.StatusOK, rr4.Code)

	var verifyResp map[string]interface{}
	err = json.Unmarshal(rr4.Body.Bytes(), &verifyResp)
	require.NoError(t, err)
	assert.NotEmpty(t, verifyResp["token"])

	// 6. 使用备用码验证
	backupReq := models.OTPVerifyRequest{
		UserID:   userID,
		OTPCode:  setupResp.BackupCodes[0],
		IsBackup: true,
	}
	backupBody, _ := json.Marshal(backupReq)

	req5 := httptest.NewRequest(http.MethodPost, "/api/otp/verify", bytes.NewReader(backupBody))
	rr5 := httptest.NewRecorder()
	handler.VerifyOTP(rr5, req5)

	assert.Equal(t, http.StatusOK, rr5.Code)

	// 7. 解绑 OTP
	unbindReq := models.OTPUnbindRequest{
		OTPCode: otpCode,
	}
	unbindBody, _ := json.Marshal(unbindReq)

	req6 := createAuthenticatedRequest(t, http.MethodPost, "/api/otp/unbind", unbindBody, userID)
	rr6 := httptest.NewRecorder()
	handler.UnbindOTP(rr6, req6)

	assert.Equal(t, http.StatusOK, rr6.Code)

	// 8. 再次检查状态 - 应该已禁用
	req7 := createAuthenticatedRequest(t, http.MethodGet, "/api/otp/status", nil, userID)
	rr7 := httptest.NewRecorder()
	handler.GetOTPStatus(rr7, req7)

	assert.Equal(t, http.StatusOK, rr7.Code)

	var finalStatus models.OTPStatus
	err = json.Unmarshal(rr7.Body.Bytes(), &finalStatus)
	require.NoError(t, err)
	assert.False(t, finalStatus.Enabled)
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	data := map[string]string{"message": "test"}

	writeJSON(rr, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var result map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "test", result["message"])
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()

	writeError(rr, http.StatusBadRequest, "test error")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var result map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "test error", result["error"])
}
