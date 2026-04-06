// Package handlers provides HTTP handlers for the petshop API.
//
// @Description 系统配置管理处理器
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"petshop/internal/models"
)

// System configuration functions

// SetSystemConfigRequest represents the request body for setting a system config.
type SetSystemConfigRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GetSystemConfigs 获取系统配置列表
// @Summary 获取系统配置列表
// @Description 获取所有系统配置项
// @Tags 系统配置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.SystemConfig "系统配置列表"
// @Failure 401 {string} string "未授权"
// @Router /api/admin/configs [get]
func GetSystemConfigs(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	configs := make([]models.SystemConfig, 0, len(systemConfigs))
	for k, v := range systemConfigs {
		configs = append(configs, models.SystemConfig{Key: k, Value: v})
	}
	json.NewEncoder(w).Encode(configs)
}

// SetSystemConfig 设置系统配置
// @Summary 设置系统配置
// @Description 设置系统配置值。如果key为"inventory_threshold"，会更新库存预警阈值
// @Tags 系统配置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SetSystemConfigRequest true "系统配置设置请求"
// @Success 200 {object} map[string]string "设置成功消息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Router /api/admin/config [post]
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

	dataMu.Lock()
	defer dataMu.Unlock()

	systemConfigs[req.Key] = req.Value

	// 更新库存预警阈值
	if req.Key == "inventory_threshold" {
		v, _ := strconv.Atoi(req.Value)
		inventoryThreshold = v
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "config updated"})
}
