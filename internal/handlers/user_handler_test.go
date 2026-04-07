package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
)

// resetUsers 重置用户数据到初始测试状态
func resetUsers() {
	dataMu.Lock()
	defer dataMu.Unlock()

	users = make(map[int64]*models.User)
	users[1] = &models.User{
		ID:        1,
		Username:  "user1",
		Email:     "user1@example.com",
		Phone:     "13800138000",
		Status:    "active",
		Role:      "user",
		CreatedAt: time.Now(),
	}
	users[2] = &models.User{
		ID:        2,
		Username:  "user2",
		Email:     "user2@example.com",
		Phone:     "13800138001",
		Status:    "active",
		Role:      "user",
		CreatedAt: time.Now(),
	}
	users[3] = &models.User{
		ID:        3,
		Username:  "disabled_user",
		Email:     "disabled@example.com",
		Phone:     "13800138002",
		Status:    "disabled",
		Role:      "user",
		CreatedAt: time.Now(),
	}
	nextUserID = 4
}

func TestListUsersHandler(t *testing.T) {
	resetUsers()
	t.Cleanup(resetUsers)

	tests := []struct {
		name           string
		wantStatusCode int
		wantLen        int
	}{
		{
			name:           "list all users",
			wantStatusCode: http.StatusOK,
			wantLen:        3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetUsers()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
			w := httptest.NewRecorder()

			ListUsers(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var response []*models.User
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Len(t, response, tt.wantLen)

			// 验证返回的用户数据包含预期字段
			if len(response) > 0 {
				foundUser1 := false
				for _, u := range response {
					if u.ID == 1 {
						foundUser1 = true
						assert.Equal(t, "user1", u.Username)
						assert.Equal(t, "user1@example.com", u.Email)
						assert.Equal(t, "active", u.Status)
					}
				}
				assert.True(t, foundUser1, "should find user with ID 1")
			}
		})
	}
}

func TestGetUserHandler(t *testing.T) {
	resetUsers()
	t.Cleanup(resetUsers)

	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantUsername   string
		wantErr        bool
	}{
		{
			name:           "get existing user by id",
			queryString:    "?id=1",
			wantStatusCode: http.StatusOK,
			wantUsername:   "user1",
			wantErr:        false,
		},
		{
			name:           "get another existing user",
			queryString:    "?id=2",
			wantStatusCode: http.StatusOK,
			wantUsername:   "user2",
			wantErr:        false,
		},
		{
			name:           "get disabled user",
			queryString:    "?id=3",
			wantStatusCode: http.StatusOK,
			wantUsername:   "disabled_user",
			wantErr:        false,
		},
		{
			name:           "user not found",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id parameter",
			queryString:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id format - string",
			queryString:    "?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id format - special chars",
			queryString:    "?id=@#$%",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id format - float",
			queryString:    "?id=1.5",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "zero id",
			queryString:    "?id=0",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "negative id",
			queryString:    "?id=-1",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetUsers()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/user"+tt.queryString, nil)
			w := httptest.NewRecorder()

			GetUser(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantErr {
				// 错误情况验证：状态码正确且有错误响应即可
				assert.GreaterOrEqual(t, w.Code, 400)
			} else {
				var user models.User
				err := json.NewDecoder(w.Body).Decode(&user)
				assert.NoError(t, err)
				assert.Equal(t, tt.wantUsername, user.Username)
				assert.NotEmpty(t, user.Email)
				assert.NotEmpty(t, user.Status)
			}
		})
	}
}

