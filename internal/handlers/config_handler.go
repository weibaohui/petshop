package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"

	"petshop/internal/models"
)

// System configuration functions

// systemConfigs holds system configuration in memory
var systemConfigs = make(map[string]string)
var configMu sync.RWMutex

// SetSystemConfigRequest represents the request body for setting a system config.
type SetSystemConfigRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func init() {
	// Initialize default configs
	configMu.Lock()
	systemConfigs["site_name"] = "宠物商店"
	systemConfigs["inventory_threshold"] = "10"
	configMu.Unlock()
}

// GetSystemConfigs handles GET /api/admin/configs and returns all system configurations.
func GetSystemConfigs(w http.ResponseWriter, r *http.Request) {
	configMu.RLock()
	defer configMu.RUnlock()

	configs := make([]models.SystemConfig, 0, len(systemConfigs))
	for k, v := range systemConfigs {
		configs = append(configs, models.SystemConfig{Key: k, Value: v})
	}
	_ = json.NewEncoder(w).Encode(configs)
}

// SetSystemConfig handles POST /api/admin/config and sets a system configuration value.
// If the key is "inventory_threshold", it updates the inventory alert threshold.
func SetSystemConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req SetSystemConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	if req.Key == "inventory_threshold" {
		v, err := strconv.Atoi(req.Value)
		if err != nil {
			http.Error(w, "inventory_threshold must be a valid integer", http.StatusBadRequest)
			return
		}
		if v < 0 {
			http.Error(w, "inventory_threshold must be non-negative", http.StatusBadRequest)
			return
		}
	}

	configMu.Lock()
	defer configMu.Unlock()

	systemConfigs[req.Key] = req.Value

	// Update inventory alert threshold
	if req.Key == "inventory_threshold" {
		v, _ := strconv.Atoi(req.Value)
		dataMu.Lock()
		inventoryThreshold = v
		dataMu.Unlock()
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "config updated"})
}
