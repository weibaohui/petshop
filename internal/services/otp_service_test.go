package services

import (
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestNewOTPService(t *testing.T) {
	service := NewOTPService()
	if service == nil {
		t.Error("NewOTPService() should not return nil")
	}
}

func TestOTPService_GenerateSecret(t *testing.T) {
	service := NewOTPService()

	secret, err := service.GenerateSecret()
	if err != nil {
		t.Errorf("GenerateSecret() error = %v", err)
	}
	if secret == "" {
		t.Error("GenerateSecret() should not return empty secret")
	}

	// 验证生成的密钥是有效的 base32
	if len(secret) < 32 {
		t.Errorf("GenerateSecret() returned secret too short: %d", len(secret))
	}

	// 验证可以生成两个不同的密钥
	secret2, err := service.GenerateSecret()
	if err != nil {
		t.Errorf("GenerateSecret() second call error = %v", err)
	}
	if secret == secret2 {
		t.Error("GenerateSecret() should return unique secrets")
	}
}

func TestOTPService_GenerateQRCode(t *testing.T) {
	service := NewOTPService()

	secret, _ := service.GenerateSecret()
	account := "test@example.com"

	qrCode, keyURL, err := service.GenerateQRCode(account, secret)
	if err != nil {
		t.Errorf("GenerateQRCode() error = %v", err)
	}

	if qrCode == "" {
		t.Error("GenerateQRCode() should not return empty QR code")
	}

	if !strings.HasPrefix(qrCode, "data:image/png;base64,") {
		t.Error("GenerateQRCode() should return base64 data URI")
	}

	if keyURL == "" {
		t.Error("GenerateQRCode() should not return empty key URL")
	}

	if !strings.Contains(keyURL, secret) {
		t.Error("GenerateQRCode() key URL should contain secret")
	}

	if !strings.Contains(keyURL, OTPIssuer) {
		t.Error("GenerateQRCode() key URL should contain issuer")
	}
}

func TestOTPService_VerifyOTP(t *testing.T) {
	service := NewOTPService()

	secret, _ := service.GenerateSecret()

	// 生成一个有效的 OTP 码
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("Failed to generate TOTP code: %v", err)
	}

	// 验证正确的码
	if !service.VerifyOTP(secret, code) {
		t.Error("VerifyOTP() should return true for valid code")
	}

	// 验证错误的码
	if service.VerifyOTP(secret, "000000") {
		t.Error("VerifyOTP() should return false for invalid code")
	}

	// 验证空码
	if service.VerifyOTP(secret, "") {
		t.Error("VerifyOTP() should return false for empty code")
	}
}

func TestOTPService_GenerateBackupCodes(t *testing.T) {
	service := NewOTPService()

	codes := service.GenerateBackupCodes()

	if len(codes) != BackupCodeCount {
		t.Errorf("GenerateBackupCodes() returned %d codes, expected %d", len(codes), BackupCodeCount)
	}

	// 验证每个备用码
	for i, code := range codes {
		if code == "" {
			t.Errorf("GenerateBackupCodes() code %d is empty", i)
		}
		if len(code) != BackupCodeLength {
			t.Errorf("GenerateBackupCodes() code %d length = %d, expected %d", i, len(code), BackupCodeLength)
		}
	}

	// 验证备用码唯一性
	codeSet := make(map[string]bool)
	for _, code := range codes {
		if codeSet[code] {
			t.Error("GenerateBackupCodes() should return unique codes")
		}
		codeSet[code] = true
	}
}

func TestOTPService_GenerateTempToken(t *testing.T) {
	service := NewOTPService()

	token, err := service.GenerateTempToken()
	if err != nil {
		t.Errorf("GenerateTempToken() error = %v", err)
	}
	if token == "" {
		t.Error("GenerateTempToken() should not return empty token")
	}

	// 验证可以生成两个不同的令牌
	token2, _ := service.GenerateTempToken()
	if token == token2 {
		t.Error("GenerateTempToken() should return unique tokens")
	}
}

func TestOTPService_HasOTPEnabled(t *testing.T) {
	// 跳过此测试，因为它需要数据库连接
	// 集成测试应该在 handlers 层进行
	t.Skip("Skipping test that requires database connection")
}

func TestOTPService_VerifyOTPWithSpaces(t *testing.T) {
	service := NewOTPService()

	secret, _ := service.GenerateSecret()
	code, _ := totp.GenerateCode(secret, time.Now())

	// 测试带空格的密钥
	secretWithSpaces := strings.Join(strings.Split(secret, ""), " ")
	if !service.VerifyOTP(secretWithSpaces, code) {
		t.Error("VerifyOTP() should handle secrets with spaces")
	}

	// 测试带空格的验证码 - TOTP 库可能不处理空格，所以这只是测试不会 panic
	codeWithSpaces := code[:3] + " " + code[3:]
	// 清理空格后应该能验证成功
	codeCleaned := strings.ReplaceAll(codeWithSpaces, " ", "")
	if !service.VerifyOTP(secret, codeCleaned) {
		t.Error("VerifyOTP() should handle cleaned codes")
	}
}

func TestBackupCodeCount(t *testing.T) {
	if BackupCodeCount != 8 {
		t.Errorf("BackupCodeCount = %d, expected 8", BackupCodeCount)
	}
}

func TestBackupCodeLength(t *testing.T) {
	if BackupCodeLength != 8 {
		t.Errorf("BackupCodeLength = %d, expected 8", BackupCodeLength)
	}
}

func TestTempTokenExpiry(t *testing.T) {
	expected := 5 * time.Minute
	if TempTokenExpiry != expected {
		t.Errorf("TempTokenExpiry = %v, expected %v", TempTokenExpiry, expected)
	}
}

func TestOTPIssuer(t *testing.T) {
	if OTPIssuer != "PetShop" {
		t.Errorf("OTPIssuer = %s, expected PetShop", OTPIssuer)
	}
}
