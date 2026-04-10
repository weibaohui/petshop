package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"petshop/internal/models"
)

// User management functions

// In-memory user storage (kept for compatibility)
var users = make(map[int64]*models.User)
var userMu sync.RWMutex
var nextUserID int64 = 1

func init() {
	// Initialize sample users
	userMu.Lock()
	now := time.Now()
	users[1] = &models.User{
		ID:        1,
		Username:  "user1",
		Email:     "user1@example.com",
		Phone:     "13800138000",
		Status:    "active",
		Role:      "user",
		CreatedAt: now,
	}
	users[2] = &models.User{
		ID:        2,
		Username:  "user2",
		Email:     "user2@example.com",
		Phone:     "13800138001",
		Status:    "active",
		Role:      "user",
		CreatedAt: now,
	}
	nextUserID = 3
	userMu.Unlock()
}

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
// @Summary List users
// @Description Get a list of all users (admin only)
// @Tags admin-users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.User
// @Router /api/admin/users [get]
func ListUsers(w http.ResponseWriter, r *http.Request) {
	userMu.RLock()
	defer userMu.RUnlock()

	userList := make([]*models.User, 0, len(users))
	for _, u := range users {
		userList = append(userList, u)
	}
	_ = json.NewEncoder(w).Encode(userList)
}

// GetUser handles GET /api/admin/user?id=<id> and returns the user.
// @Summary Get user
// @Description Get a user by ID (admin only)
// @Tags admin-users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query int true "User ID"
// @Success 200 {object} models.User
// @Failure 400 {string} string "id is required"
// @Failure 404 {string} string "user not found"
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

	userMu.RLock()
	defer userMu.RUnlock()

	if u, ok := users[id]; ok {
		_ = json.NewEncoder(w).Encode(u)
		return
	}
	http.Error(w, "user not found", http.StatusNotFound)
}

// UpdateUserStatus handles PUT /api/admin/user and updates the user status.
// @Summary Update user status
// @Description Update a user's status (admin only)
// @Tags admin-users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateUserStatusRequest true "User status update"
// @Success 200 {object} models.User
// @Failure 400 {string} string "invalid request body"
// @Failure 404 {string} string "user not found"
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

	userMu.Lock()
	defer userMu.Unlock()

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
	_ = json.NewEncoder(w).Encode(u)
}

// ResetUserPassword handles POST /api/admin/user/reset-password and resets the user password.
// @Summary Reset user password
// @Description Reset a user's password (admin only)
// @Tags admin-users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ResetPasswordRequest true "Reset password request"
// @Success 200 {object} map[string]string
// @Failure 400 {string} string "invalid request body"
// @Failure 404 {string} string "user not found"
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

	userMu.Lock()
	defer userMu.Unlock()

	u, ok := users[req.UserID]
	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// In real app, send reset email or generate new password
	// Simplified: return success message
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":  fmt.Sprintf("密码已重置，用户%s的新密码已发送至邮箱", u.Username),
		"password": "reset123456",
	})
}
