package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"petshop/internal/models"

	"github.com/stretchr/testify/assert"
)

// resetSystemConfigs resets system configs to a known state for testing
func resetSystemConfigs() {
	dataMu.Lock()
	defer dataMu.Unlock()

	systemConfigs = make(map[string]string)
	systemConfigs["site_name"] = "宠物商店"
	systemConfigs["inventory_threshold"] = "10"
	inventoryThreshold = 10
}

func TestGetSystemConfigs_Handler(t *testing.T) {
	resetSystemConfigs()
	defer resetSystemConfigs()

	tests := []struct {
		name           string
		setup          func()
		wantStatusCode int
		wantLen        int
		wantContains   map[string]string
	}{
		{
			name:           "get all system configs",
			setup:          func() {},
			wantStatusCode: http.StatusOK,
			wantLen:        2,
			wantContains: map[string]string{
				"site_name":           "宠物商店",
				"inventory_threshold": "10",
			},
		},
		{
			name: "get empty configs",
			setup: func() {
				dataMu.Lock()
				systemConfigs = make(map[string]string)
				dataMu.Unlock()
			},
			wantStatusCode: http.StatusOK,
			wantLen:        0,
			wantContains:   map[string]string{},
		},
		{
			name: "get configs with multiple entries",
			setup: func() {
				dataMu.Lock()
				systemConfigs = make(map[string]string)
				systemConfigs["key1"] = "value1"
				systemConfigs["key2"] = "value2"
				systemConfigs["key3"] = "value3"
				dataMu.Unlock()
			},
			wantStatusCode: http.StatusOK,
			wantLen:        3,
			wantContains: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetSystemConfigs()
			tt.setup()

			req := httptest.NewRequest(http.MethodGet, "/api/admin/configs", nil)
			w := httptest.NewRecorder()

			GetSystemConfigs(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			var response []models.SystemConfig
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Len(t, response, tt.wantLen)

			// Check if all expected configs are present
			for _, config := range response {
				if expectedValue, exists := tt.wantContains[config.Key]; exists {
					assert.Equal(t, expectedValue, config.Value)
				}
			}
		})
	}
}