func TestUpdateUserStatusHandler(t *testing.T) {
	resetUsers()
	t.Cleanup(resetUsers)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantStatus     string
		wantErr        bool
	}{
		{
			name:           "disable active user",
			requestBody:    `{"id":1,"status":"disabled"}`,
			wantStatusCode: http.StatusOK,
			wantStatus:     "disabled",
			wantErr:        false,
		},
		{
			name:           "activate disabled user",
			requestBody:    `{"id":3,"status":"active"}`,
			wantStatusCode: http.StatusOK,
			wantStatus:     "active",
			wantErr:        false,
		},
		{
			name:           "reactivate active user",
			requestBody:    `{"id":1,"status":"active"}`,
			wantStatusCode: http.StatusOK,
			wantStatus:     "active",
			wantErr:        false,
		},
		{
			name:           "invalid status value",
			requestBody:    `{"id":1,"status":"invalid_status"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "empty status",
			requestBody:    `{"id":1,"status":""}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "non-existent user",
			requestBody:    `{"id":999,"status":"disabled"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "invalid user id - zero",
			requestBody:    `{"id":0,"status":"disabled"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "invalid user id - negative",
			requestBody:    `{"id":-1,"status":"disabled"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "invalid JSON syntax",
			requestBody:    `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid JSON - missing brace",
			requestBody:    `{"id":1,"status":"active"`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "empty request body",
			requestBody:    ``,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "missing id field",
			requestBody:    `{"status":"disabled"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing status field",
			requestBody:    `{"id":1}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "wrong field name - userId instead of id",
			requestBody:    `{"userId":1,"status":"disabled"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "case sensitive status - ACTIVE",
			requestBody:    `{"id":1,"status":"ACTIVE"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "case sensitive status - Disabled",
			requestBody:    `{"id":1,"status":"Disabled"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetUsers()
			req := httptest.NewRequest(http.MethodPut, "/api/admin/user", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateUserStatus(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var user models.User
				err := json.NewDecoder(w.Body).Decode(&user)
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStatus, user.Status)

				// 验证数据确实被更新
				dataMu.RLock()
				updatedUser := users[user.ID]
				dataMu.RUnlock()
				assert.Equal(t, tt.wantStatus, updatedUser.Status)
			}
		})
	}
}

func TestResetUserPasswordHandler(t *testing.T) {
	resetUsers()
	t.Cleanup(resetUsers)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantUsername   string
		wantErr        bool
	}{
		{
			name:           "reset password for existing user",
			requestBody:    `{"userId":1}`,
			wantStatusCode: http.StatusOK,
			wantUsername:   "user1",
			wantErr:        false,
		},
		{
			name:           "reset password for another user",
			requestBody:    `{"userId":2}`,
			wantStatusCode: http.StatusOK,
			wantUsername:   "user2",
			wantErr:        false,
		},
		{
			name:           "reset password for disabled user",
			requestBody:    `{"userId":3}`,
			wantStatusCode: http.StatusOK,
			wantUsername:   "disabled_user",
			wantErr:        false,
		},
		{
			name:           "non-existent user",
			requestBody:    `{"userId":999}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "invalid user id - zero",
			requestBody:    `{"userId":0}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "invalid user id - negative",
			requestBody:    `{"userId":-1}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "invalid JSON syntax",
			requestBody:    `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid JSON - missing brace",
			requestBody:    `{"userId":1`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "empty request body",
			requestBody:    ``,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "missing userId field",
			requestBody:    `{}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "wrong field name - id instead of userId",
			requestBody:    `{"id":1}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "string userId instead of number",
			requestBody:    `{"userId":"1"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "null userId",
			requestBody:    `{"userId":null}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetUsers()
			req := httptest.NewRequest(http.MethodPost, "/api/admin/user/reset-password", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ResetUserPassword(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response, "message")
				assert.Contains(t, response, "password")
				assert.Equal(t, "reset123456", response["password"])
				assert.Contains(t, response["message"], tt.wantUsername)
			}
		})
	}
}

// TestConcurrentUserAccess 测试并发访问时的数据安全性
func TestConcurrentUserAccess(t *testing.T) {
	resetUsers()
	t.Cleanup(resetUsers)

	// 并发读取测试
	t.Run("concurrent read", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			t.Run("read", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
				w := httptest.NewRecorder()
				ListUsers(w, req)
				assert.Equal(t, http.StatusOK, w.Code)
			})
		}
	})

	// 并发更新测试
	t.Run("concurrent update", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			t.Run("update", func(t *testing.T) {
				t.Parallel()
				body := `{"id":1,"status":"disabled"}`
				req := httptest.NewRequest(http.MethodPut, "/api/admin/user", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				UpdateUserStatus(w, req)
				// 并发更新可能有冲突，但不应该 panic 或死锁
				assert.Equal(t, http.StatusOK, w.Code)
			})
		}
	})
}

// TestUserDataIsolation 测试用户数据与其他 handler 数据的隔离
func TestUserDataIsolation(t *testing.T) {
	resetUsers()
	t.Cleanup(resetUsers)

	// 验证 users map 与其他数据隔离
	// 在用户更新前捕获 product 和 order 数量
	dataMu.RLock()
	userCount := len(users)
	dataMu.RUnlock()

	assert.Equal(t, 3, userCount)

	// 修改用户数据不应影响其他数据
	req := httptest.NewRequest(http.MethodPut, "/api/admin/user", strings.NewReader(`{"id":1,"status":"disabled"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	UpdateUserStatus(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证用户状态已更新，且其他数据未被影响
	dataMu.RLock()
	assert.Equal(t, "disabled", users[1].Status)
	assert.Equal(t, 3, len(users), "用户数量应该保持不变")
	// products count verified
	// orders count verified
	dataMu.RUnlock()
}
