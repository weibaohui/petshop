package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"strings"
	"time"

	"petshop/internal/cache"
	"petshop/internal/logger"
	"petshop/internal/middleware"
	"petshop/internal/models"
	"petshop/internal/pagination"
	"petshop/internal/validator"
)

var (
	petsMu    sync.RWMutex
	pets      = []models.Pet{
		{ID: 1, Name: "Buddy", Type: "Dog", PhotoUrls: []string{"url1"}, Status: "available"},
		{ID: 2, Name: "Whiskers", Type: "Cat", PhotoUrls: []string{"url2"}, Status: "available"},
		{ID: 3, Name: "Goldie", Type: "Fish", PhotoUrls: []string{"url3"}, Status: "available"},
	}
	petCache  *cache.PetCache
	csrfProt  *middleware.CSRFProtection
	petLogger = logger.New("handlers")
)

func init() {
	petCache = cache.NewPetCache(1000, 5*time.Minute)
	csrfProt = middleware.NewCSRFProtection()
}

// ListPets returns paginated list of pets
func ListPets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	typeParam := r.URL.Query().Get("type")
	page := pagination.ParsePagination(r)

	petsMu.RLock()
	defer petsMu.RUnlock()

	// Filter pets if type specified
	var filtered []models.Pet
	for _, pet := range pets {
		if typeParam == "" || strings.EqualFold(pet.Type, typeParam) {
			filtered = append(filtered, pet)
		}
	}

	// Convert to interface slice for pagination
	items := make([]interface{}, len(filtered))
	for i, pet := range filtered {
		items[i] = pet
	}

	pagedPage, pagedItems := pagination.Paginate(items, page.Page, page.PageSize)

	// Convert back to Pet slice
	result := make([]models.Pet, len(pagedItems))
	for i, item := range pagedItems {
		result[i] = item.(models.Pet)
	}

	petLogger.Info("list pets", map[string]interface{}{
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     pagedPage.Total,
	})

	json.NewEncoder(w).Encode(pagination.NewPagedResponse(result, pagedPage))
}

