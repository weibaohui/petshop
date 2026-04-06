// Package handlers provides HTTP handlers for the petshop API.
//
// @Description 轮播图管理处理器
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"petshop/internal/models"
)

// Carousel management functions

// CreateCarouselRequest represents the request body for creating a carousel.
type CreateCarouselRequest struct {
	ImageURL  string `json:"imageUrl"`
	LinkURL   string `json:"linkUrl"`
	SortOrder int    `json:"sortOrder"`
	Title     string `json:"title"`
}

// ListCarousels 获取轮播图列表
// @Summary 获取轮播图列表
// @Description 获取所有轮播图
// @Tags 轮播图管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Carousel "轮播图列表"
// @Failure 401 {string} string "未授权"
// @Router /api/admin/carousels [get]
func ListCarousels(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	defer dataMu.RUnlock()

	carouselList := make([]*models.Carousel, 0, len(carousels))
	for _, c := range carousels {
		carouselList = append(carouselList, c)
	}
	json.NewEncoder(w).Encode(carouselList)
}

// CreateCarousel 创建轮播图
// @Summary 创建轮播图
// @Description 创建新的轮播图
// @Tags 轮播图管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateCarouselRequest true "轮播图创建请求"
// @Success 201 {object} models.Carousel "创建成功的轮播图"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Router /api/admin/carousels [post]
func CreateCarousel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req CreateCarouselRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ImageURL == "" {
		http.Error(w, "imageUrl is required", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	c := &models.Carousel{
		ID:        nextCarouselID,
		ImageURL:  req.ImageURL,
		LinkURL:   req.LinkURL,
		SortOrder: req.SortOrder,
		Title:     req.Title,
		Status:    "active",
	}
	carousels[nextCarouselID] = c
	nextCarouselID++

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

// UpdateCarousel 更新轮播图
// @Summary 更新轮播图
// @Description 更新现有轮播图信息
// @Tags 轮播图管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param carousel body models.Carousel true "轮播图信息"
// @Success 200 {object} models.Carousel "更新后的轮播图"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "轮播图不存在"
// @Router /api/admin/carousel [put]
func UpdateCarousel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var c models.Carousel
	if err := json.Unmarshal(body, &c); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	dataMu.Lock()
	defer dataMu.Unlock()

	if existing, ok := carousels[c.ID]; ok {
		if c.ImageURL != "" {
			existing.ImageURL = c.ImageURL
		}
		if c.LinkURL != "" {
			existing.LinkURL = c.LinkURL
		}
		existing.SortOrder = c.SortOrder
		if c.Title != "" {
			existing.Title = c.Title
		}
		if c.Status != "" {
			existing.Status = c.Status
		}
		json.NewEncoder(w).Encode(existing)
		return
	}
	http.Error(w, "carousel not found", http.StatusNotFound)
}

// DeleteCarousel 删除轮播图
// @Summary 删除轮播图
// @Description 根据ID删除轮播图
// @Tags 轮播图管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "轮播图ID"
// @Success 200 {object} map[string]string "删除成功消息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "轮播图不存在"
// @Router /api/admin/carousel [delete]
func DeleteCarousel(w http.ResponseWriter, r *http.Request) {
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

	dataMu.Lock()
	defer dataMu.Unlock()

	if _, ok := carousels[id]; ok {
		delete(carousels, id)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
		return
	}
	http.Error(w, "carousel not found", http.StatusNotFound)
}
