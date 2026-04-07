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

	t.Run("Pet with all fields populated", func(t *testing.T) {
		pet := Pet{
			ID:           4,
			Name:         "Max",
			Type:         "Dog",
			Breed:        "Golden Retriever",
			PhotoUrls:    []string{"url1", "url2", "url3"},
			Status:       "available",
			Age:          24,
			AgeDisplay:   "2岁",
			Price:        2999.99,
			Description:  "Friendly golden retriever",
			HealthStatus: "healthy",
			VaccinationRecords: []VaccinationRecord{
				{Name: "Rabies", Date: "2024-01-01", Completed: true},
				{Name: "DHPP", Date: "2024-02-01", Completed: true},
			},
			CreatedAt: "2024-01-01T00:00:00Z",
		}

		data, err := json.Marshal(pet)
		assert.NoError(t, err)

		var decoded Pet
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, pet.ID, decoded.ID)
		assert.Equal(t, pet.Name, decoded.Name)
		assert.Equal(t, pet.Breed, decoded.Breed)
		assert.Equal(t, pet.Age, decoded.Age)
		assert.Equal(t, pet.AgeDisplay, decoded.AgeDisplay)
		assert.Equal(t, pet.Price, decoded.Price)
		assert.Equal(t, pet.Description, decoded.Description)
		assert.Equal(t, pet.HealthStatus, decoded.HealthStatus)
		assert.Len(t, decoded.VaccinationRecords, 2)
		assert.Equal(t, pet.VaccinationRecords[0].Name, decoded.VaccinationRecords[0].Name)
		assert.Equal(t, pet.VaccinationRecords[1].Completed, decoded.VaccinationRecords[1].Completed)
		assert.Equal(t, pet.CreatedAt, decoded.CreatedAt)
	})

	t.Run("Pet with zero values", func(t *testing.T) {
		pet := Pet{
			ID:     0,
			Name:   "",
			Type:   "",
			Status: "",
			Age:    0,
			Price:  0,
		}

		data, err := json.Marshal(pet)
		assert.NoError(t, err)

		var decoded Pet
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, int64(0), decoded.ID)
		assert.Empty(t, decoded.Name)
		assert.Empty(t, decoded.Type)
		assert.Empty(t, decoded.Status)
		assert.Equal(t, 0, decoded.Age)
		assert.Equal(t, 0.0, decoded.Price)
	})

	t.Run("Pet with negative price", func(t *testing.T) {
		pet := Pet{
			ID:    5,
			Name:  "Test",
			Price: -100.50,
		}

		data, err := json.Marshal(pet)
		assert.NoError(t, err)

		var decoded Pet
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, -100.50, decoded.Price)
	})
}

func TestVaccinationRecordStruct(t *testing.T) {
	t.Run("VaccinationRecord JSON serialization and deserialization", func(t *testing.T) {
		record := VaccinationRecord{
			Name:      "Rabies",
			Date:      "2024-01-15",
			Completed: true,
		}

		data, err := json.Marshal(record)
		assert.NoError(t, err)

		var decoded VaccinationRecord
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, record.Name, decoded.Name)
		assert.Equal(t, record.Date, decoded.Date)
		assert.Equal(t, record.Completed, decoded.Completed)
	})

	t.Run("VaccinationRecord with false completed", func(t *testing.T) {
		record := VaccinationRecord{
			Name:      "DHPP",
			Date:      "2024-02-01",
			Completed: false,
		}

		data, err := json.Marshal(record)
		assert.NoError(t, err)

		var decoded VaccinationRecord
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.False(t, decoded.Completed)
	})

	t.Run("VaccinationRecord JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"name":"Bordetella","date":"2024-03-01","completed":true}`

		var record VaccinationRecord
		err := json.Unmarshal([]byte(jsonStr), &record)
		assert.NoError(t, err)

		assert.Equal(t, "Bordetella", record.Name)
		assert.Equal(t, "2024-03-01", record.Date)
		assert.True(t, record.Completed)
	})
}

func TestPetFilterStruct(t *testing.T) {
	t.Run("PetFilter JSON serialization and deserialization", func(t *testing.T) {
		filter := PetFilter{
			Type:     "Dog",
			MinPrice: 100.00,
			MaxPrice: 1000.00,
			Search:   "Golden",
			Status:   "available",
			Page:     1,
			PageSize: 20,
		}

		data, err := json.Marshal(filter)
		assert.NoError(t, err)

		var decoded PetFilter
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, filter.Type, decoded.Type)
		assert.Equal(t, filter.MinPrice, decoded.MinPrice)
		assert.Equal(t, filter.MaxPrice, decoded.MaxPrice)
		assert.Equal(t, filter.Search, decoded.Search)
		assert.Equal(t, filter.Status, decoded.Status)
		assert.Equal(t, filter.Page, decoded.Page)
		assert.Equal(t, filter.PageSize, decoded.PageSize)
	})

	t.Run("PetFilter with zero values", func(t *testing.T) {
		filter := PetFilter{}

		data, err := json.Marshal(filter)
		assert.NoError(t, err)

		var decoded PetFilter
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Empty(t, decoded.Type)
		assert.Equal(t, 0.0, decoded.MinPrice)
		assert.Equal(t, 0.0, decoded.MaxPrice)
		assert.Empty(t, decoded.Search)
		assert.Empty(t, decoded.Status)
		assert.Equal(t, 0, decoded.Page)
		assert.Equal(t, 0, decoded.PageSize)
	})

	t.Run("PetFilter with negative price range", func(t *testing.T) {
		filter := PetFilter{
			MinPrice: -50.00,
			MaxPrice: -10.00,
		}

		data, err := json.Marshal(filter)
		assert.NoError(t, err)

		var decoded PetFilter
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, -50.00, decoded.MinPrice)
		assert.Equal(t, -10.00, decoded.MaxPrice)
	})

	t.Run("PetFilter JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"type":"Cat","minPrice":50,"maxPrice":500,"search":"Persian","status":"pending","page":2,"pageSize":10}`

		var filter PetFilter
		err := json.Unmarshal([]byte(jsonStr), &filter)
		assert.NoError(t, err)

		assert.Equal(t, "Cat", filter.Type)
		assert.Equal(t, 50.0, filter.MinPrice)
		assert.Equal(t, 500.0, filter.MaxPrice)
		assert.Equal(t, "Persian", filter.Search)
		assert.Equal(t, "pending", filter.Status)
		assert.Equal(t, 2, filter.Page)
		assert.Equal(t, 10, filter.PageSize)
	})
}

func TestCategoryStruct(t *testing.T) {
	t.Run("Category JSON serialization and deserialization", func(t *testing.T) {
		category := Category{
			ID:   1,
			Name: "狗狗",
		}

		data, err := json.Marshal(category)
		assert.NoError(t, err)

		var decoded Category
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, category.ID, decoded.ID)
		assert.Equal(t, category.Name, decoded.Name)
	})

	t.Run("Category with zero ID", func(t *testing.T) {
		category := Category{
			ID:   0,
			Name: "Unknown",
		}

		data, err := json.Marshal(category)
		assert.NoError(t, err)

		var decoded Category
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, int64(0), decoded.ID)
		assert.Equal(t, "Unknown", decoded.Name)
	})

	t.Run("Category JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"id":5,"name":"兔子"}`

		var category Category
		err := json.Unmarshal([]byte(jsonStr), &category)
		assert.NoError(t, err)

		assert.Equal(t, int64(5), category.ID)
		assert.Equal(t, "兔子", category.Name)
	})
}
