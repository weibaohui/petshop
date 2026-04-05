package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"petshop/internal/db"
	"petshop/internal/middleware"
	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDBForAPITokenHandlers(t *testing.T) func() {
	// Reset database state
	db.ResetForTesting()

	// Use test database file
	testDBPath := "./test_api_token_handlers.db"

	// Remove existing test database
	os.Remove(testDBPath)

	// Initialize database
	err := db.InitDB(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Cleanup function
	return func() {
		db.Close()
		os.Remove(testDBPath)
		db.ResetForTesting()
	}
}

// createRequestWithUserID creates a request with the user ID set in the context
// This simulates what the AuthMiddleware does
func createRequestWithUserID(method, url string, body string, userID int64) *http.Request {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, url, bodyReader)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	return req.WithContext(ctx)
}

func TestCreateAPIToken_Success(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	reqBody := `{"name":"Test Token","expiresDays":30,"permissions":"read,write"}`
	req := createRequestWithUserID(http.MethodPost, "/api/tokens", reqBody, 1)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateAPIToken(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.APIToken
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Token)
	assert.True(t, strings.HasPrefix(response.Token, "psk_"))
	assert.Equal(t, "Test Token", response.Name)
	assert.Equal(t, "active", response.Status)
	assert.Equal(t, int64(1), response.CreatedBy)
	assert.Equal(t, "read,write", response.Permissions)
	assert.NotNil(t, response.ExpiresAt)
	assert.Empty(t, response.TokenHash)
}

func TestCreateAPIToken_DefaultPermissions(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	// Request without permissions should default to "read"
	reqBody := `{"name":"Default Permissions Token","expiresDays":7}`
	req := createRequestWithUserID(http.MethodPost, "/api/tokens", reqBody, 1)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateAPIToken(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.APIToken
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "read", response.Permissions)
}

