// Package handlers provides HTTP handlers for the petshop API.
//
// @Description 用户管理处理器
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"petshop/internal/models"
)

// User management functions

// UpdateUserStatusRequest represents the request body for updating user status.
type UpdateUserStatusRequest struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

// ResetPasswordRequest represents the request body for resetting user password.
type ResetPasswordRequest struct {
	UserID int64 `json:"userId"`
}

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 获取所有用户列表
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.User "用户列表"
// @Failure 401 {string} string "未授权"
// @Router /api/admin/users [get]
func ListUsers(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	userList := make([]*models.User, 0, len(users))
	for _, u := range users {
		userList = append(userList, u)
	}
	json.NewEncoder(w).Encode(userList)
}

// GetUser 获取用户详情
// @Summary 获取用户详情
// @Description 根据ID获取用户详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "用户ID"
// @Success 200 {object} models.User "用户详情"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "用户不存在"
// @Router /api/admin/user [get]
func GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dataMu.RLock()
	defer dataMu.RUnlock()

	if u, ok := users[id]; ok {
		json.NewEncoder(w).Encode(u)
		return
	}
	http.Error(w, "user not found", http.StatusNotFound)
}

// UpdateUserStatus 更新用户状态
// @Summary 更新用户状态
// @Description 启用或禁用用户账户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateUserStatusRequest true "状态更新请求"
// @Success 200 {object} models.User "更新后的用户信息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "用户不存在"
// @Router /api/admin/user [put]
func UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req UpdateUserStatusRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	u, ok := users[req.ID]
	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if req.Status != "active" && req.Status != "disabled" {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	u.Status = req.Status
	json.NewEncoder(w).Encode(u)
}

// ResetUserPassword 重置用户密码
// @Summary 重置用户密码
// @Description 重置指定用户的密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ResetPasswordRequest true "密码重置请求"
// @Success 200 {object} map[string]string "重置成功消息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "用户不存在"
// @Router /api/admin/user/reset-password [post]
func ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req ResetPasswordRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	u, ok := users[req.UserID]
	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// 实际应用中这里会发送重置邮件或生成新密码
	// 简化处理，返回成功消息
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message":  fmt.Sprintf("密码已重置，用户%s的新密码已发送至邮箱", u.Username),
		"password": "reset123456",
	})
}
