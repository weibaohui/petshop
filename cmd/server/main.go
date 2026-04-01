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
	http.HandleFunc("/api/pet", handlers.PetHandler)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func run() error {
	// Application initialization
	return nil
}