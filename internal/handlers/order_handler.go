package handlers

import (
	"encoding/json"
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
	status := r.URL.Query().Get("status")

	var orderList []*models.Order
	var err error

	if status == "" {
		orderList, err = orderRepo.GetAll()
	} else {
		orderList, err = orderRepo.GetByStatus(status)
	}

	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
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

	o, err := orderRepo.GetByID(id)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(o)
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

	o, err := orderRepo.GetByID(req.ID)
	if err != nil {
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

	if err := orderRepo.UpdateStatus(req.ID, req.Status); err != nil {
		http.Error(w, "failed to update order", http.StatusInternalServerError)
		return
	}

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

	o, err := orderRepo.GetByID(req.OrderID)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	// Restore inventory for each item
	for _, item := range o.Products {
		p, err := productRepo.GetByID(item.ProductID)
		if err == nil {
			beforeStock := p.Stock
			p.Stock += item.Quantity
			productRepo.UpdateStock(p.ID, p.Stock)

			inventoryRepo.Create(&models.Inventory{
				ProductID:   p.ID,
				ChangeType:  "in",
				Quantity:    item.Quantity,
				BeforeStock: beforeStock,
				AfterStock:  p.Stock,
				Reason:      "退款返还",
				Operator:    "system",
				CreatedAt:   time.Now(),
			})
		}
	}

	if err := orderRepo.UpdateRefund(req.OrderID, req.Reason); err != nil {
		http.Error(w, "failed to process refund", http.StatusInternalServerError)
		return
	}

	o.Status = "refunded"
	o.RefundReason = req.Reason
	o.UpdatedAt = time.Now()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "refund processed"})
}
