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
