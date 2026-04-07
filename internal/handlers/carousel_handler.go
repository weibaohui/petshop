package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"petshop/internal/logger"
	"petshop/internal/models"
)

// carouselLogger is the logger for carousel operations
var carouselLogger = logger.New("carousel_handler")

// Carousel management functions

// CreateCarouselRequest represents the request body for creating a carousel.
type CreateCarouselRequest struct {
	ImageURL  string `json:"imageUrl"`
	LinkURL   string `json:"linkUrl"`
	SortOrder int    `json:"sortOrder"`
	Title     string `json:"title"`
}

// UpdateCarouselRequest represents the request body for updating a carousel.
type UpdateCarouselRequest struct {
	ID        int64   `json:"id"`
	ImageURL  *string `json:"imageUrl"`
	LinkURL   *string `json:"linkUrl"`
	SortOrder *int    `json:"sortOrder"`
	Title     *string `json:"title"`
	Status    *string `json:"status"`
}

// isValidURL validates if the given string is a valid URL with HTTP/HTTPS scheme
func isValidURL(str string) bool {
	if str == "" {
		return false
	}
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// isValidStatus validates if the given string is a valid carousel status
func isValidStatus(status string) bool {
	return status == "active" || status == "inactive"
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
		carouselLogger.Error("failed to read request body", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req CreateCarouselRequest
	if err := json.Unmarshal(body, &req); err != nil {
		carouselLogger.Error("failed to unmarshal JSON", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ImageURL == "" {
		carouselLogger.Warn("create carousel failed: imageUrl is required", map[string]interface{}{
			"title": req.Title,
		})
		http.Error(w, "imageUrl is required", http.StatusBadRequest)
		return
	}

	// Validate ImageURL format (must be valid HTTP/HTTPS URL)
	if !isValidURL(req.ImageURL) {
		carouselLogger.Warn("create carousel failed: invalid imageUrl format", map[string]interface{}{
			"imageUrl": req.ImageURL,
			"title":    req.Title,
		})
		http.Error(w, "imageUrl must be a valid HTTP/HTTPS URL", http.StatusBadRequest)
		return
	}

	// Validate LinkURL format if provided
	if req.LinkURL != "" && !isValidURL(req.LinkURL) {
		carouselLogger.Warn("create carousel failed: invalid linkUrl format", map[string]interface{}{
			"linkUrl": req.LinkURL,
			"title":   req.Title,
		})
		http.Error(w, "linkUrl must be a valid HTTP/HTTPS URL", http.StatusBadRequest)
		return
	}

	// Validate SortOrder is non-negative
	if req.SortOrder < 0 {
		carouselLogger.Warn("create carousel failed: sortOrder must be non-negative", map[string]interface{}{
			"sortOrder": req.SortOrder,
			"title":     req.Title,
		})
		http.Error(w, "sortOrder must be non-negative", http.StatusBadRequest)
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

	carouselLogger.Info("carousel created successfully", map[string]interface{}{
		"id":       c.ID,
		"imageUrl": c.ImageURL,
		"title":    c.Title,
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

// UpdateCarousel handles PUT /api/admin/carousel and updates an existing carousel.
func UpdateCarousel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		carouselLogger.Error("failed to read request body", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req UpdateCarouselRequest
	if err := json.Unmarshal(body, &req); err != nil {
		carouselLogger.Error("failed to unmarshal JSON", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	if existing, ok := carousels[req.ID]; ok {
		// Validate and update ImageURL
		if req.ImageURL != nil {
			if *req.ImageURL == "" {
				carouselLogger.Warn("update carousel failed: imageUrl cannot be empty", map[string]interface{}{
					"id": req.ID,
				})
				http.Error(w, "imageUrl cannot be empty", http.StatusBadRequest)
				return
			}
			if !isValidURL(*req.ImageURL) {
				carouselLogger.Warn("update carousel failed: invalid imageUrl format", map[string]interface{}{
					"id":       req.ID,
					"imageUrl": *req.ImageURL,
				})
				http.Error(w, "imageUrl must be a valid HTTP/HTTPS URL", http.StatusBadRequest)
				return
			}
			existing.ImageURL = *req.ImageURL
		}

		// Validate and update LinkURL
		if req.LinkURL != nil {
			if *req.LinkURL != "" && !isValidURL(*req.LinkURL) {
				carouselLogger.Warn("update carousel failed: invalid linkUrl format", map[string]interface{}{
					"id":      req.ID,
					"linkUrl": *req.LinkURL,
				})
				http.Error(w, "linkUrl must be a valid HTTP/HTTPS URL", http.StatusBadRequest)
				return
			}
			existing.LinkURL = *req.LinkURL
		}

		// Validate and update SortOrder
		if req.SortOrder != nil {
			if *req.SortOrder < 0 {
				carouselLogger.Warn("update carousel failed: sortOrder must be non-negative", map[string]interface{}{
					"id":        req.ID,
					"sortOrder": *req.SortOrder,
				})
				http.Error(w, "sortOrder must be non-negative", http.StatusBadRequest)
				return
			}
			existing.SortOrder = *req.SortOrder
		}

		// Update Title
		if req.Title != nil {
			existing.Title = *req.Title
		}

		// Validate and update Status
		if req.Status != nil {
			if !isValidStatus(*req.Status) {
				carouselLogger.Warn("update carousel failed: invalid status value", map[string]interface{}{
					"id":     req.ID,
					"status": *req.Status,
				})
				http.Error(w, "status must be either 'active' or 'inactive'", http.StatusBadRequest)
				return
			}
			existing.Status = *req.Status
		}

		carouselLogger.Info("carousel updated successfully", map[string]interface{}{
			"id":     existing.ID,
			"title":  existing.Title,
			"status": existing.Status,
		})

		json.NewEncoder(w).Encode(existing)
		return
	}

	carouselLogger.Warn("carousel not found", map[string]interface{}{
		"id": req.ID,
	})
	http.Error(w, "carousel not found", http.StatusNotFound)
}

// DeleteCarousel handles DELETE /api/admin/carousel?id=<id> and deletes the carousel.
func DeleteCarousel(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		carouselLogger.Warn("delete carousel failed: id is required", nil)
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		carouselLogger.Error("delete carousel failed: invalid id format", map[string]interface{}{
			"id":    idStr,
			"error": err.Error(),
		})
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	if _, ok := carousels[id]; ok {
		delete(carousels, id)
		carouselLogger.Info("carousel deleted successfully", map[string]interface{}{
			"id": id,
		})
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
		return
	}

	carouselLogger.Warn("delete carousel failed: carousel not found", map[string]interface{}{
		"id": id,
	})
	http.Error(w, "carousel not found", http.StatusNotFound)
}
