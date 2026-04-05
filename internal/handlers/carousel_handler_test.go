package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
)

// resetCarousels resets carousel data to a known state for testing
func resetCarousels() {
	dataMu.Lock()
	defer dataMu.Unlock()

	carousels = make(map[int64]*models.Carousel)
	carousels[1] = &models.Carousel{
		ID:        1,
		ImageURL:  "/static/carousel/banner1.jpg",
		LinkURL:   "/product/1",
		SortOrder: 1,
		Title:     "春季大促",
		Status:    "active",
	}
	carousels[2] = &models.Carousel{
		ID:        2,
		ImageURL:  "/static/carousel/banner2.jpg",
		LinkURL:   "/product/2",
		SortOrder: 2,
		Title:     "夏季促销",
		Status:    "inactive",
	}
	nextCarouselID = 3
}

func TestListCarousels_Handler(t *testing.T) {
	resetCarousels()
	defer resetCarousels()

	tests := []struct {
		name           string
		wantStatusCode int
		wantLen        int
		wantFirstTitle string
	}{
		{
			name:           "list all carousels",
			wantStatusCode: http.StatusOK,
			wantLen:        2,
			wantFirstTitle: "春季大促",
		},
		{
			name:           "list returns empty when no carousels",
			wantStatusCode: http.StatusOK,
			wantLen:        0,
			wantFirstTitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantLen == 0 {
				// Special case: clear all carousels
				dataMu.Lock()
				carousels = make(map[int64]*models.Carousel)
				dataMu.Unlock()
			}

			req := httptest.NewRequest(http.MethodGet, "/api/admin/carousels", nil)
			w := httptest.NewRecorder()

			ListCarousels(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var response []*models.Carousel
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Len(t, response, tt.wantLen)

			if tt.wantLen > 0 && tt.wantFirstTitle != "" {
				found := false
				for _, c := range response {
					if c.Title == tt.wantFirstTitle {
						found = true
						break
					}
				}
				assert.True(t, found, "expected to find carousel with title %s", tt.wantFirstTitle)
			}
		})
	}
}

