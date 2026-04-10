package db

import (
	"os"
	"testing"
	"time"

	"petshop/internal/models"
)

func setupTestDBForAPITokens(t *testing.T) func() {
	// Reset database state
	ResetForTesting()

	// Use test database file
	testDBPath := "./test_api_tokens.db"

	// Remove existing test database
	os.Remove(testDBPath)

	// Initialize database
	err := InitDB(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Cleanup function
	return func() {
		_ = Close()
		os.Remove(testDBPath)
		ResetForTesting()
	}
}

func TestHashToken(t *testing.T) {
	token := "test_token_123"
	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 != hash2 {
		t.Error("HashToken should return consistent results for same input")
	}

	if len(hash1) != 64 {
		t.Errorf("Expected SHA256 hash length of 64, got %d", len(hash1))
	}

	// Different tokens should have different hashes
	hash3 := HashToken("different_token")
	if hash1 == hash3 {
		t.Error("Different tokens should have different hashes")
	}
}

func TestAPITokenRepository_Create(t *testing.T) {
	cleanup := setupTestDBForAPITokens(t)
	defer cleanup()

	repo := NewAPITokenRepository()

	token := &models.APIToken{
		Name:        "Test Token",
		TokenHash:   HashToken("test_token_value"),
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
	}

	err := repo.Create(token)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	if token.ID == 0 {
		t.Error("Expected token ID to be set after creation")
	}
}

func TestAPITokenRepository_GetByTokenHash(t *testing.T) {
	cleanup := setupTestDBForAPITokens(t)
	defer cleanup()

	repo := NewAPITokenRepository()

	// Create a token
	tokenValue := "test_token_for_get"
	tokenHash := HashToken(tokenValue)
	token := &models.APIToken{
		Name:        "Get Test Token",
		TokenHash:   tokenHash,
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read,write",
	}

	err := repo.Create(token)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Retrieve by hash
	retrieved, err := repo.GetByTokenHash(tokenHash)
	if err != nil {
		t.Fatalf("Failed to get token by hash: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to find token, got nil")
	}

	if retrieved.Name != token.Name {
		t.Errorf("Expected name %s, got %s", token.Name, retrieved.Name)
	}

	if retrieved.Permissions != token.Permissions {
		t.Errorf("Expected permissions %s, got %s", token.Permissions, retrieved.Permissions)
	}

	// Try to get non-existent token
	nonExistent, err := repo.GetByTokenHash(HashToken("non_existent"))
	if err != nil {
		t.Fatalf("Expected no error for non-existent token: %v", err)
	}
	if nonExistent != nil {
		t.Error("Expected nil for non-existent token")
	}
}

func TestAPITokenRepository_List(t *testing.T) {
	cleanup := setupTestDBForAPITokens(t)
	defer cleanup()

	repo := NewAPITokenRepository()

	// Create multiple tokens
	for i := 0; i < 5; i++ {
		token := &models.APIToken{
			Name:        "Token " + string(rune('A'+i)),
			TokenHash:   HashToken("token_" + string(rune('A'+i))),
			Status:      "active",
			CreatedBy:   1,
			Permissions: "read",
		}
		err := repo.Create(token)
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}
	}

	// Test listing
	tokens, total, err := repo.List(0, 10)
	if err != nil {
		t.Fatalf("Failed to list tokens: %v", err)
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}

	if len(tokens) != 5 {
		t.Errorf("Expected 5 tokens, got %d", len(tokens))
	}

	// Test pagination
	tokens, total, err = repo.List(0, 2)
	if err != nil {
		t.Fatalf("Failed to list tokens with pagination: %v", err)
	}

	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(tokens))
	}
}

func TestAPITokenRepository_UpdateStatus(t *testing.T) {
	cleanup := setupTestDBForAPITokens(t)
	defer cleanup()

	repo := NewAPITokenRepository()

	// Create a token
	token := &models.APIToken{
		Name:        "Status Test Token",
		TokenHash:   HashToken("status_test"),
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
	}

	err := repo.Create(token)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Update status
	err = repo.UpdateStatus(token.ID, "disabled")
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Verify update
	retrieved, _ := repo.GetByTokenHash(token.TokenHash)
	if retrieved.Status != "disabled" {
		t.Errorf("Expected status 'disabled', got '%s'", retrieved.Status)
	}
}

