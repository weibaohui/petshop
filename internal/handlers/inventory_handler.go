// Package handlers provides HTTP handlers for the petshop API.
//
// @Description 库存管理处理器
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"petshop/internal/models"
)

// Inventory management functions

// InventoryAdjustRequest represents the request body for adjusting inventory.
type InventoryAdjustRequest struct {
	ProductID int64  `json:"productId"`
	Quantity  int    `json:"quantity"`
	Reason    string `json:"reason"`
}

// ListInventoryLogs 获取库存变更日志
// @Summary 获取库存变更日志
// @Description 获取所有库存变更记录
// @Tags 库存管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Inventory "库存变更日志列表"
// @Failure 401 {string} string "未授权"
// @Router /api/admin/inventory/logs [get]
func ListInventoryLogs(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	json.NewEncoder(w).Encode(inventoryLogs)
}

// GetInventoryAlerts 获取库存预警
// @Summary 获取库存预警
// @Description 获取库存低于预警阈值的商品列表
// @Tags 库存管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.InventoryAlert "库存预警列表"
// @Failure 401 {string} string "未授权"
// @Router /api/admin/inventory/alerts [get]
func GetInventoryAlerts(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	alerts := make([]models.InventoryAlert, 0)
	for _, p := range products {
		if p.Status != "deleted" && p.Stock <= inventoryThreshold {
			alerts = append(alerts, models.InventoryAlert{
				ProductID:    p.ID,
				ProductName:  p.Name,
				Threshold:    inventoryThreshold,
				CurrentStock: p.Stock,
			})
		}
	}
	json.NewEncoder(w).Encode(alerts)
}

// AdjustInventory 调整库存
// @Summary 调整商品库存
// @Description 调整指定商品的库存数量（增加或减少）
// @Tags 库存管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body InventoryAdjustRequest true "库存调整请求"
// @Success 200 {object} models.Product "更新后的商品信息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "商品不存在"
// @Router /api/admin/inventory/adjust [post]
func AdjustInventory(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req InventoryAdjustRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	p, ok := products[req.ProductID]
	if !ok {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	beforeStock := p.Stock
	p.Stock += req.Quantity
	if p.Stock < 0 {
		p.Stock = 0
	}
	p.UpdatedAt = time.Now()

	changeType := "adjust"
	if req.Quantity > 0 {
		changeType = "in"
	} else if req.Quantity < 0 {
		changeType = "out"
	}

	inventoryLogs = append(inventoryLogs, models.Inventory{
		ID:          nextInventoryID,
		ProductID:   p.ID,
		ChangeType:  changeType,
		Quantity:    abs(req.Quantity),
		BeforeStock: beforeStock,
		AfterStock:  p.Stock,
		Reason:      req.Reason,
		Operator:    "admin",
		CreatedAt:   time.Now(),
	})
	nextInventoryID++

	json.NewEncoder(w).Encode(p)
}