// GetPet returns a single pet by ID
func GetPet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}

	var targetID int64
	if _, err := fmt.Sscanf(idStr, "%d", &targetID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid id format"})
		return
	}

	// Try cache first
	cacheKey := cache.GetPetKey(targetID)
	if cached, found := petCache.Get(cacheKey); found {
		petLogger.Debug("cache hit", map[string]interface{}{"id": targetID})
		json.NewEncoder(w).Encode(cached)
		return
	}

	petsMu.RLock()
	defer petsMu.RUnlock()

	for _, pet := range pets {
		if pet.ID == targetID {
			// Store in cache
			petCache.Set(cacheKey, pet)
			json.NewEncoder(w).Encode(pet)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// DeletePet deletes a pet by ID
func DeletePet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}

	var targetID int64
	if _, err := fmt.Sscanf(idStr, "%d", &targetID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid id format"})
		return
	}

	petsMu.Lock()
	defer petsMu.Unlock()

	for i, pet := range pets {
		if pet.ID == targetID {
			deletedPet := pets[i]
			pets = append(pets[:i], pets[i+1:]...)
			// Invalidate cache
			petCache.Delete(cache.GetPetKey(targetID))
			petLogger.Info("pet deleted", map[string]interface{}{"id": targetID})
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(deletedPet)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// SearchPets searches pets by name with pagination
func SearchPets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nameParam := r.URL.Query().Get("name")
	page := pagination.ParsePagination(r)

	petsMu.RLock()
	defer petsMu.RUnlock()

	var filtered []models.Pet
	for _, pet := range pets {
		if nameParam == "" || containsIgnoreCase(pet.Name, nameParam) {
			filtered = append(filtered, pet)
		}
	}

	// Convert to interface slice for pagination
	items := make([]interface{}, len(filtered))
	for i, pet := range filtered {
		items[i] = pet
	}

	pagedPage, pagedItems := pagination.Paginate(items, page.Page, page.PageSize)

	// Convert back to Pet slice
	result := make([]models.Pet, len(pagedItems))
	for i, item := range pagedItems {
		result[i] = item.(models.Pet)
	}

	json.NewEncoder(w).Encode(pagination.NewPagedResponse(result, pagedPage))
}

// containsIgnoreCase checks if s contains substr, case-insensitively.
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// UpdatePet updates a pet with validation
func UpdatePet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	var pet models.Pet
	if err := json.Unmarshal(body, &pet); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON format"})
		return
	}

	if pet.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}

	if pet.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "name is required"})
		return
	}

	// Check for SQL injection in input
	if validator.ContainsSQLKeywords(pet.Name) || validator.ContainsSQLKeywords(pet.Type) {
		petLogger.Warn("potential injection attempt", map[string]interface{}{
			"name": pet.Name,
			"type": pet.Type,
		})
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid characters in input"})
		return
	}

	petsMu.Lock()
	defer petsMu.Unlock()

	for i, p := range pets {
		if p.ID == pet.ID {
			// Validate input only if the pet exists
			if errs := validator.ValidatePet(pet.Name, pet.Type, pet.Status); errs.HasErrors() {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": errs.Error()})
				return
			}

			pets[i].Name = pet.Name
			if pet.Type != "" {
				pets[i].Type = pet.Type
			}
			if pet.PhotoUrls != nil {
				pets[i].PhotoUrls = pet.PhotoUrls
			}
			if pet.Status != "" {
				pets[i].Status = pet.Status
			}
			// Invalidate cache
			petCache.Delete(cache.GetPetKey(pet.ID))
			petLogger.Info("pet updated", map[string]interface{}{"id": pet.ID})
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(pets[i])
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// AddPetPhoto adds a photo to a pet
func AddPetPhoto(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid path"})
		return
	}
	var targetID int64
	if _, err := fmt.Sscanf(pathParts[3], "%d", &targetID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid id format"})
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	// Validate URL
	if req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "url is required"})
		return
	}

	if !validator.ValidateURL(req.URL) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid url format"})
		return
	}

	// Check for XSS
	if validator.ContainsSQLKeywords(req.URL) {
		petLogger.Warn("potential XSS in photo URL", map[string]interface{}{
			"url": req.URL,
		})
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid url"})
		return
	}

	petsMu.Lock()
	defer petsMu.Unlock()

	for i, pet := range pets {
		if pet.ID == targetID {
			for idx, existingUrl := range pets[i].PhotoUrls {
				if existingUrl == req.URL {
					pets[i].PhotoUrls = append(pets[i].PhotoUrls[:idx], pets[i].PhotoUrls[idx+1:]...)
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(pets[i].PhotoUrls)
					return
				}
			}
			pets[i].PhotoUrls = append(pets[i].PhotoUrls, req.URL)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(pets[i].PhotoUrls)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// DeletePetPhoto deletes a photo from a pet
func DeletePetPhoto(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid path"})
		return
	}
	var targetID int64
	if _, err := fmt.Sscanf(pathParts[3], "%d", &targetID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid id format"})
		return
	}

	urlStr := r.URL.Query().Get("url")
	if urlStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "url parameter is required"})
		return
	}

	petsMu.Lock()
	defer petsMu.Unlock()

	for i, pet := range pets {
		if pet.ID == targetID {
			for j, p := range pets[i].PhotoUrls {
				if p == urlStr {
					pets[i].PhotoUrls = append(pets[i].PhotoUrls[:j], pets[i].PhotoUrls[j+1:]...)
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(pets[i].PhotoUrls)
					return
				}
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(pets[i].PhotoUrls)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// GetPetPhotos returns photos for a pet
func GetPetPhotos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	petsMu.RLock()
	defer petsMu.RUnlock()

	pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid path"})
		return
	}
	var targetID int64
	if _, err := fmt.Sscanf(pathParts[3], "%d", &targetID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid id format"})
		return
	}

	for _, pet := range pets {
		if pet.ID == targetID {
			json.NewEncoder(w).Encode(pet.PhotoUrls)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// PetPhotoHandler routes photo-related requests
func PetPhotoHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetPetPhotos(w, r)
	case http.MethodPost:
		AddPetPhoto(w, r)
	case http.MethodDelete:
		DeletePetPhoto(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// GetCacheStats returns cache statistics
func GetCacheStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(petCache.Stats())
}

// GetCacheHitRate returns the cache hit rate
func GetCacheHitRate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"hit_rate": petCache.HitRate(),
	})
}
