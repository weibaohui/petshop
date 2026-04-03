package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"

	"petshop/internal/models"
)

// APITokenRepository API Token数据访问层
type APITokenRepository struct {
	db *sql.DB
}

// NewAPITokenRepository 创建Token仓库实例
func NewAPITokenRepository() *APITokenRepository {
	return &APITokenRepository{db: db}
}

// InitAPITokenTable 初始化Token表
func InitAPITokenTable() error {
	schema := `
	CREATE TABLE IF NOT EXISTS api_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME,
		expires_at DATETIME,
		created_by INTEGER NOT NULL,
		permissions TEXT DEFAULT 'read'
	);
	CREATE INDEX IF NOT EXISTS idx_api_tokens_hash ON api_tokens(token_hash);
	CREATE INDEX IF NOT EXISTS idx_api_tokens_status ON api_tokens(status);
	`
	_, err := db.Exec(schema)
	return err
}

// HashToken 计算Token的哈希值
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Create 创建新的API Token
func (r *APITokenRepository) Create(token *models.APIToken) error {
	query := `
		INSERT INTO api_tokens (name, token_hash, status, created_by, permissions, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query, token.Name, token.TokenHash, token.Status,
		token.CreatedBy, token.Permissions, token.ExpiresAt)
	if err != nil {
		return err
	}
	token.ID, _ = result.LastInsertId()
	return nil
}

// GetByTokenHash 根据Token哈希值获取Token信息
func (r *APITokenRepository) GetByTokenHash(tokenHash string) (*models.APIToken, error) {
	query := `
		SELECT id, name, token_hash, status, created_at, updated_at, last_used_at, expires_at, created_by, permissions
		FROM api_tokens WHERE token_hash = ?
	`
	row := r.db.QueryRow(query, tokenHash)

	token := &models.APIToken{}
	var expiresAt, lastUsedAt sql.NullTime
	err := row.Scan(&token.ID, &token.Name, &token.TokenHash, &token.Status,
		&token.CreatedAt, &token.UpdatedAt, &lastUsedAt, &expiresAt, &token.CreatedBy, &token.Permissions)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		token.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		token.LastUsedAt = &lastUsedAt.Time
	}
	return token, nil
}

// List 获取Token列表
func (r *APITokenRepository) List(offset, limit int) ([]models.APIToken, int, error) {
	// 获取总数
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM api_tokens").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表（不返回token_hash）
	query := `
		SELECT id, name, status, created_at, updated_at, last_used_at, expires_at, created_by, permissions
		FROM api_tokens ORDER BY created_at DESC LIMIT ? OFFSET ?
	`
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tokens []models.APIToken
	for rows.Next() {
		token := models.APIToken{}
		var expiresAt, lastUsedAt sql.NullTime
		err := rows.Scan(&token.ID, &token.Name, &token.Status,
			&token.CreatedAt, &token.UpdatedAt, &lastUsedAt, &expiresAt, &token.CreatedBy, &token.Permissions)
		if err != nil {
			continue
		}
		if expiresAt.Valid {
			token.ExpiresAt = &expiresAt.Time
		}
		if lastUsedAt.Valid {
			token.LastUsedAt = &lastUsedAt.Time
		}
		tokens = append(tokens, token)
	}
	return tokens, total, nil
}

// UpdateStatus 更新Token状态
func (r *APITokenRepository) UpdateStatus(id int64, status string) error {
	query := "UPDATE api_tokens SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := r.db.Exec(query, status, id)
	return err
}

// Delete 删除Token
func (r *APITokenRepository) Delete(id int64) error {
	query := "DELETE FROM api_tokens WHERE id = ?"
	_, err := r.db.Exec(query, id)
	return err
}

// UpdateLastUsedAt 更新最后使用时间
func (r *APITokenRepository) UpdateLastUsedAt(id int64) error {
	query := "UPDATE api_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := r.db.Exec(query, id)
	return err
}

// IsTokenValid 检查Token是否有效
func (r *APITokenRepository) IsTokenValid(tokenHash string) (bool, *models.APIToken) {
	token, err := r.GetByTokenHash(tokenHash)
	if err != nil || token == nil {
		return false, nil
	}

	// 检查状态
	if token.Status != "active" {
		return false, nil
	}

	// 检查是否过期
	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		return false, nil
	}

	return true, token
}
