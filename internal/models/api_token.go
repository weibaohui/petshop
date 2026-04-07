package models

import "time"

// APIToken 开放API调用令牌
type APIToken struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`   // Token名称/描述
	Token       string     `json:"token"`  // Token值（仅创建时返回）
	TokenHash   string     `json:"-"`      // Token哈希值（存储）
	Status      string     `json:"status"` // active, disabled
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"` // 过期时间，null表示永不过期
	CreatedBy   int64      `json:"createdBy"`           // 创建者用户ID
	Permissions string     `json:"permissions"`         // 权限列表，逗号分隔，如 "read,write"
}

// APITokenCreateRequest 创建Token请求
type APITokenCreateRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	ExpiresDays int    `json:"expiresDays"` // 0表示永不过期
	Permissions string `json:"permissions"` // 逗号分隔的权限列表
}

// APITokenListResponse Token列表响应
type APITokenListResponse struct {
	List  []APIToken `json:"list"`
	Total int        `json:"total"`
}

// APITokenStatusUpdateRequest 更新Token状态请求
type APITokenStatusUpdateRequest struct {
	Status string `json:"status" validate:"required,oneof=active disabled"`
}
