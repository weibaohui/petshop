package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"petshop/internal/models"
)

// InitOTPTable 创建OTP表
func InitOTPTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS user_otp (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL UNIQUE,
		otp_secret TEXT NOT NULL,
		otp_enabled BOOLEAN DEFAULT 0,
		otp_enabled_at TIMESTAMP,
		backup_codes TEXT, -- JSON array
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
	CREATE INDEX IF NOT EXISTS idx_user_otp_user_id ON user_otp(user_id);
	`
	_, err := DB.Exec(query)
	return err
}

// CreateUserOTP 创建用户OTP记录
func CreateUserOTP(userID int64, secret string, backupCodes []string) error {
	backupCodesJSON, err := json.Marshal(backupCodes)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO user_otp (user_id, otp_secret, backup_codes)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			otp_secret = excluded.otp_secret,
			backup_codes = excluded.backup_codes,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err = DB.Exec(query, userID, secret, string(backupCodesJSON))
	return err
}

// EnableUserOTP 启用用户OTP
func EnableUserOTP(userID int64) error {
	query := `
		UPDATE user_otp SET
			otp_enabled = 1,
			otp_enabled_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`
	result, err := DB.Exec(query, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("OTP not set up for user")
	}
	return nil
}

// DisableUserOTP 禁用用户OTP
func DisableUserOTP(userID int64) error {
	query := `
		UPDATE user_otp SET
			otp_enabled = 0,
			otp_enabled_at = NULL,
			otp_secret = '',
			backup_codes = '[]',
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`
	_, err := DB.Exec(query, userID)
	return err
}

// GetUserOTP 获取用户OTP配置
func GetUserOTP(userID int64) (*models.UserOTP, error) {
	query := `
		SELECT id, user_id, otp_secret, otp_enabled, otp_enabled_at, backup_codes, created_at, updated_at
		FROM user_otp
		WHERE user_id = ?
	`
	row := DB.QueryRow(query, userID)

	var otp models.UserOTP
	var backupCodesJSON string
	var enabledAt sql.NullTime

	err := row.Scan(
		&otp.ID,
		&otp.UserID,
		&otp.OTPSecret,
		&otp.OTPEnabled,
		&enabledAt,
		&backupCodesJSON,
		&otp.CreatedAt,
		&otp.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if enabledAt.Valid {
		otp.OTPEnabledAt = &enabledAt.Time
	}

	if err := json.Unmarshal([]byte(backupCodesJSON), &otp.BackupCodes); err != nil {
		otp.BackupCodes = []string{}
	}

	return &otp, nil
}

// GetUserOTPSecret 获取用户OTP密钥
func GetUserOTPSecret(userID int64) (string, error) {
	query := `SELECT otp_secret FROM user_otp WHERE user_id = ? AND otp_enabled = 1`
	var secret string
	err := DB.QueryRow(query, userID).Scan(&secret)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return secret, err
}

// UpdateBackupCodes 更新备用验证码
func UpdateBackupCodes(userID int64, backupCodes []string) error {
	backupCodesJSON, err := json.Marshal(backupCodes)
	if err != nil {
		return err
	}

	query := `
		UPDATE user_otp SET
			backup_codes = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`
	_, err = DB.Exec(query, string(backupCodesJSON), userID)
	return err
}

// IsOTPEnabled 检查用户是否启用了OTP
func IsOTPEnabled(userID int64) (bool, error) {
	query := `SELECT otp_enabled FROM user_otp WHERE user_id = ?`
	var enabled bool
	err := DB.QueryRow(query, userID).Scan(&enabled)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return enabled, err
}

// OTPLoginSession OTP登录会话（用于存储临时登录状态）
type OTPLoginSession struct {
	UserID    int64     `json:"userId"`
	TempToken string    `json:"tempToken"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// 内存存储OTP登录会话（生产环境应使用Redis）
var otpLoginSessions = make(map[string]*OTPLoginSession)

// CreateOTPSession 创建OTP登录会话
func CreateOTPSession(userID int64, tempToken string, expiresIn time.Duration) {
	otpLoginSessions[tempToken] = &OTPLoginSession{
		UserID:    userID,
		TempToken: tempToken,
		ExpiresAt: time.Now().Add(expiresIn),
	}
}

// GetOTPSession 获取OTP登录会话
func GetOTPSession(tempToken string) (*OTPLoginSession, bool) {
	session, ok := otpLoginSessions[tempToken]
	if !ok {
		return nil, false
	}
	if time.Now().After(session.ExpiresAt) {
		delete(otpLoginSessions, tempToken)
		return nil, false
	}
	return session, true
}

// DeleteOTPSession 删除OTP登录会话
func DeleteOTPSession(tempToken string) {
	delete(otpLoginSessions, tempToken)
}
