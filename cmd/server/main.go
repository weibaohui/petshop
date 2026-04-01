package main

import (
	"fmt"
	"log"
	"net/http"

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
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Allow", "GET, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/pet/search", handlers.SearchPets)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func run() error {
	// Application initialization
	return nil
}