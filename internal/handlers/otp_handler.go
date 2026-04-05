package handlers

import (
	"encoding/json"
	"net/http"

	"petshop/internal/logger"
	"petshop/internal/middleware"
	"petshop/internal/models"
	"petshop/internal/services"
)

// OTPHandler OTP处理器
type OTPHandler struct {
	otpService *services.OTPService
}

// NewOTPHandler 创建OTP处理器
func NewOTPHandler() *OTPHandler {
	return &OTPHandler{
		otpService: services.NewOTPService(),
	}
}

// SetupOTP 初始化OTP设置（生成密钥和二维码）
// POST /api/otp/setup
func (h *OTPHandler) SetupOTP(w http.ResponseWriter, r *http.Request) {
	// 获取用户ID（从上下文）
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		logger.Warn("unauthorized access to OTP setup", nil)
		writeError(w, http.StatusUnauthorized, "未授权访问")
		return
	}

	// 获取用户名（实际项目中从数据库获取）
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "user"
	}

	// 设置OTP
	secret, qrCode, backupCodes, err := h.otpService.SetupOTP(userID, username)
	if err != nil {
		logger.Error("failed to setup OTP", map[string]interface{}{
			"userId": userID,
			"error":  err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "设置OTP失败")
		return
	}

	logger.Info("OTP setup initiated", map[string]interface{}{
		"userId": userID,
	})

	// 返回响应
	response := models.OTPSetupResponse{
		Secret:      secret,
		QRCodeURL:   qrCode,
		BackupCodes: backupCodes,
	}

	writeJSON(w, http.StatusOK, response)
}

// BindOTP 绑定OTP（验证并启用）
// POST /api/otp/bind
func (h *OTPHandler) BindOTP(w http.ResponseWriter, r *http.Request) {
	// 获取用户ID
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		logger.Warn("unauthorized access to OTP bind", nil)
		writeError(w, http.StatusUnauthorized, "未授权访问")
		return
	}

	// 解析请求
	var req models.OTPBindRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid request body for OTP bind", map[string]interface{}{
			"error": err.Error(),
		})
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	// 验证参数
	if req.OTPCode == "" || req.Secret == "" {
		writeError(w, http.StatusBadRequest, "OTP验证码和密钥不能为空")
		return
	}

	// 绑定OTP
	if err := h.otpService.BindOTP(userID, req.Secret, req.OTPCode); err != nil {
		logger.Warn("failed to bind OTP", map[string]interface{}{
			"userId": userID,
			"error":  err.Error(),
		})
		writeError(w, http.StatusBadRequest, "OTP绑定失败："+err.Error())
		return
	}

	logger.Info("OTP bound successfully", map[string]interface{}{
		"userId": userID,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "OTP绑定成功",
	})
}

// UnbindOTP 解绑OTP
// POST /api/otp/unbind
func (h *OTPHandler) UnbindOTP(w http.ResponseWriter, r *http.Request) {
	// 获取用户ID
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		logger.Warn("unauthorized access to OTP unbind", nil)
		writeError(w, http.StatusUnauthorized, "未授权访问")
		return
	}

	// 解析请求
	var req models.OTPUnbindRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid request body for OTP unbind", map[string]interface{}{
			"error": err.Error(),
		})
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	// 验证参数
	if req.OTPCode == "" {
		writeError(w, http.StatusBadRequest, "OTP验证码不能为空")
		return
	}

	// 解绑OTP
	if err := h.otpService.UnbindOTP(userID, req.OTPCode); err != nil {
		logger.Warn("failed to unbind OTP", map[string]interface{}{
			"userId": userID,
			"error":  err.Error(),
		})
		writeError(w, http.StatusBadRequest, "OTP解绑失败："+err.Error())
		return
	}

	logger.Info("OTP unbound successfully", map[string]interface{}{
		"userId": userID,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "OTP解绑成功",
	})
}

// GetOTPStatus 获取用户OTP状态
// GET /api/otp/status
func (h *OTPHandler) GetOTPStatus(w http.ResponseWriter, r *http.Request) {
	// 获取用户ID
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		logger.Warn("unauthorized access to OTP status", nil)
		writeError(w, http.StatusUnauthorized, "未授权访问")
		return
	}

	// 获取状态
	enabled, enabledAt, err := h.otpService.GetOTPStatus(userID)
	if err != nil {
		logger.Error("failed to get OTP status", map[string]interface{}{
			"userId": userID,
			"error":  err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "获取OTP状态失败")
		return
	}

	status := models.OTPStatus{
		Enabled:   enabled,
		EnabledAt: enabledAt,
	}

	writeJSON(w, http.StatusOK, status)
}

// VerifyOTP 验证OTP（用于登录验证）
// POST /api/otp/verify
func (h *OTPHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	// 解析请求
	var req models.OTPVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid request body for OTP verify", map[string]interface{}{
			"error": err.Error(),
		})
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	// 验证参数
	if req.UserID <= 0 || req.OTPCode == "" {
		writeError(w, http.StatusBadRequest, "用户ID和OTP验证码不能为空")
		return
	}

	// 验证OTP
	valid, err := h.otpService.VerifyLoginOTP(req.UserID, req.OTPCode, req.IsBackup)
	if err != nil {
		logger.Warn("OTP verification error", map[string]interface{}{
			"userId": req.UserID,
			"error":  err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "OTP验证失败")
		return
	}

	if !valid {
		logger.Warn("invalid OTP code", map[string]interface{}{
			"userId":  req.UserID,
			"isBackup": req.IsBackup,
		})
		writeError(w, http.StatusUnauthorized, "OTP验证码错误")
		return
	}

	// 验证成功，生成JWT令牌
	token, err := middleware.GenerateToken(req.UserID)
	if err != nil {
		logger.Error("failed to generate token after OTP verification", map[string]interface{}{
			"userId": req.UserID,
			"error":  err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "生成令牌失败")
		return
	}

	logger.Info("OTP verified successfully", map[string]interface{}{
		"userId":   req.UserID,
		"isBackup": req.IsBackup,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "验证成功",
		"token":   token,
	})
}

// writeJSON 写入JSON响应
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError 写入错误响应
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": message,
	})
}
