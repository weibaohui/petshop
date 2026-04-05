package models

import "time"

// UserOTP 用户OTP二次验证配置
type UserOTP struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"userId"`
	OTPSecret     string    `json:"-"` // OTP密钥，不返回给前端
	OTPEnabled    bool      `json:"otpEnabled"`
	OTPEnabledAt  *time.Time `json:"otpEnabledAt,omitempty"`
	BackupCodes   []string  `json:"-"` // 备用验证码，不返回给前端
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// OTPSetupResponse OTP绑定响应
type OTPSetupResponse struct {
	Secret      string `json:"secret"`
	QRCodeURL   string `json:"qrCodeUrl"`
	BackupCodes []string `json:"backupCodes"`
}

// OTPVerifyRequest OTP验证请求
type OTPVerifyRequest struct {
	UserID   int64  `json:"userId"`
	OTPCode  string `json:"otpCode"`
	IsBackup bool   `json:"isBackup,omitempty"` // 是否使用备用码
}

// OTPBindRequest OTP绑定请求
type OTPBindRequest struct {
	OTPCode string `json:"otpCode"`
	Secret  string `json:"secret"`
}

// OTPUnbindRequest OTP解绑请求
type OTPUnbindRequest struct {
	OTPCode string `json:"otpCode"`
}

// OTPStatus OTP状态响应
type OTPStatus struct {
	Enabled    bool       `json:"enabled"`
	EnabledAt  *time.Time `json:"enabledAt,omitempty"`
}
