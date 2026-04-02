package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"petshop/internal/models"
)

// Carousel management functions

// CreateCarouselRequest represents the request body for creating a carousel.
type CreateCarouselRequest struct {
	ImageURL  string `json:"imageUrl"`
	LinkURL   string `json:"linkUrl"`
	SortOrder int    `json:"sortOrder"`
	Title     string `json:"title"`
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
