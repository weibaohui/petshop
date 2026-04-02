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

// ListUsers handles GET /api/admin/users and returns all users.
func ListUsers(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	userList := make([]*models.User, 0, len(users))
	for _, u := range users {
		userList = append(userList, u)
	}
	json.NewEncoder(w).Encode(userList)
}

// GetUser handles GET /api/admin/user?id=<id> and returns the user.
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

// UpdateUserStatus handles PUT /api/admin/user and updates the user status.
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

// ResetUserPassword handles POST /api/admin/user/reset-password and resets the user password.
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
