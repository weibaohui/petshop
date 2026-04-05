package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"time"

	"petshop/internal/models"
)

// APITokenRepository API令牌数据访问
type APITokenRepository struct{}

// NewAPITokenRepository 创建仓库实例
func NewAPITokenRepository() *APITokenRepository {
	return &APITokenRepository{}
}

// hashToken 计算token的哈希值
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Create 创建API令牌
func (r *APITokenRepository) Create(token *models.APIToken) error {
	tokenHash := hashToken(token.Token)
	result, err := db.Exec(
		`INSERT INTO api_tokens (name, token_hash, description, status, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		token.Name, tokenHash, token.Description, token.Status, token.ExpiresAt, token.CreatedAt, token.UpdatedAt,
	)
	if err != nil {
		return err
	}
	token.ID, _ = result.LastInsertId()
	token.TokenHash = tokenHash
	return nil
}

// GetByTokenHash 根据token哈希获取令牌
func (r *APITokenRepository) GetByTokenHash(tokenHash string) (*models.APIToken, error) {
	var token models.APIToken
	err := db.QueryRow(
		`SELECT id, name, token_hash, description, status, last_used_at, expires_at, created_at, updated_at
		 FROM api_tokens WHERE token_hash = ?`,
		tokenHash,
	).Scan(&token.ID, &token.Name, &token.TokenHash, &token.Description, &token.Status,
		&token.LastUsedAt, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// GetByID 根据ID获取令牌
func (r *APITokenRepository) GetByID(id int64) (*models.APIToken, error) {
	var token models.APIToken
	err := db.QueryRow(
		`SELECT id, name, token_hash, description, status, last_used_at, expires_at, created_at, updated_at
		 FROM api_tokens WHERE id = ?`,
		id,
	).Scan(&token.ID, &token.Name, &token.TokenHash, &token.Description, &token.Status,
		&token.LastUsedAt, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// List 获取令牌列表
func (r *APITokenRepository) List(limit, offset int) ([]models.APIToken, int64, error) {
	// 获取总数
	var total int64
	err := db.QueryRow(`SELECT COUNT(*) FROM api_tokens`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	rows, err := db.Query(
		`SELECT id, name, token_hash, description, status, last_used_at, expires_at, created_at, updated_at
		 FROM api_tokens ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tokens := make([]models.APIToken, 0)
	for rows.Next() {
		var token models.APIToken
		err := rows.Scan(&token.ID, &token.Name, &token.TokenHash, &token.Description, &token.Status,
			&token.LastUsedAt, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		tokens = append(tokens, token)
	}

	return tokens, total, nil
}

// UpdateStatus 更新令牌状态
func (r *APITokenRepository) UpdateStatus(id int64, status string) error {
	_, err := db.Exec(
		`UPDATE api_tokens SET status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now(), id,
	)
	return err
}

// Delete 删除令牌
func (r *APITokenRepository) Delete(id int64) error {
	_, err := db.Exec(`DELETE FROM api_tokens WHERE id = ?`, id)
	return err
}

// UpdateLastUsedAt 更新最后使用时间
func (r *APITokenRepository) UpdateLastUsedAt(id int64) error {
	now := time.Now()
	_, err := db.Exec(
		`UPDATE api_tokens SET last_used_at = ? WHERE id = ?`,
		now, id,
	)
	return err
}

// ValidateToken 验证token是否有效
func (r *APITokenRepository) ValidateToken(token string) (*models.APIToken, error) {
	tokenHash := hashToken(token)
	t, err := r.GetByTokenHash(tokenHash)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}

	// 检查状态
	if t.Status != "active" {
		return nil, nil
	}

	// 检查是否过期
	if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
		return nil, nil
	}

	// 更新最后使用时间（异步）
	go func(tokenID int64) {
		if err := r.UpdateLastUsedAt(tokenID); err != nil {
			log.Printf("failed to update last_used_at for token %d: %v", tokenID, err)
		}
	}(t.ID)

	return t, nil
}
