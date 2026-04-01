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

// ListInventoryLogs handles GET /api/admin/inventory/logs and returns all inventory change logs.
func ListInventoryLogs(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	json.NewEncoder(w).Encode(inventoryLogs)
}

// GetInventoryAlerts handles GET /api/admin/inventory/alerts and returns products with low stock.
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

// AdjustInventory handles POST /api/admin/inventory/adjust and adjusts product stock quantity.
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