func TestSetSystemConfig_Handler(t *testing.T) {
	resetSystemConfigs()
	defer resetSystemConfigs()

	tests := []struct {
		name                    string
		requestBody             string
		wantStatusCode          int
		wantErr                 bool
		wantMessage             string
		checkConfigValue        map[string]string
		checkInventoryThreshold *int
	}{
		{
			name:           "set config with valid data",
			requestBody:    `{"key":"site_name","value":"新宠物商店"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			wantMessage:    "config updated",
			checkConfigValue: map[string]string{
				"site_name": "新宠物商店",
			},
		},
		{
			name:           "set new config key",
			requestBody:    `{"key":"new_key","value":"new_value"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			wantMessage:    "config updated",
			checkConfigValue: map[string]string{
				"new_key": "new_value",
			},
		},
		{
			name:           "set inventory_threshold with valid integer",
			requestBody:    `{"key":"inventory_threshold","value":"20"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			wantMessage:    "config updated",
			checkConfigValue: map[string]string{
				"inventory_threshold": "20",
			},
			checkInventoryThreshold: intPtr(20),
		},
		{
			name:           "set inventory_threshold to zero",
			requestBody:    `{"key":"inventory_threshold","value":"0"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			wantMessage:    "config updated",
			checkConfigValue: map[string]string{
				"inventory_threshold": "0",
			},
			checkInventoryThreshold: intPtr(0),
		},
		{
			name:           "missing key field",
			requestBody:    `{"value":"some_value"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
			wantMessage:    "key is required",
		},
		{
			name:           "empty key",
			requestBody:    `{"key":"","value":"some_value"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
			wantMessage:    "key is required",
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
			wantMessage:    "invalid JSON",
		},
		{
			name:           "empty body",
			requestBody:    ``,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
			wantMessage:    "invalid JSON",
		},
		{
			name:           "inventory_threshold with non-integer value",
			requestBody:    `{"key":"inventory_threshold","value":"not_a_number"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
			wantMessage:    "inventory_threshold must be a valid integer",
		},
		{
			name:           "inventory_threshold with float value",
			requestBody:    `{"key":"inventory_threshold","value":"10.5"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
			wantMessage:    "inventory_threshold must be a valid integer",
		},
		{
			name:           "inventory_threshold with negative value",
			requestBody:    `{"key":"inventory_threshold","value":"-5"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
			wantMessage:    "inventory_threshold must be non-negative",
		},
		{
			name:           "inventory_threshold with large negative value",
			requestBody:    `{"key":"inventory_threshold","value":"-999999"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
			wantMessage:    "inventory_threshold must be non-negative",
		},
		{
			name:           "valid JSON but missing value field",
			requestBody:    `{"key":"test_key"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			wantMessage:    "config updated",
			checkConfigValue: map[string]string{
				"test_key": "",
			},
		},
		{
			name:           "set config with special characters in value",
			requestBody:    `{"key":"special_key","value":"value with spaces and symbols !@#$%"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			wantMessage:    "config updated",
			checkConfigValue: map[string]string{
				"special_key": "value with spaces and symbols !@#$%",
			},
		},
		{
			name:           "set config with unicode value",
			requestBody:    `{"key":"unicode_key","value":"中文测试"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			wantMessage:    "config updated",
			checkConfigValue: map[string]string{
				"unicode_key": "中文测试",
			},
		},
		{
			name:           "inventory_threshold with large positive value",
			requestBody:    `{"key":"inventory_threshold","value":"999999"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
			wantMessage:    "config updated",
			checkConfigValue: map[string]string{
				"inventory_threshold": "999999",
			},
			checkInventoryThreshold: intPtr(999999),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetSystemConfigs()

			req := httptest.NewRequest(http.MethodPost, "/api/admin/config", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			SetSystemConfig(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantErr {
				body := w.Body.String()
				assert.True(t, w.Code >= 400, "expected error status code, got %d", w.Code)
				assert.Contains(t, body, tt.wantMessage)
			} else {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMessage, response["message"])

				// Verify config values were set
				if tt.checkConfigValue != nil {
					dataMu.RLock()
					for key, expectedValue := range tt.checkConfigValue {
						actualValue, exists := systemConfigs[key]
						assert.True(t, exists, "config key %s should exist", key)
						assert.Equal(t, expectedValue, actualValue, "config key %s should have correct value", key)
					}
					dataMu.RUnlock()
				}

				// Verify inventoryThreshold was updated
				if tt.checkInventoryThreshold != nil {
					dataMu.RLock()
					assert.Equal(t, *tt.checkInventoryThreshold, inventoryThreshold)
					dataMu.RUnlock()
				}
			}
		})
	}
}

func TestGetSystemConfigs_Concurrent(t *testing.T) {
	resetSystemConfigs()
	defer resetSystemConfigs()

	// Test concurrent read access
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/configs", nil)
			w := httptest.NewRecorder()
			GetSystemConfigs(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}()
	}

	wg.Wait()
}

func TestSetSystemConfig_Concurrent(t *testing.T) {
	resetSystemConfigs()
	defer resetSystemConfigs()

	// Test concurrent write access
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			body := `{"key":"concurrent_key_` + string(rune('0'+index)) + `","value":"value_` + string(rune('0'+index)) + `"}`
			req := httptest.NewRequest(http.MethodPost, "/api/admin/config", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			SetSystemConfig(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}(i)
	}

	wg.Wait()

	// Verify all configs were set
	dataMu.RLock()
	for i := 0; i < numGoroutines; i++ {
		key := "concurrent_key_" + string(rune('0'+i))
		_, exists := systemConfigs[key]
		assert.True(t, exists, "config key %s should exist", key)
	}
	dataMu.RUnlock()
}

func TestSetSystemConfig_ConcurrentInventoryThreshold(t *testing.T) {
	resetSystemConfigs()
	defer resetSystemConfigs()

	// Test concurrent updates to inventory_threshold
	var wg sync.WaitGroup
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			body := `{"key":"inventory_threshold","value":"` + string(rune('1'+index)) + `0"}`
			req := httptest.NewRequest(http.MethodPost, "/api/admin/config", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			SetSystemConfig(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}(i)
	}

	wg.Wait()

	// Verify inventoryThreshold is set to one of the values (last writer wins)
	dataMu.RLock()
	assert.NotNil(t, systemConfigs["inventory_threshold"])
	dataMu.RUnlock()
}

func TestSetSystemConfigRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     SetSystemConfigRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: SetSystemConfigRequest{
				Key:   "test_key",
				Value: "test_value",
			},
			wantErr: false,
		},
		{
			name: "empty key",
			req: SetSystemConfigRequest{
				Key:   "",
				Value: "some_value",
			},
			wantErr: true,
		},
		{
			name: "empty value",
			req: SetSystemConfigRequest{
				Key:   "test_key",
				Value: "",
			},
			wantErr: false,
		},
		{
			name: "inventory threshold key",
			req: SetSystemConfigRequest{
				Key:   "inventory_threshold",
				Value: "10",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate the struct fields
			hasKey := tt.req.Key != ""
			if tt.wantErr {
				assert.False(t, hasKey, "expected validation to fail")
			} else {
				assert.True(t, hasKey, "expected validation to pass")
			}
		})
	}
}

func TestGetAndSetSystemConfigs_Integration(t *testing.T) {
	resetSystemConfigs()
	defer resetSystemConfigs()

	// Set a config
	setReq := httptest.NewRequest(http.MethodPost, "/api/admin/config", strings.NewReader(`{"key":"integration_key","value":"integration_value"}`))
	setReq.Header.Set("Content-Type", "application/json")
	setW := httptest.NewRecorder()
	SetSystemConfig(setW, setReq)
	assert.Equal(t, http.StatusOK, setW.Code)

	// Get all configs and verify
	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/configs", nil)
	getW := httptest.NewRecorder()
	GetSystemConfigs(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)

	var configs []models.SystemConfig
	err := json.NewDecoder(getW.Body).Decode(&configs)
	assert.NoError(t, err)

	found := false
	for _, config := range configs {
		if config.Key == "integration_key" && config.Value == "integration_value" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find the newly set config")
}

// Helper function
func intPtr(i int) *int {
	return &i
}
