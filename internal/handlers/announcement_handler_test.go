package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestListAnnouncements(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/announcements", nil)
	w := httptest.NewRecorder()

	ListAnnouncements(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*models.Announcement
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 1)
}

func TestCreateAnnouncement(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "create announcement with valid data",
			requestBody:    `{"title":"新公告","content":"公告内容"}`,
			wantStatusCode: http.StatusCreated,
			wantErr:        false,
		},
		{
			name:           "missing title",
			requestBody:    `{"content":"no title"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "missing content",
			requestBody:    `{"title":"no content"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminData()
			req := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			CreateAnnouncement(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if !tt.wantErr {
				var announcement models.Announcement
				err := json.NewDecoder(w.Body).Decode(&announcement)
				assert.NoError(t, err)
				assert.NotZero(t, announcement.ID)
			}
		})
	}
}

func TestUpdateAnnouncement(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "update existing announcement",
			requestBody:    `{"id":1,"title":"更新后的公告","content":"更新后的内容"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "update non-existent announcement",
			requestBody:    `{"id":999,"title":"not exist"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminData()
			req := httptest.NewRequest(http.MethodPut, "/api/admin/announcement", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateAnnouncement(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			// 验证公告确实被更新
			if tt.wantStatusCode == http.StatusOK {
				dataMu.RLock()
				updatedAnnouncement, exists := announcements[1]
				dataMu.RUnlock()
				assert.True(t, exists, "公告应该存在")
				assert.Equal(t, "更新后的公告", updatedAnnouncement.Title, "标题应该被更新")
				assert.Equal(t, "更新后的内容", updatedAnnouncement.Content, "内容应该被更新")
			}
		})
	}
}

func TestDeleteAnnouncement(t *testing.T) {
	resetAdminData()
	t.Cleanup(resetAdminData)

	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "delete existing announcement",
			queryString:    "?id=1",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "delete non-existent announcement",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id",
			queryString:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id",
			queryString:    "?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAdminData()
			req := httptest.NewRequest(http.MethodDelete, "/api/admin/announcement"+tt.queryString, nil)
			w := httptest.NewRecorder()

			DeleteAnnouncement(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			// 验证公告确实被删除
			if tt.wantStatusCode == http.StatusOK {
				dataMu.RLock()
				_, exists := announcements[1]
				dataMu.RUnlock()
				assert.False(t, exists, "公告ID=1应该被从存储中删除")
			}
		})
	}
}
