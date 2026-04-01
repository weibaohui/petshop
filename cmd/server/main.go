package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"petshop/internal/handlers"
)

func main() {
	fmt.Println("Project: petshop")

	http.HandleFunc("/api/pets", handlers.ListPets)
	http.HandleFunc("/api/pet", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetPet(w, r)
		case http.MethodPut:
			handlers.UpdatePet(w, r)
		case http.MethodDelete:
			handlers.DeletePet(w, r)
		default:
			w.Header().Set("Allow", "GET, PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/pet/search", handlers.SearchPets)
	http.HandleFunc("/api/pet/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		pathParts := strings.Split(strings.TrimSuffix(path, "/"), "/")
		// Expected format: ["", "api", "pet", "{id}"] or ["", "api", "pet", "{id}", "photos"]
		if len(pathParts) != 4 && len(pathParts) != 5 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if pathParts[0] != "" || pathParts[1] != "api" || pathParts[2] != "pet" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Validate id is numeric
		var id int64
		if _, err := fmt.Sscanf(pathParts[3], "%d", &id); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// If 5 parts, must be ".../{id}/photos"
		if len(pathParts) == 5 {
			if pathParts[4] != "photos" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			handlers.PetPhotoHandler(w, r)
			return
		}
		// len == 4, base path ".../{id}"
		switch r.Method {
		case http.MethodGet:
			handlers.GetPet(w, r)
		case http.MethodPut:
			handlers.UpdatePet(w, r)
		case http.MethodDelete:
			handlers.DeletePet(w, r)
		default:
			w.Header().Set("Allow", "GET, PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func run() error {
	// Application initialization
	return nil
}