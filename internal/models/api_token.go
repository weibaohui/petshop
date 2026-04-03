package models

import "time"

// APIToken API访问令牌
type APIToken struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`        // 令牌名称
	Token       string    `json:"token"`       // 令牌值（仅创建时返回完整token）
	TokenHash   string    `json:"-"`           // 令牌哈希（存储）
	Description string    `json:"description"` // 描述
	Status      string    `json:"status"`      // active, disabled
	LastUsedAt  *time.Time `json:"lastUsedAt"` // 最后使用时间
	ExpiresAt   *time.Time `json:"expiresAt"`  // 过期时间
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// APITokenCreateRequest 创建令牌请求
type APITokenCreateRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Description string `json:"description" validate:"max=500"`
	ExpiresDays int    `json:"expiresDays"` // 过期天数，0表示永不过期
}

// APITokenResponse 令牌响应（隐藏敏感信息）
type APITokenResponse struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	LastUsedAt  *time.Time `json:"lastUsedAt"`
	ExpiresAt   *time.Time `json:"expiresAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// APITokenCreateResponse 创建令牌响应（包含完整token）
type APITokenCreateResponse struct {
	APITokenResponse
	Token string `json:"token"` // 完整令牌，仅创建时返回
}

// ToResponse 转换为响应格式
func (t *APIToken) ToResponse() APITokenResponse {
	return APITokenResponse{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Status:      t.Status,
		LastUsedAt:  t.LastUsedAt,
		ExpiresAt:   t.ExpiresAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

// ToCreateResponse 转换为创建响应格式
func (t *APIToken) ToCreateResponse() APITokenCreateResponse {
	return APITokenCreateResponse{
		APITokenResponse: t.ToResponse(),
		Token:            t.Token,
	}
}
