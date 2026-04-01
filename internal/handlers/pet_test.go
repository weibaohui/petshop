package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"petshop/internal/models"
	"petshop/internal/pagination"
)

func resetPets() {
	pets = []models.Pet{
		{ID: 1, Name: "Buddy", Type: "Dog", PhotoUrls: []string{"url1"}, Status: "available"},
		{ID: 2, Name: "Whiskers", Type: "Cat", PhotoUrls: []string{"url2"}, Status: "available"},
		{ID: 3, Name: "Goldie", Type: "Fish", PhotoUrls: []string{"url3"}, Status: "available"},
	}
}

func TestListPets(t *testing.T) {
	defer resetPets()
	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantLen        int
	}{
		{
			name:           "list all pets without filter",
			queryString:    "",
			wantStatusCode: http.StatusOK,
			wantLen:        3,
		},
		{
			name:           "filter by type Dog",
			queryString:    "?type=Dog",
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "filter by type Cat",
			queryString:    "?type=Cat",
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "filter by non-existent type",
			queryString:    "?type=Bird",
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
		{
			name:           "filter type is case-insensitive",
			queryString:    "?type=dog",
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/pets"+tt.queryString, nil)
			w := httptest.NewRecorder()

			ListPets(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("ListPets() status = %d, want %d", w.Code, tt.wantStatusCode)
			}

			var response pagination.PagedResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Errorf("ListPets() failed to decode response: %v", err)
			}

			data, ok := response.Data.([]interface{})
			if !ok {
				t.Errorf("ListPets() response data is not an array")
				return
			}

			if len(data) != tt.wantLen {
				t.Errorf("ListPets() got %d pets, want %d", len(data), tt.wantLen)
			}
		})
	}
}

func TestGetPet(t *testing.T) {
	defer resetPets()
	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantPetName    string
		wantErr        bool
	}{
		{
			name:           "get existing pet by id",
			queryString:    "?id=1",
			wantStatusCode: http.StatusOK,
			wantPetName:    "Buddy",
			wantErr:        false,
		},
		{
			name:           "get another existing pet",
			queryString:    "?id=2",
			wantStatusCode: http.StatusOK,
			wantPetName:    "Whiskers",
			wantErr:        false,
		},
		{
			name:           "pet not found",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id parameter",
			queryString:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/pet"+tt.queryString, nil)
			w := httptest.NewRecorder()

			GetPet(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("GetPet() status = %d, want %d", w.Code, tt.wantStatusCode)
			}

			if tt.wantErr {
				var errResp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Errorf("GetPet() failed to decode error response: %v", err)
				}
				if _, ok := errResp["error"]; !ok {
					t.Errorf("GetPet() expected error response, got none")
				}
			} else {
				var pet models.Pet
				if err := json.NewDecoder(w.Body).Decode(&pet); err != nil {
					t.Errorf("GetPet() failed to decode pet: %v", err)
				}
				if pet.Name != tt.wantPetName {
					t.Errorf("GetPet() got pet name = %s, want %s", pet.Name, tt.wantPetName)
				}
			}
		})
	}
}

func TestDeletePet(t *testing.T) {
	defer resetPets()
	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "delete existing pet",
			queryString:    "?id=3",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "pet not found",
			queryString:    "?id=999",
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id parameter",
			queryString:    "",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id format",
			queryString:    "?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/pet"+tt.queryString, nil)
			w := httptest.NewRecorder()

			DeletePet(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("DeletePet() status = %d, want %d", w.Code, tt.wantStatusCode)
			}

			if tt.wantErr {
				var errResp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Errorf("DeletePet() failed to decode error response: %v", err)
				}
				if _, ok := errResp["error"]; !ok {
					t.Errorf("DeletePet() expected error response, got none")
				}
			}
		})
	}
}

func TestSearchPets(t *testing.T) {
	defer resetPets()
	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantLen        int
	}{
		{
			name:           "search without filter returns all pets",
			queryString:    "",
			wantStatusCode: http.StatusOK,
			wantLen:        3,
		},
		{
			name:           "search by exact name",
			queryString:    "?name=Buddy",
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "search by partial name",
			queryString:    "?name=bud",
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "search is case-insensitive",
			queryString:    "?name=buddy",
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "search by non-existent name",
			queryString:    "?name=Dragon",
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/pets/search"+tt.queryString, nil)
			w := httptest.NewRecorder()

			SearchPets(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("SearchPets() status = %d, want %d", w.Code, tt.wantStatusCode)
			}

			var response pagination.PagedResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Errorf("SearchPets() failed to decode response: %v", err)
			}

			data, ok := response.Data.([]interface{})
			if !ok {
				t.Errorf("SearchPets() response data is not an array")
				return
			}

			if len(data) != tt.wantLen {
				t.Errorf("SearchPets() got %d pets, want %d", len(data), tt.wantLen)
			}
		})
	}
}

func TestUpdatePet(t *testing.T) {
	defer resetPets()
	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantPetName    string
		wantErr        bool
	}{
		{
			name:           "update existing pet",
			requestBody:    `{"id":1,"name":"BuddyUpdated","type":"Dog"}`,
			wantStatusCode: http.StatusOK,
			wantPetName:    "BuddyUpdated",
			wantErr:        false,
		},
		{
			name:           "pet not found",
			requestBody:    `{"id":999,"name":"NonExistent"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "missing id",
			requestBody:    `{"name":"NoId"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "missing name",
			requestBody:    `{"id":1}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid json",
			requestBody:    `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "empty body",
			requestBody:    ``,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/pet", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdatePet(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("UpdatePet() status = %d, want %d", w.Code, tt.wantStatusCode)
			}

			if tt.wantErr {
				var errResp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Errorf("UpdatePet() failed to decode error response: %v", err)
				}
				if _, ok := errResp["error"]; !ok {
					t.Errorf("UpdatePet() expected error response, got none")
				}
			} else {
				var pet models.Pet
				if err := json.NewDecoder(w.Body).Decode(&pet); err != nil {
					t.Errorf("UpdatePet() failed to decode pet: %v", err)
				}
				if pet.Name != tt.wantPetName {
					t.Errorf("UpdatePet() got pet name = %s, want %s", pet.Name, tt.wantPetName)
				}
			}
		})
	}
}
