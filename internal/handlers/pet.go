package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"strings"

	"petshop/internal/models"
)

var petsMu sync.RWMutex

var pets = []models.Pet{
	{ID: 1, Name: "Buddy", Type: "Dog", PhotoUrls: []string{"url1"}, Status: "available"},
	{ID: 2, Name: "Whiskers", Type: "Cat", PhotoUrls: []string{"url2"}, Status: "available"},
	{ID: 3, Name: "Goldie", Type: "Fish", PhotoUrls: []string{"url3"}, Status: "available"},
}

// ListPets handles GET /api/pets and returns all pets, optionally filtered by type.
// It acquires a read lock for thread-safe access to the pets slice.
func ListPets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	typeParam := r.URL.Query().Get("type")

	petsMu.RLock()
	defer petsMu.RUnlock()

	if typeParam == "" {
		if err := json.NewEncoder(w).Encode(pets); err != nil {
			http.Error(w, "encoding error", http.StatusInternalServerError)
			return
		}
		return
	}
	filtered := []models.Pet{}
	for _, pet := range pets {
		if strings.EqualFold(pet.Type, typeParam) {
			filtered = append(filtered, pet)
		}
	}
	if err := json.NewEncoder(w).Encode(filtered); err != nil {
		http.Error(w, "encoding error", http.StatusInternalServerError)
	}
}

// GetPet handles GET /api/pet?id=<id> and returns the pet with the specified ID.
// Returns 400 if id is missing, and 404 if the pet is not found.
func GetPet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}
	var targetID int64
	fmt.Sscanf(idStr, "%d", &targetID)

	petsMu.RLock()
	defer petsMu.RUnlock()

	for _, pet := range pets {
		if pet.ID == targetID {
			json.NewEncoder(w).Encode(pet)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// DeletePet handles DELETE /api/pet?id=<id> and removes the pet with the specified ID.
// Returns 400 if id is missing or invalid, and 404 if the pet is not found.
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
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(deletedPet)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// SearchPets handles GET /api/pet/search?name=<name> and returns pets matching the name.
// If name is empty, returns all pets. Search is case-insensitive.
func SearchPets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nameParam := r.URL.Query().Get("name")

	petsMu.RLock()
	defer petsMu.RUnlock()

	if nameParam == "" {
		json.NewEncoder(w).Encode(pets)
		return
	}

	filtered := []models.Pet{}
	for _, pet := range pets {
		if containsIgnoreCase(pet.Name, nameParam) {
			filtered = append(filtered, pet)
		}
	}
	json.NewEncoder(w).Encode(filtered)
}

// containsIgnoreCase checks if s contains substr, case-insensitively.
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// UpdatePet handles PUT /api/pet and updates an existing pet's information.
// It validates required fields (id, name) and updates only non-empty fields.
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

	petsMu.Lock()
	defer petsMu.Unlock()

	for i, p := range pets {
		if p.ID == pet.ID {
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
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(pets[i])
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// AddPetPhoto handles POST /api/pet/<id>/photos and adds a photo URL to the pet.
// If the URL already exists, it removes the existing one (toggle behavior).
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

	if req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "url is required"})
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

// DeletePetPhoto handles DELETE /api/pet/<id>/photos?url=<url> and removes a photo URL from the pet.
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

// GetPetPhotos handles GET /api/pet/<id>/photos and returns the photo URLs of the pet.
func GetPetPhotos(w http.ResponseWriter, r *http.Request) {
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

	petsMu.RLock()
	defer petsMu.RUnlock()

	for _, pet := range pets {
		if pet.ID == targetID {
			json.NewEncoder(w).Encode(pet.PhotoUrls)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// PetPhotoHandler handles GET, POST, DELETE for /api/pet/<id>/photos endpoints.
// It delegates to GetPetPhotos, AddPetPhoto, and DeletePetPhoto respectively.
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