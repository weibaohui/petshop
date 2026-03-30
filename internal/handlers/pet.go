package handlers

import (
	"encoding/json"
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
	json.NewEncoder(w).Encode(pets)
}

func GetPet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.URL.Query().Get("id")
	for _, pet := range pets {
		if pet.ID == 1 {
			json.NewEncoder(w).Encode(pet)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}