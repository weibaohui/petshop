// Package models provides data models for the petshop system.
package models

// Pet represents a pet in the petshop system with its identification,
// name, type, photo URLs, and availability status.
type Pet struct {
	// ID is the unique identifier for the pet
	ID int64 `json:"id"`
	// Name is the name of the pet
	Name string `json:"name"`
	// Type is the species or breed of the pet (e.g., "Dog", "Cat")
	Type string `json:"type"`
	// PhotoUrls contains a list of photo URLs for the pet
	PhotoUrls []string `json:"photoUrls"`
	// Status indicates the availability of the pet (e.g., "available", "pending")
	Status string `json:"status"`
}