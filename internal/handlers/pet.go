package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"petshop/internal/models"
)

var petsMu sync.RWMutex

var pets = []models.Pet{
	{ID: 1, Name: "Buddy", Type: "Dog", PhotoUrls: []string{"url1"}, Status: "available"},
	{ID: 2, Name: "Whiskers", Type: "Cat", PhotoUrls: []string{"url2"}, Status: "available"},
	{ID: 3, Name: "Goldie", Type: "Fish", PhotoUrls: []string{"url3"}, Status: "available"},
}

func ListPets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	typeParam := r.URL.Query().Get("type")
	if typeParam == "" {
		json.NewEncoder(w).Encode(pets)
		return
	}
	filtered := []models.Pet{}
	for _, pet := range pets {
		if pet.Type == typeParam {
			filtered = append(filtered, pet)
		}
	}
	json.NewEncoder(w).Encode(filtered)
}

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
	for _, pet := range pets {
		if pet.ID == targetID {
			json.NewEncoder(w).Encode(pet)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

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

func PetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetPet(w, r)
	case http.MethodPut:
		UpdatePet(w, r)
	default:
		w.Header().Set("Allow", "GET, PUT")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

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