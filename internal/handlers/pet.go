package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"petshop/internal/models"
)

var pets = []models.Pet{
	{ID: 1, Name: "Buddy", Type: "Dog", Price: 299.99},
	{ID: 2, Name: "Whiskers", Type: "Cat", Price: 199.99},
	{ID: 3, Name: "Goldie", Type: "Fish", Price: 49.99},
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