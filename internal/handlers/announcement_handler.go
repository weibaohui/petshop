// Package handlers provides HTTP handlers for the petshop API.
//
// @Description 公告管理处理器
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"petshop/internal/models"
)

// Announcement management functions

// CreateAnnouncementRequest represents the request body for creating an announcement.
type CreateAnnouncementRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// UpdateAnnouncementRequest represents the request body for updating an announcement.
type UpdateAnnouncementRequest struct {
	ID      int64   `json:"id"`
	Title   *string `json:"title"`
	Content *string `json:"content"`
	Status  *string `json:"status"`
}

// ListAnnouncements 获取公告列表
// @Summary 获取公告列表
// @Description 获取所有公告
// @Tags 公告管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Announcement "公告列表"
// @Failure 401 {string} string "未授权"
// @Router /api/admin/announcements [get]
func ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	announcementList := make([]*models.Announcement, 0, len(announcements))
	for _, a := range announcements {
		announcementList = append(announcementList, a)
	}
	json.NewEncoder(w).Encode(announcementList)
}

// CreateAnnouncement 创建公告
// @Summary 创建公告
// @Description 创建新的公告
// @Tags 公告管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateAnnouncementRequest true "公告创建请求"
// @Success 201 {object} models.Announcement "创建成功的公告"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Router /api/admin/announcements [post]
func CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req CreateAnnouncementRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Title == "" || req.Content == "" {
		http.Error(w, "title and content are required", http.StatusBadRequest)
		return
	}

	now := time.Now()
	dataMu.Lock()
	defer dataMu.Unlock()

	a := &models.Announcement{
		ID:        nextAnnouncementID,
		Title:     req.Title,
		Content:   req.Content,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	announcements[nextAnnouncementID] = a
	nextAnnouncementID++

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

// UpdateAnnouncement 更新公告
// @Summary 更新公告
// @Description 更新现有公告信息
// @Tags 公告管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateAnnouncementRequest true "公告更新请求"
// @Success 200 {object} models.Announcement "更新后的公告"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "公告不存在"
// @Router /api/admin/announcement [put]
func UpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req UpdateAnnouncementRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	if existing, ok := announcements[req.ID]; ok {
		if req.Title != nil {
			existing.Title = *req.Title
		}
		if req.Content != nil {
			existing.Content = *req.Content
		}
		if req.Status != nil {
			existing.Status = *req.Status
		}
		existing.UpdatedAt = time.Now()
		json.NewEncoder(w).Encode(existing)
		return
	}
	http.Error(w, "announcement not found", http.StatusNotFound)
}

// DeleteAnnouncement 删除公告
// @Summary 删除公告
// @Description 根据ID删除公告
// @Tags 公告管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "公告ID"
// @Success 200 {object} map[string]string "删除成功消息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "公告不存在"
// @Router /api/admin/announcement [delete]
func DeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
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

	dataMu.Lock()
	defer dataMu.Unlock()

	if _, ok := announcements[id]; ok {
		delete(announcements, id)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
		return
	}
	http.Error(w, "announcement not found", http.StatusNotFound)
}
