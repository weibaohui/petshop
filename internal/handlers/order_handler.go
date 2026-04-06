package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"petshop/internal/models"
)

// Order management functions

// UpdateOrderStatusRequest represents the request body for updating order status.
type UpdateOrderStatusRequest struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

// RefundRequest represents the request body for processing a refund.
type RefundRequest struct {
	OrderID int64  `json:"orderId"`
	Reason  string `json:"reason"`
}

// ListOrders handles GET /api/admin/orders and returns all orders, optionally filtered by status.
// @Summary List orders
// @Description Get a list of all orders with optional status filter (admin only)
// @Tags admin-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by order status"
// @Success 200 {array} models.Order
// @Router /api/admin/orders [get]
func ListOrders(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	status := r.URL.Query().Get("status")
	orderList := make([]*models.Order, 0, len(orders))

	for _, o := range orders {
		if status == "" || o.Status == status {
			orderList = append(orderList, o)
		}
	}
	json.NewEncoder(w).Encode(orderList)
}

// GetOrder handles GET /api/admin/order?id=<id> and returns the order.
// @Summary Get order
// @Description Get an order by ID (admin only)
// @Tags admin-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query int true "Order ID"
// @Success 200 {object} models.Order
// @Failure 400 {string} string "id is required"
// @Failure 404 {string} string "order not found"
// @Router /api/admin/order [get]
func GetOrder(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dataMu.RLock()
	defer dataMu.RUnlock()

	if o, ok := orders[id]; ok {
		json.NewEncoder(w).Encode(o)
		return
	}
	http.Error(w, "order not found", http.StatusNotFound)
}

// UpdateOrderStatus handles PUT /api/admin/order and updates the order status.
// @Summary Update order status
// @Description Update the status of an order (admin only)
// @Tags admin-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateOrderStatusRequest true "Order status update"
// @Success 200 {object} models.Order
// @Failure 400 {string} string "invalid request body"
// @Failure 404 {string} string "order not found"
// @Router /api/admin/order [put]
func UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req UpdateOrderStatusRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	o, ok := orders[req.ID]
	if !ok {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	validStatuses := map[string]bool{
		"pending": true, "paid": true, "shipped": true, "delivered": true,
		"cancelled": true, "refunding": true, "refunded": true,
	}
	if !validStatuses[req.Status] {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	o.Status = req.Status
	o.UpdatedAt = time.Now()

	json.NewEncoder(w).Encode(o)
}

// ProcessRefund handles POST /api/admin/order/refund and processes a refund for an order.
// @Summary Process refund
// @Description Process a refund for an order (admin only)
// @Tags admin-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RefundRequest true "Refund request"
// @Success 200 {object} map[string]string
// @Failure 400 {string} string "invalid request body"
// @Failure 404 {string} string "order not found"
// @Router /api/admin/order/refund [post]
func ProcessRefund(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req RefundRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	o, ok := orders[req.OrderID]
	if !ok {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	o.Status = "refunded"
	o.RefundReason = req.Reason
	o.UpdatedAt = time.Now()

	// 恢复库存
	for _, item := range o.Products {
		if p, ok := products[item.ProductID]; ok {
			beforeStock := p.Stock
			p.Stock += item.Quantity
			inventoryLogs = append(inventoryLogs, models.Inventory{
				ID:          nextInventoryID,
				ProductID:   p.ID,
				ChangeType:  "in",
				Quantity:    item.Quantity,
				BeforeStock: beforeStock,
				AfterStock:  p.Stock,
				Reason:      fmt.Sprintf("退款返还: 订单%d", req.OrderID),
				Operator:    "system",
				CreatedAt:   time.Now(),
			})
			nextInventoryID++
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "refund processed"})
}
