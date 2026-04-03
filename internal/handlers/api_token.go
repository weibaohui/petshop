package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"petshop/internal/db"
	"petshop/internal/logger"
	"petshop/internal/models"
	"petshop/internal/pagination"
)

// apiTokenRepo is the repository for API tokens
var apiTokenRepo = db.NewAPITokenRepository()

// generateToken generates a secure random token
func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "ps_" + hex.EncodeToString(bytes)
}

// CreateAPIToken creates a new API token
func CreateAPIToken(w http.ResponseWriter, r *http.Request) {
	var req models.APITokenCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("failed to decode create token request", map[string]interface{}{
			"error": err.Error(),
		})
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Manual validation
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(req.Name) > 100 {
		writeError(w, http.StatusBadRequest, "name must be less than 100 characters")
		return
	}
	if len(req.Description) > 500 {
		writeError(w, http.StatusBadRequest, "description must be less than 500 characters")
		return
	}

	// Generate token
	tokenValue := generateToken()

	token := &models.APIToken{
		Name:        req.Name,
		Token:       tokenValue,
		Description: req.Description,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set expiration if specified
	if req.ExpiresDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, req.ExpiresDays)
		token.ExpiresAt = &expiresAt
	}

	if err := apiTokenRepo.Create(token); err != nil {
		logger.Error("failed to create api token", map[string]interface{}{
			"error": err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	logger.Info("api token created", map[string]interface{}{
		"token_id": token.ID,
		"name":     token.Name,
	})

	writeJSON(w, http.StatusCreated, token.ToCreateResponse())
}

// ListAPITokens lists all API tokens
func ListAPITokens(w http.ResponseWriter, r *http.Request) {
	pageInfo := pagination.ParsePagination(r)
	offset := (pageInfo.Page - 1) * pageInfo.PageSize

	tokens, total, err := apiTokenRepo.List(pageInfo.PageSize, offset)
	if err != nil {
		logger.Error("failed to list api tokens", map[string]interface{}{
			"error": err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "failed to list tokens")
		return
	}

	// Convert to response format
	responses := make([]models.APITokenResponse, len(tokens))
	for i, t := range tokens {
		responses[i] = t.ToResponse()
	}

	// Update pagination info with total
	pageInfo.Total = int(total)
	pageInfo.TotalPages = (int(total) + pageInfo.PageSize - 1) / pageInfo.PageSize

	writeJSON(w, http.StatusOK, pagination.NewPagedResponse(responses, pageInfo))
}

// UpdateAPITokenStatus updates the status of an API token
func UpdateAPITokenStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeError(w, http.StatusBadRequest, "missing id parameter")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id parameter")
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Manual validation
	if req.Status != "active" && req.Status != "disabled" {
		writeError(w, http.StatusBadRequest, "status must be either 'active' or 'disabled'")
		return
	}

	// Check if token exists
	token, err := apiTokenRepo.GetByID(id)
	if err != nil {
		logger.Error("failed to get api token", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		writeError(w, http.StatusInternalServerError, "failed to get token")
		return
	}
	if token == nil {
		writeError(w, http.StatusNotFound, "token not found")
		return
	}

	if err := apiTokenRepo.UpdateStatus(id, req.Status); err != nil {
		logger.Error("failed to update api token status", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		writeError(w, http.StatusInternalServerError, "failed to update token status")
		return
	}

	logger.Info("api token status updated", map[string]interface{}{
		"token_id": id,
		"status":   req.Status,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "token status updated",
		"id":      id,
		"status":  req.Status,
	})
}

// DeleteAPIToken deletes an API token
func DeleteAPIToken(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeError(w, http.StatusBadRequest, "missing id parameter")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id parameter")
		return
	}

	// Check if token exists
	token, err := apiTokenRepo.GetByID(id)
	if err != nil {
		logger.Error("failed to get api token", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		writeError(w, http.StatusInternalServerError, "failed to get token")
		return
	}
	if token == nil {
		writeError(w, http.StatusNotFound, "token not found")
		return
	}

	if err := apiTokenRepo.Delete(id); err != nil {
		logger.Error("failed to delete api token", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		writeError(w, http.StatusInternalServerError, "failed to delete token")
		return
	}

	logger.Info("api token deleted", map[string]interface{}{
		"token_id": id,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "token deleted",
		"id":      id,
	})
}
