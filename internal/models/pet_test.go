package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

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