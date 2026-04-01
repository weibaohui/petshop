package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"petshop/internal/models"
)

// System configuration functions (carousels, announcements, configs)

// CreateCarouselRequest represents the request body for creating a carousel.
type CreateCarouselRequest struct {
	ImageURL  string `json:"imageUrl"`
	LinkURL   string `json:"linkUrl"`
	SortOrder int    `json:"sortOrder"`
	Title     string `json:"title"`
}

// CreateAnnouncementRequest represents the request body for creating an announcement.
type CreateAnnouncementRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// SetSystemConfigRequest represents the request body for setting a system config.
type SetSystemConfigRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ListCarousels handles GET /api/admin/carousels and returns all carousels.
func ListCarousels(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	carouselList := make([]*models.Carousel, 0, len(carousels))
	for _, c := range carousels {
		carouselList = append(carouselList, c)
	}
	json.NewEncoder(w).Encode(carouselList)
}

// CreateCarousel handles POST /api/admin/carousels and creates a new carousel.
func CreateCarousel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req CreateCarouselRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	c := &models.Carousel{
		ID:        nextCarouselID,
		ImageURL:  req.ImageURL,
		LinkURL:   req.LinkURL,
		SortOrder: req.SortOrder,
		Title:     req.Title,
		Status:    "active",
	}
	carousels[nextCarouselID] = c
	nextCarouselID++

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

// UpdateCarousel handles PUT /api/admin/carousel and updates an existing carousel.
func UpdateCarousel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var c models.Carousel
	if err := json.Unmarshal(body, &c); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	if existing, ok := carousels[c.ID]; ok {
		if c.ImageURL != "" {
			existing.ImageURL = c.ImageURL
		}
		if c.LinkURL != "" {
			existing.LinkURL = c.LinkURL
		}
		existing.SortOrder = c.SortOrder
		if c.Title != "" {
			existing.Title = c.Title
		}
		if c.Status != "" {
			existing.Status = c.Status
		}
		json.NewEncoder(w).Encode(existing)
		return
	}
	http.Error(w, "carousel not found", http.StatusNotFound)
}

// DeleteCarousel handles DELETE /api/admin/carousel?id=<id> and deletes the carousel.
func DeleteCarousel(w http.ResponseWriter, r *http.Request) {
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

	if _, ok := carousels[id]; ok {
		delete(carousels, id)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
		return
	}
	http.Error(w, "carousel not found", http.StatusNotFound)
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

	var a models.Announcement
	if err := json.Unmarshal(body, &a); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	if existing, ok := announcements[a.ID]; ok {
		if a.Title != "" {
			existing.Title = a.Title
		}
		if a.Content != "" {
			existing.Content = a.Content
		}
		if a.Status != "" {
			existing.Status = a.Status
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

// GetSystemConfigs handles GET /api/admin/configs and returns all system configurations.
func GetSystemConfigs(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	configs := make([]models.SystemConfig, 0, len(systemConfigs))
	for k, v := range systemConfigs {
		configs = append(configs, models.SystemConfig{Key: k, Value: v})
	}
	json.NewEncoder(w).Encode(configs)
}

// SetSystemConfig handles POST /api/admin/config and sets a system configuration value.
// If the key is "inventory_threshold", it updates the inventory alert threshold.
func SetSystemConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req SetSystemConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	systemConfigs[req.Key] = req.Value

	// 更新库存预警阈值
	if req.Key == "inventory_threshold" {
		if v, err := strconv.Atoi(req.Value); err == nil {
			inventoryThreshold = v
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "config updated"})
}