func TestAPITokenRepository_Delete(t *testing.T) {
	cleanup := setupTestDBForAPITokens(t)
	defer cleanup()

	repo := NewAPITokenRepository()

	// Create a token
	token := &models.APIToken{
		Name:        "Delete Test Token",
		TokenHash:   HashToken("delete_test"),
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
	}

	err := repo.Create(token)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Delete token
	err = repo.Delete(token.ID)
	if err != nil {
		t.Fatalf("Failed to delete token: %v", err)
	}

	// Verify deletion
	retrieved, _ := repo.GetByTokenHash(token.TokenHash)
	if retrieved != nil {
		t.Error("Expected token to be deleted")
	}
}

func TestAPITokenRepository_UpdateLastUsedAt(t *testing.T) {
	cleanup := setupTestDBForAPITokens(t)
	defer cleanup()

	repo := NewAPITokenRepository()

	// Create a token
	token := &models.APIToken{
		Name:        "Last Used Test Token",
		TokenHash:   HashToken("last_used_test"),
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
	}

	err := repo.Create(token)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Update last used time
	err = repo.UpdateLastUsedAt(token.ID)
	if err != nil {
		t.Fatalf("Failed to update last used time: %v", err)
	}

	// Verify update
	retrieved, _ := repo.GetByTokenHash(token.TokenHash)
	if retrieved.LastUsedAt == nil {
		t.Error("Expected LastUsedAt to be set")
	}
}

func TestAPITokenRepository_IsTokenValid(t *testing.T) {
	cleanup := setupTestDBForAPITokens(t)
	defer cleanup()

	repo := NewAPITokenRepository()

	// Test 1: Active token without expiration
	tokenValue := "valid_active_token"
	tokenHash := HashToken(tokenValue)
	activeToken := &models.APIToken{
		Name:        "Valid Active Token",
		TokenHash:   tokenHash,
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
	}

	err := repo.Create(activeToken)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	valid, retrieved := repo.IsTokenValid(tokenHash)
	if !valid {
		t.Error("Expected active token without expiration to be valid")
	}
	if retrieved == nil {
		t.Error("Expected retrieved token to not be nil")
	}

	// Test 2: Disabled token
	disabledTokenValue := "disabled_token"
	disabledTokenHash := HashToken(disabledTokenValue)
	disabledToken := &models.APIToken{
		Name:        "Disabled Token",
		TokenHash:   disabledTokenHash,
		Status:      "disabled",
		CreatedBy:   1,
		Permissions: "read",
	}

	err = repo.Create(disabledToken)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	valid, _ = repo.IsTokenValid(disabledTokenHash)
	if valid {
		t.Error("Expected disabled token to be invalid")
	}

	// Test 3: Expired token
	expiredTokenValue := "expired_token"
	expiredTokenHash := HashToken(expiredTokenValue)
	pastTime := time.Now().Add(-24 * time.Hour)
	expiredToken := &models.APIToken{
		Name:        "Expired Token",
		TokenHash:   expiredTokenHash,
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
		ExpiresAt:   &pastTime,
	}

	err = repo.Create(expiredToken)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	valid, _ = repo.IsTokenValid(expiredTokenHash)
	if valid {
		t.Error("Expected expired token to be invalid")
	}

	// Test 4: Non-existent token
	valid, _ = repo.IsTokenValid(HashToken("non_existent_token"))
	if valid {
		t.Error("Expected non-existent token to be invalid")
	}
}

func TestAPITokenRepository_TokenWithExpiration(t *testing.T) {
	cleanup := setupTestDBForAPITokens(t)
	defer cleanup()

	repo := NewAPITokenRepository()

	// Create a token with future expiration
	tokenValue := "expiring_token"
	tokenHash := HashToken(tokenValue)
	futureTime := time.Now().Add(24 * time.Hour)
	token := &models.APIToken{
		Name:        "Expiring Token",
		TokenHash:   tokenHash,
		Status:      "active",
		CreatedBy:   1,
		Permissions: "read",
		ExpiresAt:   &futureTime,
	}

	err := repo.Create(token)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Verify token is valid
	valid, _ := repo.IsTokenValid(tokenHash)
	if !valid {
		t.Error("Expected non-expired token to be valid")
	}

	// Verify ExpiresAt is preserved
	retrieved, _ := repo.GetByTokenHash(tokenHash)
	if retrieved.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}
}
