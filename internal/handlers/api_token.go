// Package handlers provides HTTP handlers for the petshop API.
//
// @Description API Token 管理处理器
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"petshop/internal/db"
	"petshop/internal/logger"
	"petshop/internal/middleware"
	"petshop/internal/models"
	"petshop/internal/pagination"
	"petshop/internal/validator"
)

// generateRandomToken 生成随机Token
func generateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "psk_" + hex.EncodeToString(bytes), nil
}

// CreateAPIToken 创建新的API Token
// @Summary 创建API Token
// @Description 创建新的API Token用于Open API访问
// @Tags API Token管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.APITokenCreateRequest true "创建Token请求"
// @Success 201 {object} models.APIToken "创建成功的Token（仅返回一次完整token）"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /api/admin/tokens [post]
func CreateAPIToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取当前用户ID
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req models.APITokenCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if err := validator.Validate(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 生成随机Token
	rawToken, err := generateRandomToken()
	if err != nil {
		logger.Error("failed to generate random token", map[string]interface{}{
			"error": err.Error(),
		})
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate token"})
		return
	}
	tokenHash := db.HashToken(rawToken)

	// 设置权限，默认为只读
	permissions := req.Permissions
	if permissions == "" {
		permissions = "read"
	}

	token := &models.APIToken{
		Name:        req.Name,
		Token:       rawToken, // 仅在此处返回给客户端
		TokenHash:   tokenHash,
		Status:      "active",
		CreatedBy:   userID,
		Permissions: permissions,
	}

	// 设置过期时间
	if req.ExpiresDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, req.ExpiresDays)
		token.ExpiresAt = &expiresAt
	}

	repo := db.NewAPITokenRepository()
	if err := repo.Create(token); err != nil {
		logger.Error("failed to create api token", map[string]interface{}{
			"error": err.Error(),
		})
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create token"})
		return
	}

	logger.Info("api token created", map[string]interface{}{
		"token_id":   token.ID,
		"created_by": userID,
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(token)
}

// ListAPITokens 获取API Token列表
// @Summary 获取API Token列表
// @Description 分页获取所有API Token列表
// @Tags API Token管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Success 200 {object} models.APITokenListResponse "Token列表"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /api/admin/tokens [get]
func ListAPITokens(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取分页参数
	page, limit := pagination.GetPageAndLimit(r)
	offset := (page - 1) * limit

	repo := db.NewAPITokenRepository()
	tokens, total, err := repo.List(offset, limit)
	if err != nil {
		logger.Error("failed to list api tokens", map[string]interface{}{
			"error": err.Error(),
		})
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list tokens"})
		return
	}

	response := models.APITokenListResponse{
		List:  tokens,
		Total: total,
	}

	// 添加分页头
	pagination.SetPaginationHeaders(w, page, limit, total)
	json.NewEncoder(w).Encode(response)
}

// UpdateAPITokenStatus 更新API Token状态
// @Summary 更新API Token状态
// @Description 启用或禁用指定的API Token
// @Tags API Token管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "Token ID"
// @Param request body models.APITokenStatusUpdateRequest true "状态更新请求"
// @Success 200 {object} map[string]string "更新成功消息"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /api/admin/token [put]
func UpdateAPITokenStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取Token ID
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing token id"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid token id"})
		return
	}

	var req models.APITokenStatusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if err := validator.Validate(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	repo := db.NewAPITokenRepository()
	if err := repo.UpdateStatus(id, req.Status); err != nil {
		logger.Error("failed to update api token status", map[string]interface{}{
			"error":    err.Error(),
			"token_id": id,
		})
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to update token status"})
		return
	}

	logger.Info("api token status updated", map[string]interface{}{
		"token_id": id,
		"status":   req.Status,
	})

	json.NewEncoder(w).Encode(map[string]string{"message": "token status updated successfully"})
}

// DeleteAPIToken 删除API Token
// @Summary 删除API Token
// @Description 删除指定的API Token
// @Tags API Token管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "Token ID"
// @Success 200 {object} map[string]string "删除成功消息"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /api/admin/token [delete]
func DeleteAPIToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取Token ID
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing token id"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid token id"})
		return
	}

	repo := db.NewAPITokenRepository()
	if err := repo.Delete(id); err != nil {
		logger.Error("failed to delete api token", map[string]interface{}{
			"error":    err.Error(),
			"token_id": id,
		})
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to delete token"})
		return
	}

	logger.Info("api token deleted", map[string]interface{}{
		"token_id": id,
	})

	json.NewEncoder(w).Encode(map[string]string{"message": "token deleted successfully"})
}
