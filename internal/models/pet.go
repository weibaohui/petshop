// Package models provides data models for the petshop system.
package models

// PetStatus represents the availability status of a pet
type PetStatus string

const (
	// StatusAvailable indicates the pet is available for sale
	StatusAvailable PetStatus = "available"
	// StatusPending indicates the pet has been reserved
	StatusPending PetStatus = "pending"
	// StatusSold indicates the pet has been sold
	StatusSold PetStatus = "sold"
)

// Category represents a pet category
type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Pet represents a pet in the petshop system with its identification,
// name, type, photo URLs, and availability status.
type Pet struct {
	// ID is the unique identifier for the pet
	ID int64 `json:"id"`
	// Name is the name of the pet
	Name string `json:"name"`
	// Type is the species or breed of the pet (e.g., "Dog", "Cat")
	Type string `json:"type"`
	// Breed is the specific breed of the pet
	Breed string `json:"breed"`
	// PhotoUrls contains a list of photo URLs for the pet
	PhotoUrls []string `json:"photoUrls"`
	// Status indicates the availability of the pet (e.g., "available", "pending", "sold")
	Status string `json:"status"`
	// Age is the age of the pet in months
	Age int `json:"age"`
	// AgeDisplay is a human-readable age string (e.g., "2个月", "1岁")
	AgeDisplay string `json:"ageDisplay"`
	// Price is the price of the pet in CNY
	Price float64 `json:"price"`
	// Description is a detailed description of the pet
	Description string `json:"description"`
	// HealthStatus describes the health condition of the pet
	HealthStatus string `json:"healthStatus"`
	// VaccinationRecords contains vaccination information
	VaccinationRecords []VaccinationRecord `json:"vaccinationRecords"`
	// Rating is the customer rating for the pet (1-5 stars)
	Rating int `json:"rating"`
	// CreatedAt is when the pet was added
	CreatedAt string `json:"createdAt"`
}

// VaccinationRecord represents a vaccination record for a pet
type VaccinationRecord struct {
	Name      string `json:"name"`
	Date      string `json:"date"`
	Completed bool   `json:"completed"`
}

// PetFilter represents filter options for pet search
type PetFilter struct {
	Type     string  `json:"type"`
	MinPrice float64 `json:"minPrice"`
	MaxPrice float64 `json:"maxPrice"`
	Search   string  `json:"search"`
	Status   string  `json:"status"`
	Page     int     `json:"page"`
	PageSize int     `json:"pageSize"`
}

// IsValidStatusTransition checks if a pet status transition is valid.
// Valid transitions: available -> pending, available -> sold, pending -> sold.
func IsValidStatusTransition(from, to PetStatus) bool {
	switch from {
	case StatusAvailable:
		return to == StatusPending || to == StatusSold
	case StatusPending:
		return to == StatusSold
	case StatusSold:
		return false
	default:
		return false
	}
}

// Categories returns all available pet categories
func Categories() []Category {
	return []Category{
		{ID: 1, Name: "狗狗"},
		{ID: 2, Name: "猫咪"},
		{ID: 3, Name: "鸟类"},
		{ID: 4, Name: "其他"},
	}
}
