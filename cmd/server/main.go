package main

import (
	"log"
	"net/http"

	"petshop/internal/handlers"
)

func main() {
	http.HandleFunc("/api/pets", handlers.ListPets)
	http.HandleFunc("/api/pet", handlers.GetPet)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}