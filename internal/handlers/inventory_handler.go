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
// @Summary List inventory logs
// @Description Get all inventory change logs (admin only)
// @Tags admin-inventory
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Inventory
// @Router /api/admin/inventory/logs [get]
func ListInventoryLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := inventoryRepo.GetAll()
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(logs)
}

// GetInventoryAlerts handles GET /api/admin/inventory/alerts and returns products with low stock.
// @Summary Get inventory alerts
// @Description Get products with low stock alerts (admin only)
// @Tags admin-inventory
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.InventoryAlert
// @Router /api/admin/inventory/alerts [get]
func GetInventoryAlerts(w http.ResponseWriter, r *http.Request) {
	products, err := productRepo.GetLowStock(inventoryThreshold)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	alerts := make([]models.InventoryAlert, 0)
	for _, p := range products {
		alerts = append(alerts, models.InventoryAlert{
			ProductID:    p.ID,
			ProductName:  p.Name,
			Threshold:    inventoryThreshold,
			CurrentStock: p.Stock,
		})
	}
	json.NewEncoder(w).Encode(alerts)
}

// AdjustInventory handles POST /api/admin/inventory/adjust and adjusts product stock quantity.
// @Summary Adjust inventory
// @Description Adjust product stock quantity (admin only)
// @Tags admin-inventory
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body InventoryAdjustRequest true "Inventory adjustment"
// @Success 200 {object} models.Product
// @Failure 400 {string} string "invalid request body"
// @Failure 404 {string} string "product not found"
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

	p, err := productRepo.GetByID(req.ProductID)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	beforeStock := p.Stock
	p.Stock += req.Quantity
	if p.Stock < 0 {
		p.Stock = 0
	}
	p.UpdatedAt = time.Now()

	if err := productRepo.UpdateStock(p.ID, p.Stock); err != nil {
		http.Error(w, "failed to update stock", http.StatusInternalServerError)
		return
	}

	changeType := "adjust"
	if req.Quantity > 0 {
		changeType = "in"
	} else if req.Quantity < 0 {
		changeType = "out"
	}

	inventoryRepo.Create(&models.Inventory{
		ProductID:   p.ID,
		ChangeType:  changeType,
		Quantity:    abs(req.Quantity),
		BeforeStock: beforeStock,
		AfterStock:  p.Stock,
		Reason:      req.Reason,
		Operator:    "admin",
		CreatedAt:   time.Now(),
	})

	json.NewEncoder(w).Encode(p)
}
