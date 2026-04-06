// Package handlers provides HTTP handlers for the petshop API.
//
// @Description 订单管理处理器
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

// ListOrders 获取订单列表
// @Summary 获取订单列表
// @Description 获取所有订单列表，支持按状态筛选
// @Tags 订单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "订单状态筛选 (pending, paid, shipped, delivered, cancelled, refunding, refunded)"
// @Success 200 {array} models.Order "订单列表"
// @Failure 401 {string} string "未授权"
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

// GetOrder 获取订单详情
// @Summary 获取订单详情
// @Description 根据订单ID获取订单详细信息
// @Tags 订单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "订单ID"
// @Success 200 {object} models.Order "订单详情"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "订单不存在"
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

// UpdateOrderStatus 更新订单状态
// @Summary 更新订单状态
// @Description 更新订单的状态
// @Tags 订单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateOrderStatusRequest true "状态更新请求"
// @Success 200 {object} models.Order "更新后的订单"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "订单不存在"
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

// ProcessRefund 处理订单退款
// @Summary 处理订单退款
// @Description 为订单处理退款并恢复库存
// @Tags 订单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RefundRequest true "退款请求"
// @Success 200 {object} map[string]string "退款成功消息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "订单不存在"
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
