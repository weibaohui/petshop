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

// ListAnnouncements handles GET /api/admin/announcements and returns all announcements.
func ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	announcements, err := announcementRepo.GetAll()
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(announcements)
}

// CreateAnnouncement handles POST /api/admin/announcements and creates a new announcement.
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
	a := &models.Announcement{
		Title:     req.Title,
		Content:   req.Content,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := announcementRepo.Create(a); err != nil {
		http.Error(w, "failed to create announcement", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(a)
}

// UpdateAnnouncement handles PUT /api/admin/announcement and updates an existing announcement.
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

	existing, err := announcementRepo.GetByID(req.ID)
	if err != nil {
		http.Error(w, "announcement not found", http.StatusNotFound)
		return
	}

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

	if err := announcementRepo.Update(existing); err != nil {
		http.Error(w, "failed to update announcement", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(existing)
}

// DeleteAnnouncement handles DELETE /api/admin/announcement?id=<id> and deletes the announcement.
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

	if _, err := announcementRepo.GetByID(id); err != nil {
		http.Error(w, "announcement not found", http.StatusNotFound)
		return
	}

	if err := announcementRepo.Delete(id); err != nil {
		http.Error(w, "failed to delete announcement", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
}
