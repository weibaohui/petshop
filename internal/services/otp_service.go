package services

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"

	"petshop/internal/db"
)

const (
	// OTP 发行者名称
	OTPIssuer = "PetShop"
	// 备用验证码数量
	BackupCodeCount = 8
	// 备用验证码长度
	BackupCodeLength = 8
	// 临时令牌过期时间
	TempTokenExpiry = 5 * time.Minute
)

// OTPService OTP服务
type OTPService struct{}

// NewOTPService 创建OTP服务实例
func NewOTPService() *OTPService {
	return &OTPService{}
}

// GenerateSecret 生成OTP密钥
func (s *OTPService) GenerateSecret() (string, error) {
	// 生成32字节随机密钥
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		return "", err
	}
	// 使用base32编码
	return base32.StdEncoding.EncodeToString(secret), nil
}

// GenerateQRCode 生成OTP二维码
func (s *OTPService) GenerateQRCode(account, secret string) (string, string, error) {
	// 创建TOTP密钥
	key, err := otp.NewKeyFromURL(fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s",
		OTPIssuer, account, secret, OTPIssuer,
	))
	if err != nil {
		return "", "", err
	}

	// 生成二维码
	png, err := qrcode.Encode(key.URL(), qrcode.Medium, 256)
	if err != nil {
		return "", "", err
	}

	// Base64编码二维码图片
	qrCodeBase64 := base64.StdEncoding.EncodeToString(png)
	dataURI := fmt.Sprintf("data:image/png;base64,%s", qrCodeBase64)

	return dataURI, key.URL(), nil
}

// VerifyOTP 验证OTP验证码
func (s *OTPService) VerifyOTP(secret, code string) bool {
	// 清理密钥中的空格
	secret = strings.ToUpper(strings.ReplaceAll(secret, " ", ""))

	// 验证TOTP码（允许30秒时间窗口，前后各1个时间步）
	valid := totp.Validate(code, secret)
	return valid
}

// GenerateBackupCodes 生成备用验证码
func (s *OTPService) GenerateBackupCodes() []string {
	codes := make([]string, BackupCodeCount)
	for i := 0; i < BackupCodeCount; i++ {
		// 生成随机字节
		bytes := make([]byte, BackupCodeLength/2)
		rand.Read(bytes)
		// 使用十六进制编码，确保可读性
		codes[i] = strings.ToUpper(fmt.Sprintf("%x", bytes))
	}
	return codes
}

// VerifyBackupCode 验证备用验证码
func (s *OTPService) VerifyBackupCode(userID int64, code string) (bool, error) {
	// 获取用户OTP配置
	userOTP, err := db.GetUserOTP(userID)
	if err != nil {
		return false, err
	}
	if userOTP == nil {
		return false, nil
	}

	// 转换为全大写进行比较
	code = strings.ToUpper(strings.ReplaceAll(code, " ", ""))

	// 查找并移除已使用的备用码
	for i, backupCode := range userOTP.BackupCodes {
		if strings.EqualFold(backupCode, code) {
			// 移除已使用的备用码
			userOTP.BackupCodes = append(userOTP.BackupCodes[:i], userOTP.BackupCodes[i+1:]...)
			// 更新数据库
			if err := db.UpdateBackupCodes(userID, userOTP.BackupCodes); err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}

// GenerateTempToken 生成临时登录令牌
func (s *OTPService) GenerateTempToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(bytes), nil
}

// SetupOTP 设置OTP（生成密钥和二维码）
func (s *OTPService) SetupOTP(userID int64, username string) (secret string, qrCode string, backupCodes []string, err error) {
	// 生成密钥
	secret, err = s.GenerateSecret()
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	// 生成二维码
	qrCode, _, err = s.GenerateQRCode(username, secret)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	// 生成备用验证码
	backupCodes = s.GenerateBackupCodes()

	// 保存到数据库（尚未启用）
	if err := db.CreateUserOTP(userID, secret, backupCodes); err != nil {
		return "", "", nil, fmt.Errorf("failed to save OTP config: %w", err)
	}

	return secret, qrCode, backupCodes, nil
}

// BindOTP 绑定OTP（验证并启用）
func (s *OTPService) BindOTP(userID int64, secret, code string) error {
	// 验证OTP码
	if !s.VerifyOTP(secret, code) {
		return fmt.Errorf("invalid OTP code")
	}

	// 启用OTP
	if err := db.EnableUserOTP(userID); err != nil {
		return fmt.Errorf("failed to enable OTP: %w", err)
	}

	return nil
}

// UnbindOTP 解绑OTP
func (s *OTPService) UnbindOTP(userID int64, code string) error {
	// 获取用户OTP配置
	userOTP, err := db.GetUserOTP(userID)
	if err != nil {
		return fmt.Errorf("failed to get OTP config: %w", err)
	}
	if userOTP == nil || !userOTP.OTPEnabled {
		return fmt.Errorf("OTP not enabled")
	}

	// 验证OTP码
	if !s.VerifyOTP(userOTP.OTPSecret, code) {
		// 尝试备用码
		valid, err := s.VerifyBackupCode(userID, code)
		if err != nil || !valid {
			return fmt.Errorf("invalid OTP code")
		}
	}

	// 禁用OTP
	if err := db.DisableUserOTP(userID); err != nil {
		return fmt.Errorf("failed to disable OTP: %w", err)
	}

	return nil
}

// VerifyLoginOTP 验证登录OTP
func (s *OTPService) VerifyLoginOTP(userID int64, code string, isBackup bool) (bool, error) {
	if isBackup {
		// 验证备用码
		return s.VerifyBackupCode(userID, code)
	}

	// 获取用户OTP密钥
	secret, err := db.GetUserOTPSecret(userID)
	if err != nil {
		return false, err
	}
	if secret == "" {
		return false, fmt.Errorf("OTP not enabled")
	}

	// 验证TOTP码
	return s.VerifyOTP(secret, code), nil
}

// GetOTPStatus 获取用户OTP状态
func (s *OTPService) GetOTPStatus(userID int64) (bool, *time.Time, error) {
	userOTP, err := db.GetUserOTP(userID)
	if err != nil {
		return false, nil, err
	}
	if userOTP == nil {
		return false, nil, nil
	}
	return userOTP.OTPEnabled, userOTP.OTPEnabledAt, nil
}

// HasOTPEnabled 检查用户是否启用了OTP
func (s *OTPService) HasOTPEnabled(userID int64) (bool, error) {
	return db.IsOTPEnabled(userID)
}

// CreateLoginSession 创建OTP登录会话
func (s *OTPService) CreateLoginSession(userID int64) (string, error) {
	tempToken, err := s.GenerateTempToken()
	if err != nil {
		return "", err
	}

	db.CreateOTPSession(userID, tempToken, TempTokenExpiry)
	return tempToken, nil
}

// ValidateLoginSession 验证OTP登录会话
func (s *OTPService) ValidateLoginSession(tempToken string) (int64, bool) {
	session, ok := db.GetOTPSession(tempToken)
	if !ok {
		return 0, false
	}
	return session.UserID, true
}

// ClearLoginSession 清除OTP登录会话
func (s *OTPService) ClearLoginSession(tempToken string) {
	db.DeleteOTPSession(tempToken)
}