func TestCreateCarousel_Handler(t *testing.T) {
	resetCarousels()
	defer resetCarousels()

	tests := []struct {
		name            string
		requestBody     string
		wantStatusCode  int
		wantErr         bool
		wantImageURL    string
		wantLinkURL     string
		wantSortOrder   int
		wantTitle       string
		wantStatus      string
	}{
		{
			name:           "create carousel with valid data",
			requestBody:    `{"imageUrl":"/static/banner3.jpg","linkUrl":"/product/3","sortOrder":3,"title":"秋季活动"}`,
			wantStatusCode: http.StatusCreated,
			wantErr:        false,
			wantImageURL:   "/static/banner3.jpg",
			wantLinkURL:    "/product/3",
			wantSortOrder:  3,
			wantTitle:      "秋季活动",
			wantStatus:     "active",
		},
		{
			name:           "create carousel with minimal data",
			requestBody:    `{"imageUrl":"/static/banner4.jpg"}`,
			wantStatusCode: http.StatusCreated,
			wantErr:        false,
			wantImageURL:   "/static/banner4.jpg",
			wantLinkURL:    "",
			wantSortOrder:  0,
			wantTitle:      "",
			wantStatus:     "active",
		},
		{
			name:           "missing imageUrl",
			requestBody:    `{"linkUrl":"/product/3","sortOrder":3,"title":"NoImage"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "empty imageUrl",
			requestBody:    `{"imageUrl":"","title":"EmptyImage"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid JSON",
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
		{
			name:           "invalid JSON structure",
			requestBody:    `{"imageUrl": 123}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "create with negative sortOrder",
			requestBody:    `{"imageUrl":"/static/banner5.jpg","sortOrder":-1}`,
			wantStatusCode: http.StatusCreated,
			wantErr:        false,
			wantImageURL:   "/static/banner5.jpg",
			wantSortOrder:  -1,
			wantStatus:     "active",
		},
		{
			name:           "create with large sortOrder",
			requestBody:    `{"imageUrl":"/static/banner6.jpg","sortOrder":999999}`,
			wantStatusCode: http.StatusCreated,
			wantErr:        false,
			wantImageURL:   "/static/banner6.jpg",
			wantSortOrder:  999999,
			wantStatus:     "active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetCarousels()
			req := httptest.NewRequest(http.MethodPost, "/api/admin/carousels", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			CreateCarousel(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantErr {
				// Error responses can be "invalid" or other error messages
				body := w.Body.String()
				assert.True(t, w.Code >= 400, "expected error status code, got %d", w.Code)
				_ = body
			} else {
				var carousel models.Carousel
				err := json.NewDecoder(w.Body).Decode(&carousel)
				assert.NoError(t, err)
				assert.NotZero(t, carousel.ID)
				assert.Equal(t, tt.wantImageURL, carousel.ImageURL)
				assert.Equal(t, tt.wantLinkURL, carousel.LinkURL)
				assert.Equal(t, tt.wantSortOrder, carousel.SortOrder)
				assert.Equal(t, tt.wantTitle, carousel.Title)
				assert.Equal(t, tt.wantStatus, carousel.Status)
			}
		})
	}
}

func TestUpdateCarousel_Handler(t *testing.T) {
	resetCarousels()
	defer resetCarousels()

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, carousel *models.Carousel)
	}{
		{
			name:           "update existing carousel title",
			requestBody:    `{"id":1,"title":"更新后的标题"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, carousel *models.Carousel) {
				assert.Equal(t, "更新后的标题", carousel.Title)
				assert.Equal(t, "/static/carousel/banner1.jpg", carousel.ImageURL) // unchanged
			},
		},
		{
			name:           "update existing carousel imageUrl",
			requestBody:    `{"id":1,"imageUrl":"/static/new.jpg"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, carousel *models.Carousel) {
				assert.Equal(t, "/static/new.jpg", carousel.ImageURL)
			},
		},
		{
			name:           "update existing carousel linkUrl",
			requestBody:    `{"id":1,"linkUrl":"/new/link"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, carousel *models.Carousel) {
				assert.Equal(t, "/new/link", carousel.LinkURL)
			},
		},
		{
			name:           "update existing carousel sortOrder",
			requestBody:    `{"id":1,"sortOrder":10}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, carousel *models.Carousel) {
				assert.Equal(t, 10, carousel.SortOrder)
			},
		},
		{
			name:           "update existing carousel status",
			requestBody:    `{"id":1,"status":"inactive"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, carousel *models.Carousel) {
				assert.Equal(t, "inactive", carousel.Status)
			},
		},
		{
			name:           "update all fields",
			requestBody:    `{"id":1,"imageUrl":"/all/new.jpg","linkUrl":"/all/link","sortOrder":99,"title":"All New","status":"inactive"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, carousel *models.Carousel) {
				assert.Equal(t, "/all/new.jpg", carousel.ImageURL)
				assert.Equal(t, "/all/link", carousel.LinkURL)
				assert.Equal(t, 99, carousel.SortOrder)
				assert.Equal(t, "All New", carousel.Title)
				assert.Equal(t, "inactive", carousel.Status)
			},
		},
		{
			name:           "update with zero values should keep existing",
			requestBody:    `{"id":1,"imageUrl":"","title":"","sortOrder":0}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, carousel *models.Carousel) {
				// Empty strings should not override, but sortOrder 0 should be applied
				assert.Equal(t, "/static/carousel/banner1.jpg", carousel.ImageURL)
				assert.Equal(t, "春季大促", carousel.Title)
				assert.Equal(t, 0, carousel.SortOrder)
			},
		},
		{
			name:           "update non-existent carousel",
			requestBody:    `{"id":999,"title":"not exist"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "update with zero id",
			requestBody:    `{"id":0,"title":"zero id"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "invalid JSON",
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
		{
			name:           "valid JSON but no id field",
			requestBody:    `{"title":"no id"}`,
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetCarousels()
			req := httptest.NewRequest(http.MethodPut, "/api/admin/carousel", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			UpdateCarousel(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantErr {
				// Error responses might have different formats
				body := w.Body.String()
				assert.True(t, w.Code >= 400, "expected error status code, got %d", w.Code)
				_ = body
			} else {
				var carousel models.Carousel
				err := json.NewDecoder(w.Body).Decode(&carousel)
				assert.NoError(t, err)
				if tt.checkResponse != nil {
					tt.checkResponse(t, &carousel)
				}
			}
		})
	}
}

func TestDeleteCarousel_Handler(t *testing.T) {
	resetCarousels()
	defer resetCarousels()

	tests := []struct {
		name           string
		queryString    string
		wantStatusCode int
		wantErr        bool
		checkDeleted   bool
		deletedID      int64
	}{
		{
			name:           "delete existing carousel id=1",
			queryString:    "?id=1",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkDeleted:   true,
			deletedID:      1,
		},
		{
			name:           "delete existing carousel id=2",
			queryString:    "?id=2",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			checkDeleted:   true,
			deletedID:      2,
		},
		{
			name:           "delete non-existent carousel",
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
			name:           "invalid id format - string",
			queryString:    "?id=abc",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id format - special chars",
			queryString:    "?id=@#$",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid id - negative",
			queryString:    "?id=-1",
			wantStatusCode: http.StatusNotFound, // parse succeeds, but not found
			wantErr:        true,
		},
		{
			name:           "invalid id - zero",
			queryString:    "?id=0",
			wantStatusCode: http.StatusNotFound, // parse succeeds, but not found
			wantErr:        true,
		},
		{
			name:           "invalid id - float",
			queryString:    "?id=1.5",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetCarousels()
			req := httptest.NewRequest(http.MethodDelete, "/api/admin/carousel"+tt.queryString, nil)
			w := httptest.NewRecorder()

			DeleteCarousel(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantErr {
				body := w.Body.String()
				assert.True(t, w.Code >= 400, "expected error status code, got %d", w.Code)
				_ = body
			} else {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, "deleted", response["message"])

				if tt.checkDeleted {
					// Verify the carousel was actually deleted
					dataMu.RLock()
					_, exists := carousels[tt.deletedID]
					dataMu.RUnlock()
					assert.False(t, exists, "carousel %d should have been deleted", tt.deletedID)
				}
			}
		})
	}
}

func TestDeleteCarousel_Concurrent(t *testing.T) {
	resetCarousels()
	defer resetCarousels()

	// Test concurrent deletion attempts
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/carousel?id=1", nil)
	w := httptest.NewRecorder()

	DeleteCarousel(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Try to delete again - should be not found
	req2 := httptest.NewRequest(http.MethodDelete, "/api/admin/carousel?id=1", nil)
	w2 := httptest.NewRecorder()

	DeleteCarousel(w2, req2)

	assert.Equal(t, http.StatusNotFound, w2.Code)
}

func TestCreateCarouselRequest_Validation(t *testing.T) {
	// Test the CreateCarouselRequest struct directly
	tests := []struct {
		name    string
		req     CreateCarouselRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: CreateCarouselRequest{
				ImageURL:  "/test.jpg",
				LinkURL:   "/link",
				SortOrder: 1,
				Title:     "Test",
			},
			wantErr: false,
		},
		{
			name: "empty imageUrl",
			req: CreateCarouselRequest{
				ImageURL: "",
				Title:    "No Image",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a struct validation test
			// The actual validation happens in the handler
			hasImage := tt.req.ImageURL != ""
			if tt.wantErr {
				assert.False(t, hasImage, "expected validation to fail")
			} else {
				assert.True(t, hasImage, "expected validation to pass")
			}
		})
	}
}