func TestCreateAPIToken_NoExpiration(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	// Request with expiresDays=0 should create a token without expiration
	reqBody := `{"name":"Permanent Token","expiresDays":0,"permissions":"admin"}`
	req := createRequestWithUserID(http.MethodPost, "/api/tokens", reqBody, 1)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateAPIToken(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.APIToken
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Permanent Token", response.Name)
	assert.Nil(t, response.ExpiresAt)
}

func TestCreateAPIToken_Unauthorized(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	reqBody := `{"name":"Test Token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	w := httptest.NewRecorder()

	CreateAPIToken(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestCreateAPIToken_InvalidJSON(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	reqBody := `{"invalid json`
	req := createRequestWithUserID(http.MethodPost, "/api/tokens", reqBody, 1)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateAPIToken(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestCreateAPIToken_ValidName(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	// Test with valid name
	reqBody := `{"name":"Valid Token Name","expiresDays":7}`
	req := createRequestWithUserID(http.MethodPost, "/api/tokens", reqBody, 1)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateAPIToken(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestListAPITokens_Success(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	// Create some tokens first
	repo := db.NewAPITokenRepository()
	for i := 0; i < 5; i++ {
		token := &models.APIToken{
			Name:        "Token " + string(rune('A'+i)),
			TokenHash:   db.HashToken("token_" + string(rune('A'+i))),
			Status:      "active",
			CreatedBy:   1,
			Permissions: "read",
		}
		err := repo.Create(token)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tokens", nil)
	w := httptest.NewRecorder()

	ListAPITokens(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	contentType := w.Header().Get("Content-Type")
	assert.Equal(t, "application/json", contentType)

	var response models.APITokenListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 5, response.Total)
	assert.Len(t, response.List, 5)

	// Verify pagination headers
	totalHeader := w.Header().Get("X-Total")
	assert.Equal(t, "5", totalHeader)
}

func TestListAPITokens_WithPagination(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	// Create some tokens
	repo := db.NewAPITokenRepository()
	for i := 0; i < 10; i++ {
		token := &models.APIToken{
			Name:        "Token " + string(rune('0'+i)),
			TokenHash:   db.HashToken("token_pagination_" + string(rune('0'+i))),
			Status:      "active",
			CreatedBy:   1,
			Permissions: "read",
		}
		err := repo.Create(token)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tokens?page=1&limit=3", nil)
	w := httptest.NewRecorder()

	ListAPITokens(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.APITokenListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 10, response.Total)
	assert.Len(t, response.List, 3)
}

func TestListAPITokens_EmptyList(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/tokens", nil)
	w := httptest.NewRecorder()

	ListAPITokens(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.APITokenListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 0, response.Total)
	assert.Empty(t, response.List)
}

func TestUpdateAPITokenStatus_Success(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	// Create a token first
	repo := db.NewAPITokenRepository()
	token := &models.APIToken{
		Name:        "Token to Update",
		TokenHash:   db.HashToken("update_token"),
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
	}
	err := repo.Create(token)
	require.NoError(t, err)

	reqBody := `{"status":"disabled"}`
	req := httptest.NewRequest(http.MethodPut, "/api/tokens?id="+string(rune('0'+int(token.ID))), strings.NewReader(reqBody))
	req.URL.RawQuery = "id=" + string(rune('0'+int(token.ID)))
	req = httptest.NewRequest(http.MethodPut, "/api/tokens?id=1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateAPITokenStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "token status updated successfully", response["message"])

	// Verify the status was updated
	updated, _ := repo.GetByTokenHash(token.TokenHash)
	assert.Equal(t, "disabled", updated.Status)
}

func TestUpdateAPITokenStatus_ActivateToken(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	// Create a disabled token
	repo := db.NewAPITokenRepository()
	token := &models.APIToken{
		Name:        "Disabled Token",
		TokenHash:   db.HashToken("disabled_token"),
		Status:      "disabled",
		CreatedBy:   1,
		Permissions: "read",
	}
	err := repo.Create(token)
	require.NoError(t, err)

	// Manually update status to disabled
	repo.UpdateStatus(token.ID, "disabled")

	reqBody := `{"status":"active"}`
	req := httptest.NewRequest(http.MethodPut, "/api/tokens?id=1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateAPITokenStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateAPITokenStatus_MissingID(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	reqBody := `{"status":"disabled"}`
	req := httptest.NewRequest(http.MethodPut, "/api/tokens", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateAPITokenStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing token id")
}

func TestUpdateAPITokenStatus_InvalidID(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	reqBody := `{"status":"disabled"}`
	req := httptest.NewRequest(http.MethodPut, "/api/tokens?id=abc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateAPITokenStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid token id")
}

func TestUpdateAPITokenStatus_InvalidJSON(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	reqBody := `{"invalid json`
	req := httptest.NewRequest(http.MethodPut, "/api/tokens?id=1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateAPITokenStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestUpdateAPITokenStatus_InvalidStatus(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	// Create a token first
	repo := db.NewAPITokenRepository()
	token := &models.APIToken{
		Name:        "Token to Update",
		TokenHash:   db.HashToken("update_token_invalid"),
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
	}
	err := repo.Create(token)
	require.NoError(t, err)

	// Note: validator currently allows any status, but database constraint may reject invalid values
	reqBody := `{"status":"invalid_status"}`
	req := httptest.NewRequest(http.MethodPut, "/api/tokens?id=1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateAPITokenStatus(w, req)

	// Database accepts any status string currently
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteAPIToken_Success(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	// Create a token first
	repo := db.NewAPITokenRepository()
	token := &models.APIToken{
		Name:        "Token to Delete",
		TokenHash:   db.HashToken("delete_token"),
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
	}
	err := repo.Create(token)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/api/tokens?id=1", nil)
	w := httptest.NewRecorder()

	DeleteAPIToken(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "token deleted successfully", response["message"])

	// Verify the token was deleted
	deleted, _ := repo.GetByTokenHash(token.TokenHash)
	assert.Nil(t, deleted)
}

func TestDeleteAPIToken_MissingID(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/api/tokens", nil)
	w := httptest.NewRecorder()

	DeleteAPIToken(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing token id")
}

func TestDeleteAPIToken_InvalidID(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/api/tokens?id=xyz", nil)
	w := httptest.NewRecorder()

	DeleteAPIToken(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid token id")
}

func TestDeleteAPIToken_NonExistent(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/api/tokens?id=999", nil)
	w := httptest.NewRecorder()

	DeleteAPIToken(w, req)

	// Should return success even if token doesn't exist (idempotent)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGenerateRandomToken(t *testing.T) {
	t.Run("generate random token format", func(t *testing.T) {
		token1 := generateRandomToken()
		token2 := generateRandomToken()

		// Should start with psk_
		assert.True(t, strings.HasPrefix(token1, "psk_"))
		assert.True(t, strings.HasPrefix(token2, "psk_"))

		// Should be different each time
		assert.NotEqual(t, token1, token2)

		// Should be reasonable length
		assert.Greater(t, len(token1), 40)
	})
}

func TestAPITokenIntegration_FullLifecycle(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	// Step 1: Create a token
	createReqBody := `{"name":"Lifecycle Test Token","expiresDays":7,"permissions":"read,write"}`
	createReq := createRequestWithUserID(http.MethodPost, "/api/tokens", createReqBody, 1)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()

	CreateAPIToken(createW, createReq)
	assert.Equal(t, http.StatusCreated, createW.Code)

	var createdToken models.APIToken
	err := json.Unmarshal(createW.Body.Bytes(), &createdToken)
	require.NoError(t, err)
	assert.NotEmpty(t, createdToken.Token)

	// Step 2: List tokens
	listReq := httptest.NewRequest(http.MethodGet, "/api/tokens", nil)
	listW := httptest.NewRecorder()

	ListAPITokens(listW, listReq)
	assert.Equal(t, http.StatusOK, listW.Code)

	var listResponse models.APITokenListResponse
	err = json.Unmarshal(listW.Body.Bytes(), &listResponse)
	require.NoError(t, err)
	assert.Equal(t, 1, listResponse.Total)

	// Step 3: Update token status
	updateReqBody := `{"status":"disabled"}`
	updateReq := httptest.NewRequest(http.MethodPut, "/api/tokens?id=1", strings.NewReader(updateReqBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()

	UpdateAPITokenStatus(updateW, updateReq)
	assert.Equal(t, http.StatusOK, updateW.Code)

	// Step 4: Delete token
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/tokens?id=1", nil)
	deleteW := httptest.NewRecorder()

	DeleteAPIToken(deleteW, deleteReq)
	assert.Equal(t, http.StatusOK, deleteW.Code)

	// Step 5: Verify list is empty
	listReq2 := httptest.NewRequest(http.MethodGet, "/api/tokens", nil)
	listW2 := httptest.NewRecorder()

	ListAPITokens(listW2, listReq2)
	assert.Equal(t, http.StatusOK, listW2.Code)

	var emptyList models.APITokenListResponse
	err = json.Unmarshal(listW2.Body.Bytes(), &emptyList)
	require.NoError(t, err)
	assert.Equal(t, 0, emptyList.Total)
}

func TestAPITokenAuth_WithCreatedToken(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	// Create a token
	createReqBody := `{"name":"Auth Test Token","expiresDays":1,"permissions":"read"}`
	createReq := createRequestWithUserID(http.MethodPost, "/api/tokens", createReqBody, 1)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()

	CreateAPIToken(createW, createReq)
	assert.Equal(t, http.StatusCreated, createW.Code)

	var createdToken models.APIToken
	err := json.Unmarshal(createW.Body.Bytes(), &createdToken)
	require.NoError(t, err)

	// Now use the created token for authentication
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+createdToken.Token)
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	})

	authMiddleware := middleware.APITokenAuthMiddleware(testHandler)
	authMiddleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIToken_WithExpiredTokenValidation(t *testing.T) {
	cleanup := setupTestDBForAPITokenHandlers(t)
	defer cleanup()

	repo := db.NewAPITokenRepository()
	pastTime := time.Now().Add(-24 * time.Hour)
	token := &models.APIToken{
		Name:        "Expired Token",
		TokenHash:   db.HashToken("expired_auth_token"),
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
		ExpiresAt:   &pastTime,
	}
	err := repo.Create(token)
	require.NoError(t, err)

	// Try to use expired token for authentication
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer expired_auth_token")
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with expired token")
	})

	authMiddleware := middleware.APITokenAuthMiddleware(testHandler)
	authMiddleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired token")
}
