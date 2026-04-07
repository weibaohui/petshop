package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCategories(t *testing.T) {
	t.Run("Categories returns correct categories", func(t *testing.T) {
		categories := Categories()

		assert.Len(t, categories, 4)
		assert.Equal(t, int64(1), categories[0].ID)
		assert.Equal(t, "狗狗", categories[0].Name)
		assert.Equal(t, int64(2), categories[1].ID)
		assert.Equal(t, "猫咪", categories[1].Name)
		assert.Equal(t, int64(3), categories[2].ID)
		assert.Equal(t, "鸟类", categories[2].Name)
		assert.Equal(t, int64(4), categories[3].ID)
		assert.Equal(t, "其他", categories[3].Name)
	})

	t.Run("Categories returns a new slice each time", func(t *testing.T) {
		categories1 := Categories()
		categories2 := Categories()

		// Ensure they are independent slices
		categories1[0].Name = "Modified"
		assert.Equal(t, "狗狗", categories2[0].Name)
	})
}

func TestPetStatusConstants(t *testing.T) {
	t.Run("PetStatus constants have correct values", func(t *testing.T) {
		assert.Equal(t, PetStatus("available"), StatusAvailable)
		assert.Equal(t, PetStatus("pending"), StatusPending)
		assert.Equal(t, PetStatus("sold"), StatusSold)
	})

	t.Run("PetStatus can be compared", func(t *testing.T) {
		var status PetStatus = "available"
		assert.Equal(t, StatusAvailable, status)
		assert.NotEqual(t, StatusPending, status)
	})
}

func TestPetStruct(t *testing.T) {
	t.Run("Pet JSON serialization and deserialization", func(t *testing.T) {
		pet := Pet{
			ID:        1,
			Name:      "Buddy",
			Type:      "Dog",
			PhotoUrls: []string{"url1", "url2"},
			Status:    "available",
		}

		data, err := json.Marshal(pet)
		assert.NoError(t, err)

		var decoded Pet
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, pet.ID, decoded.ID)
		assert.Equal(t, pet.Name, decoded.Name)
		assert.Equal(t, pet.Type, decoded.Type)
		assert.Equal(t, pet.PhotoUrls, decoded.PhotoUrls)
		assert.Equal(t, pet.Status, decoded.Status)
	})

	t.Run("Pet JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"id":2,"name":"Whiskers","type":"Cat","photoUrls":["url3"],"status":"available"}`

		var pet Pet
		err := json.Unmarshal([]byte(jsonStr), &pet)
		assert.NoError(t, err)

		assert.Equal(t, int64(2), pet.ID)
		assert.Equal(t, "Whiskers", pet.Name)
		assert.Equal(t, "Cat", pet.Type)
		assert.Equal(t, []string{"url3"}, pet.PhotoUrls)
		assert.Equal(t, "available", pet.Status)
	})

	t.Run("Pet with empty slices", func(t *testing.T) {
		pet := Pet{
			ID:        3,
			Name:      "Goldie",
			Type:      "Fish",
			PhotoUrls: []string{},
			Status:    "available",
		}

		data, err := json.Marshal(pet)
		assert.NoError(t, err)

		var decoded Pet
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, pet.ID, decoded.ID)
		assert.Empty(t, decoded.PhotoUrls)
		assert.Equal(t, pet.Status, decoded.Status)
	})
}
