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
	dataMu.RLock()
	defer dataMu.RUnlock()

	announcementList := make([]*models.Announcement, 0, len(announcements))
	for _, a := range announcements {
		announcementList = append(announcementList, a)
	}
	json.NewEncoder(w).Encode(announcementList)
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
