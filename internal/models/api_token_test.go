package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAPITokenStruct(t *testing.T) {
	t.Run("APIToken JSON serialization and deserialization", func(t *testing.T) {
		expiresAt := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
		lastUsedAt := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
		token := APIToken{
			ID:          1,
			Name:        "Test API Token",
			Token:       "psk_test_token_123",
			TokenHash:   "abc123hash",
			Status:      "active",
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			LastUsedAt:  &lastUsedAt,
			ExpiresAt:   &expiresAt,
			CreatedBy:   100,
			Permissions: "read,write",
		}

		data, err := json.Marshal(token)
		assert.NoError(t, err)

		var decoded APIToken
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, token.ID, decoded.ID)
		assert.Equal(t, token.Name, decoded.Name)
		assert.Equal(t, token.Token, decoded.Token)
		assert.Empty(t, decoded.TokenHash, "TokenHash should not be serialized to JSON")
		assert.Equal(t, token.Status, decoded.Status)
		assert.Equal(t, token.CreatedAt, decoded.CreatedAt)
		assert.Equal(t, token.UpdatedAt, decoded.UpdatedAt)
		assert.Equal(t, token.LastUsedAt, decoded.LastUsedAt)
		assert.Equal(t, token.ExpiresAt, decoded.ExpiresAt)
		assert.Equal(t, token.CreatedBy, decoded.CreatedBy)
		assert.Equal(t, token.Permissions, decoded.Permissions)
	})

	t.Run("APIToken with nil optional fields", func(t *testing.T) {
		token := APIToken{
			ID:          2,
			Name:        "Token without expiration",
			Token:       "psk_no_expiry",
			TokenHash:   "def456hash",
			Status:      "active",
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			LastUsedAt:  nil,
			ExpiresAt:   nil,
			CreatedBy:   101,
			Permissions: "read",
		}

		data, err := json.Marshal(token)
		assert.NoError(t, err)

		var decoded APIToken
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Nil(t, decoded.LastUsedAt)
		assert.Nil(t, decoded.ExpiresAt)
	})

	t.Run("APIToken JSON deserialization from JSON string", func(t *testing.T) {
		jsonStr := `{"id":3,"name":"Production Token","token":"psk_prod_456","status":"disabled","createdAt":"2024-03-01T00:00:00Z","updatedAt":"2024-03-01T00:00:00Z","createdBy":102,"permissions":"admin"}`

		var token APIToken
		err := json.Unmarshal([]byte(jsonStr), &token)
		assert.NoError(t, err)

		assert.Equal(t, int64(3), token.ID)
		assert.Equal(t, "Production Token", token.Name)
		assert.Equal(t, "psk_prod_456", token.Token)
		assert.Equal(t, "disabled", token.Status)
		assert.Equal(t, int64(102), token.CreatedBy)
		assert.Equal(t, "admin", token.Permissions)
	})

	t.Run("APIToken disabled status", func(t *testing.T) {
		token := APIToken{
			ID:          4,
			Name:        "Disabled Token",
			Token:       "psk_disabled_789",
			Status:      "disabled",
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			CreatedBy:   103,
			Permissions: "read",
		}

		assert.Equal(t, "disabled", token.Status)
	})
}

func TestAPITokenCreateRequestStruct(t *testing.T) {
	t.Run("APITokenCreateRequest JSON serialization", func(t *testing.T) {
		req := APITokenCreateRequest{
			Name:        "New Token",
			ExpiresDays: 30,
			Permissions: "read,write",
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)

		var decoded APITokenCreateRequest
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, req.Name, decoded.Name)
		assert.Equal(t, req.ExpiresDays, decoded.ExpiresDays)
		assert.Equal(t, req.Permissions, decoded.Permissions)
	})

	t.Run("APITokenCreateRequest with zero expires days", func(t *testing.T) {
		req := APITokenCreateRequest{
			Name:        "Permanent Token",
			ExpiresDays: 0,
			Permissions: "read",
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)

		var decoded APITokenCreateRequest
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, 0, decoded.ExpiresDays)
	})

	t.Run("APITokenCreateRequest JSON deserialization", func(t *testing.T) {
		jsonStr := `{"name":"Test Token","expiresDays":90,"permissions":"admin"}`

		var req APITokenCreateRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		assert.NoError(t, err)

		assert.Equal(t, "Test Token", req.Name)
		assert.Equal(t, 90, req.ExpiresDays)
		assert.Equal(t, "admin", req.Permissions)
	})

	t.Run("APITokenCreateRequest with empty permissions", func(t *testing.T) {
		req := APITokenCreateRequest{
			Name:        "Basic Token",
			ExpiresDays: 7,
			Permissions: "",
		}

		assert.Empty(t, req.Permissions)
	})
}

func TestAPITokenListResponseStruct(t *testing.T) {
	t.Run("APITokenListResponse JSON serialization", func(t *testing.T) {
		response := APITokenListResponse{
			List: []APIToken{
				{
					ID:          1,
					Name:        "Token 1",
					Status:      "active",
					CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					CreatedBy:   100,
					Permissions: "read",
				},
				{
					ID:          2,
					Name:        "Token 2",
					Status:      "disabled",
					CreatedAt:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
					CreatedBy:   101,
					Permissions: "write",
				},
			},
			Total: 2,
		}

		data, err := json.Marshal(response)
		assert.NoError(t, err)

		var decoded APITokenListResponse
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, 2, decoded.Total)
		assert.Len(t, decoded.List, 2)
		assert.Equal(t, "Token 1", decoded.List[0].Name)
		assert.Equal(t, "Token 2", decoded.List[1].Name)
	})

	t.Run("APITokenListResponse empty list", func(t *testing.T) {
		response := APITokenListResponse{
			List:  []APIToken{},
			Total: 0,
		}

		data, err := json.Marshal(response)
		assert.NoError(t, err)

		var decoded APITokenListResponse
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, 0, decoded.Total)
		assert.Empty(t, decoded.List)
	})

	t.Run("APITokenListResponse JSON deserialization", func(t *testing.T) {
		jsonStr := `{"list":[{"id":1,"name":"Test","status":"active","createdAt":"2024-01-01T00:00:00Z","updatedAt":"2024-01-01T00:00:00Z","createdBy":1,"permissions":"read"}],"total":1}`

		var response APITokenListResponse
		err := json.Unmarshal([]byte(jsonStr), &response)
		assert.NoError(t, err)

		assert.Equal(t, 1, response.Total)
		assert.Len(t, response.List, 1)
		assert.Equal(t, int64(1), response.List[0].ID)
	})
}

func TestAPITokenStatusUpdateRequestStruct(t *testing.T) {
	t.Run("APITokenStatusUpdateRequest JSON serialization", func(t *testing.T) {
		req := APITokenStatusUpdateRequest{
			Status: "disabled",
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)

		var decoded APITokenStatusUpdateRequest
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, req.Status, decoded.Status)
	})

	t.Run("APITokenStatusUpdateRequest JSON deserialization", func(t *testing.T) {
		jsonStr := `{"status":"active"}`

		var req APITokenStatusUpdateRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		assert.NoError(t, err)

		assert.Equal(t, "active", req.Status)
	})

	t.Run("APITokenStatusUpdateRequest with invalid status", func(t *testing.T) {
		// Note: validation is handled by the validator, not the struct itself
		req := APITokenStatusUpdateRequest{
			Status: "invalid_status",
		}

		assert.Equal(t, "invalid_status", req.Status)
	})
}
