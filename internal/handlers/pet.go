// Package handlers provides HTTP handlers for the petshop API.
//
// @Description PetShop API 处理器
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"strings"
	"time"

	"petshop/internal/cache"
	"petshop/internal/logger"
	"petshop/internal/middleware"
	"petshop/internal/models"
	"petshop/internal/pagination"
	"petshop/internal/validator"
)

var (
	// petsMu protects concurrent access to the pets slice
	petsMu sync.RWMutex
	// pets is the in-memory store of pets
	pets = []models.Pet{
		{
			ID:         1,
			Name:       "旺财",
			Type:       "狗狗",
			Breed:      "金毛寻回犬",
			PhotoUrls:  []string{"https://images.unsplash.com/photo-1552053831-71594a27632d?w=400", "https://images.unsplash.com/photo-1633722715463-d30f4f325e24?w=400"},
			Status:     "available",
			Age:        6,
			AgeDisplay: "6个月",
			Price:      3500,
			Description: "活泼可爱的金毛宝宝，性格温顺，适合家庭饲养。已完成基础训练，会坐下、握手等基本指令。",
			HealthStatus: "健康状况良好，已完成体内外驱虫",
			VaccinationRecords: []models.VaccinationRecord{
				{Name: "狂犬疫苗", Date: "2024-01-15", Completed: true},
				{Name: "六联疫苗", Date: "2024-02-01", Completed: true},
			},
			CreatedAt: "2024-01-10",
		},
		{
			ID:         2,
			Name:       "咪咪",
			Type:       "猫咪",
			Breed:      "英国短毛猫",
			PhotoUrls:  []string{"https://images.unsplash.com/photo-1574158622682-e40e69881006?w=400", "https://images.unsplash.com/photo-1513245543132-31f507417b26?w=400"},
			Status:     "available",
			Age:        4,
			AgeDisplay: "4个月",
			Price:      2800,
			Description: "蓝白英短，品相极佳，毛色亮丽。性格粘人，喜欢在主人怀里睡觉。",
			HealthStatus: "健康状况优秀，定期体检",
			VaccinationRecords: []models.VaccinationRecord{
				{Name: "猫三联", Date: "2024-02-10", Completed: true},
				{Name: "狂犬疫苗", Date: "2024-02-25", Completed: false},
			},
			CreatedAt: "2024-02-01",
		},
		{
			ID:         3,
			Name:       "小白",
			Type:       "狗狗",
			Breed:      "萨摩耶",
			PhotoUrls:  []string{"https://images.unsplash.com/photo-1529429617124-95b109e86bb8?w=400"},
			Status:     "pending",
			Age:        8,
			AgeDisplay: "8个月",
			Price:      4200,
			Description: "微笑天使萨摩耶，雪白蓬松的毛发，性格活泼亲人。",
			HealthStatus: "健康状况良好",
			VaccinationRecords: []models.VaccinationRecord{
				{Name: "狂犬疫苗", Date: "2024-01-20", Completed: true},
				{Name: "八联疫苗", Date: "2024-02-05", Completed: true},
			},
			CreatedAt: "2024-01-15",
		},
		{
			ID:         4,
			Name:       "豆豆",
			Type:       "猫咪",
			Breed:      "布偶猫",
			PhotoUrls:  []string{"https://images.unsplash.com/photo-1533738363-b7f9aef128ce?w=400"},
			Status:     "available",
			Age:        12,
			AgeDisplay: "1岁",
			Price:      5500,
			Description: "海双布偶猫，蓝宝石般的眼睛，性格温柔如布偶。",
			HealthStatus: "健康状况优秀，已绝育",
			VaccinationRecords: []models.VaccinationRecord{
				{Name: "猫三联", Date: "2024-01-05", Completed: true},
				{Name: "狂犬疫苗", Date: "2024-01-20", Completed: true},
			},
			CreatedAt: "2024-01-05",
		},
		{
			ID:         5,
			Name:       "球球",
			Type:       "鸟类",
			Breed:      "虎皮鹦鹉",
			PhotoUrls:  []string{"https://images.unsplash.com/photo-1552728089-57bdde30beb3?w=400"},
			Status:     "available",
			Age:        3,
			AgeDisplay: "3个月",
			Price:      180,
			Description: "色彩鲜艳的虎皮鹦鹉，聪明好学，可以学说话。",
			HealthStatus: "健康状况良好",
			VaccinationRecords: []models.VaccinationRecord{},
			CreatedAt: "2024-03-01",
		},
		{
			ID:         6,
			Name:       "小黑",
			Type:       "狗狗",
			Breed:      "拉布拉多",
			PhotoUrls:  []string{"https://images.unsplash.com/photo-1591769225440-811ad7d6eca6?w=400"},
			Status:     "sold",
			Age:        10,
			AgeDisplay: "10个月",
			Price:      3800,
			Description: "聪明伶俐的黑色拉布拉多，导盲犬潜质。",
			HealthStatus: "健康状况优秀",
			VaccinationRecords: []models.VaccinationRecord{
				{Name: "狂犬疫苗", Date: "2024-01-10", Completed: true},
				{Name: "八联疫苗", Date: "2024-01-25", Completed: true},
			},
			CreatedAt: "2024-01-08",
		},
		{
			ID:         7,
			Name:       "橘橘",
			Type:       "猫咪",
			Breed:      "橘猫",
			PhotoUrls:  []string{"https://images.unsplash.com/photo-1514888286974-6c03e2ca1dba?w=400"},
			Status:     "available",
			Age:        5,
			AgeDisplay: "5个月",
			Price:      500,
			Description: "胖乎乎的橘猫，爱吃爱玩，性格超好。",
			HealthStatus: "健康状况良好",
			VaccinationRecords: []models.VaccinationRecord{
				{Name: "猫三联", Date: "2024-02-15", Completed: true},
			},
			CreatedAt: "2024-02-10",
		},
		{
			ID:         8,
			Name:       "豆豆",
			Type:       "其他",
			Breed:      "垂耳兔",
			PhotoUrls:  []string{"https://images.unsplash.com/photo-1585110396000-c9ffd4e4b308?w=400"},
			Status:     "available",
			Age:        4,
			AgeDisplay: "4个月",
			Price:      350,
			Description: "可爱的垂耳兔，毛茸茸软绵绵，很亲人。",
			HealthStatus: "健康状况良好",
			VaccinationRecords: []models.VaccinationRecord{},
			CreatedAt: "2024-02-20",
		},
	}
	// petCache provides caching for pet data
	petCache *cache.PetCache
	// csrfProt provides CSRF protection for state-changing operations
	csrfProt *middleware.CSRFProtection
	// petLogger is the logger for pet-related operations
	petLogger = logger.New("handlers")
)

// init initializes the pet cache and CSRF protection.
func init() {
	petCache = cache.NewPetCache(1000, 5*time.Minute)
	csrfProt = middleware.NewCSRFProtection()
}

// GetPetCache returns the global pet cache instance
func GetPetCache() *cache.PetCache {
	return petCache
}

// ResetPetsForTesting resets the pets state for testing
// This should only be used in tests
func ResetPetsForTesting() {
	petsMu.Lock()
	defer petsMu.Unlock()

	// Reset pets to initial state
	pets = []models.Pet{
		{ID: 1, Name: "Buddy", Type: "Dog", PhotoUrls: []string{"url1"}, Status: "available"},
		{ID: 2, Name: "Whiskers", Type: "Cat", PhotoUrls: []string{"url2"}, Status: "available"},
		{ID: 3, Name: "Goldie", Type: "Fish", PhotoUrls: []string{"url3"}, Status: "available"},
	}

	// Reset cache if it exists
	if petCache != nil {
		petCache.ResetForTesting()
	} else {
		petCache = cache.NewPetCache(1000, 5*time.Minute)
	}
}

// ListPets 返回分页的宠物列表
// @Summary 获取宠物列表
// @Description 获取分页的宠物列表，支持按类型筛选
// @Tags 宠物管理
// @Accept json
// @Produce json
// @Param type query string false "宠物类型筛选"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "分页的宠物列表"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /api/pets [get]
func ListPets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	typeParam := r.URL.Query().Get("type")
	page := pagination.ParsePagination(r)

	petsMu.RLock()
	defer petsMu.RUnlock()

	// Filter pets if type specified
	var filtered []models.Pet
	for _, pet := range pets {
		if typeParam == "" || strings.EqualFold(pet.Type, typeParam) {
			filtered = append(filtered, pet)
		}
	}

	// Convert to interface slice for pagination
	items := make([]interface{}, len(filtered))
	for i, pet := range filtered {
		items[i] = pet
	}

	pagedPage, pagedItems := pagination.Paginate(items, page.Page, page.PageSize)

	// Convert back to Pet slice
	result := make([]models.Pet, len(pagedItems))
	for i, item := range pagedItems {
		result[i] = item.(models.Pet)
	}

	petLogger.Info("list pets", map[string]interface{}{
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     pagedPage.Total,
	})

	json.NewEncoder(w).Encode(pagination.NewPagedResponse(result, pagedPage))
}

// GetPet 根据ID获取单个宠物详情
// @Summary 获取宠物详情
// @Description 根据宠物ID获取详细信息，支持缓存
// @Tags 宠物管理
// @Accept json
// @Produce json
// @Param id query string true "宠物ID"
// @Success 200 {object} models.Pet "宠物详情"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "宠物未找到"
// @Router /api/pet [get]
func GetPet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}

	var targetID int64
	if _, err := fmt.Sscanf(idStr, "%d", &targetID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid id format"})
		return
	}

	// Try cache first
	cacheKey := cache.GetPetKey(targetID)
	if cached, found := petCache.Get(cacheKey); found {
		petLogger.Debug("cache hit", map[string]interface{}{"id": targetID})
		json.NewEncoder(w).Encode(cached)
		return
	}

	petsMu.RLock()
	defer petsMu.RUnlock()

	for _, pet := range pets {
		if pet.ID == targetID {
			// Store in cache
			petCache.Set(cacheKey, pet)
			json.NewEncoder(w).Encode(pet)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// DeletePet 删除宠物
// @Summary 删除宠物
// @Description 根据ID删除宠物
// @Tags 宠物管理
// @Accept json
// @Produce json
// @Param id query string true "宠物ID"
// @Success 200 {object} models.Pet "已删除的宠物信息"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "宠物未找到"
// @Router /api/pet [delete]
func DeletePet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}

	var targetID int64
	if _, err := fmt.Sscanf(idStr, "%d", &targetID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid id format"})
		return
	}

	petsMu.Lock()
	defer petsMu.Unlock()

	for i, pet := range pets {
		if pet.ID == targetID {
			deletedPet := pets[i]
			pets = append(pets[:i], pets[i+1:]...)
			// Invalidate cache
			petCache.Delete(cache.GetPetKey(targetID))
			petLogger.Info("pet deleted", map[string]interface{}{"id": targetID})
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(deletedPet)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// SearchPets 根据名称搜索宠物
// @Summary 搜索宠物
// @Description 根据宠物名称搜索，支持分页
// @Tags 宠物管理
// @Accept json
// @Produce json
// @Param name query string false "宠物名称关键词"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "搜索结果"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /api/pet/search [get]
func SearchPets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nameParam := r.URL.Query().Get("name")
	page := pagination.ParsePagination(r)

	petsMu.RLock()
	defer petsMu.RUnlock()

	var filtered []models.Pet
	for _, pet := range pets {
		if nameParam == "" || containsIgnoreCase(pet.Name, nameParam) {
			filtered = append(filtered, pet)
		}
	}

	// Convert to interface slice for pagination
	items := make([]interface{}, len(filtered))
	for i, pet := range filtered {
		items[i] = pet
	}

	pagedPage, pagedItems := pagination.Paginate(items, page.Page, page.PageSize)

	// Convert back to Pet slice
	result := make([]models.Pet, len(pagedItems))
	for i, item := range pagedItems {
		result[i] = item.(models.Pet)
	}

	json.NewEncoder(w).Encode(pagination.NewPagedResponse(result, pagedPage))
}

// containsIgnoreCase checks if s contains substr, case-insensitively.
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// UpdatePet 更新宠物信息
// @Summary 更新宠物
// @Description 更新宠物的详细信息，支持部分字段更新
// @Tags 宠物管理
// @Accept json
// @Produce json
// @Param pet body models.Pet true "宠物信息"
// @Success 200 {object} models.Pet "更新后的宠物信息"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "宠物未找到"
// @Router /api/pet [put]
func UpdatePet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	var pet models.Pet
	if err := json.Unmarshal(body, &pet); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON format"})
		return
	}

	if pet.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}

	// Validate path ID matches body ID
	pathIDStr := r.URL.Query().Get("id")
	if pathIDStr != "" {
		var pathID int64
		if _, err := fmt.Sscanf(pathIDStr, "%d", &pathID); err == nil && pathID != pet.ID {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "path id does not match body id"})
			return
		}
	}

	if pet.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "name is required"})
		return
	}

	// Check for SQL injection in input
	if validator.ContainsSQLKeywords(pet.Name) || validator.ContainsSQLKeywords(pet.Type) {
		petLogger.Warn("potential injection attempt", map[string]interface{}{
			"name": pet.Name,
			"type": pet.Type,
		})
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid characters in input"})
		return
	}

	petsMu.Lock()
	defer petsMu.Unlock()

	for i, p := range pets {
		if p.ID == pet.ID {
			// Validate input only if the pet exists
			if errs := validator.ValidatePet(pet.Name, pet.Type, pet.Status); errs.HasErrors() {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": errs.Error()})
				return
			}

			pets[i].Name = pet.Name
			if pet.Type != "" {
				pets[i].Type = pet.Type
			}
			if pet.PhotoUrls != nil {
				pets[i].PhotoUrls = pet.PhotoUrls
			}
			if pet.Status != "" {
				pets[i].Status = pet.Status
			}
			// Invalidate cache
			petCache.Delete(cache.GetPetKey(pet.ID))
			petLogger.Info("pet updated", map[string]interface{}{"id": pet.ID})
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(pets[i])
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// AddPetPhoto adds a photo to a pet
func AddPetPhoto(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid path"})
		return
	}
	var targetID int64
	if _, err := fmt.Sscanf(pathParts[3], "%d", &targetID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid id format"})
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	// Validate URL
	if req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "url is required"})
		return
	}

	if !validator.ValidateURL(req.URL) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid url format"})
		return
	}

	// Check for XSS
	if validator.ContainsSQLKeywords(req.URL) {
		petLogger.Warn("potential XSS in photo URL", map[string]interface{}{
			"url": req.URL,
		})
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid url"})
		return
	}

	petsMu.Lock()
	defer petsMu.Unlock()

	for i, pet := range pets {
		if pet.ID == targetID {
			for _, existingUrl := range pets[i].PhotoUrls {
				if existingUrl == req.URL {
					// URL already exists, return current list without modification
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(pets[i].PhotoUrls)
					return
				}
			}
			pets[i].PhotoUrls = append(pets[i].PhotoUrls, req.URL)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(pets[i].PhotoUrls)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// DeletePetPhoto deletes a photo from a pet
func DeletePetPhoto(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid path"})
		return
	}
	var targetID int64
	if _, err := fmt.Sscanf(pathParts[3], "%d", &targetID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid id format"})
		return
	}

	urlStr := r.URL.Query().Get("url")
	if urlStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "url parameter is required"})
		return
	}

	petsMu.Lock()
	defer petsMu.Unlock()

	for i, pet := range pets {
		if pet.ID == targetID {
			for j, p := range pets[i].PhotoUrls {
				if p == urlStr {
					pets[i].PhotoUrls = append(pets[i].PhotoUrls[:j], pets[i].PhotoUrls[j+1:]...)
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(pets[i].PhotoUrls)
					return
				}
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(pets[i].PhotoUrls)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// GetPetPhotos returns photos for a pet
func GetPetPhotos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	petsMu.RLock()
	defer petsMu.RUnlock()

	pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid path"})
		return
	}
	var targetID int64
	if _, err := fmt.Sscanf(pathParts[3], "%d", &targetID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid id format"})
		return
	}

	for _, pet := range pets {
		if pet.ID == targetID {
			json.NewEncoder(w).Encode(pet.PhotoUrls)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
}

// PetPhotoHandler routes photo-related requests
func PetPhotoHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetPetPhotos(w, r)
	case http.MethodPost:
		AddPetPhoto(w, r)
	case http.MethodDelete:
		DeletePetPhoto(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// GetCacheStats 获取缓存统计信息
// @Summary 获取缓存统计
// @Description 获取宠物缓存的统计信息，包括条目数、命中率等
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "缓存统计信息"
// @Router /api/pet/cache/stats [get]
func GetCacheStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(petCache.Stats())
}

// GetCacheHitRate 获取缓存命中率
// @Summary 获取缓存命中率
// @Description 获取宠物缓存的命中率
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "缓存命中率"
// @Router /api/pet/cache/hitrate [get]
func GetCacheHitRate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"hit_rate": petCache.HitRate(),
	})
}

// GetCategories 获取所有宠物分类
// @Summary 获取宠物分类
// @Description 获取所有可用的宠物分类列表
// @Tags 宠物管理
// @Accept json
// @Produce json
// @Success 200 {array} models.Category "分类列表"
// @Router /api/v1/categories [get]
func GetCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.Categories())
}

// FilterPets 根据多个条件筛选宠物
// @Summary 筛选宠物
// @Description 根据类型、状态、价格范围、搜索关键词等条件筛选宠物
// @Tags 宠物管理
// @Accept json
// @Produce json
// @Param type query string false "宠物类型"
// @Param status query string false "宠物状态 (available, pending, sold)"
// @Param minPrice query number false "最低价格"
// @Param maxPrice query number false "最高价格"
// @Param search query string false "搜索关键词（名称或品种）"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "筛选结果"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Router /api/v1/pets [get]
func FilterPets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	typeParam := query.Get("type")
	statusParam := query.Get("status")
	searchParam := query.Get("search")
	minPriceStr := query.Get("minPrice")
	maxPriceStr := query.Get("maxPrice")
	page := pagination.ParsePagination(r)

	var minPrice, maxPrice float64
	var hasMinPrice, hasMaxPrice bool
	if minPriceStr != "" {
		var err error
		minPrice, err = strconv.ParseFloat(minPriceStr, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid minPrice"})
			return
		}
		hasMinPrice = true
	}
	if maxPriceStr != "" {
		var err error
		maxPrice, err = strconv.ParseFloat(maxPriceStr, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid maxPrice"})
			return
		}
		hasMaxPrice = true
	}

	petsMu.RLock()
	defer petsMu.RUnlock()

	var filtered []models.Pet
	for _, pet := range pets {
		// Filter by type
		if typeParam != "" && !strings.EqualFold(pet.Type, typeParam) {
			continue
		}
		// Filter by status
		if statusParam != "" && !strings.EqualFold(pet.Status, statusParam) {
			continue
		}
		// Filter by price
		if hasMinPrice && pet.Price < minPrice {
			continue
		}
		if hasMaxPrice && pet.Price > maxPrice {
			continue
		}
		// Search by name or breed
		if searchParam != "" {
			searchLower := strings.ToLower(searchParam)
			if !strings.Contains(strings.ToLower(pet.Name), searchLower) &&
				!strings.Contains(strings.ToLower(pet.Breed), searchLower) {
				continue
			}
		}
		filtered = append(filtered, pet)
	}

	// Convert to interface slice for pagination
	items := make([]interface{}, len(filtered))
	for i, pet := range filtered {
		items[i] = pet
	}

	pagedPage, pagedItems := pagination.Paginate(items, page.Page, page.PageSize)

	// Convert back to Pet slice
	result := make([]models.Pet, len(pagedItems))
	for i, item := range pagedItems {
		result[i] = item.(models.Pet)
	}

	petLogger.Info("filter pets", map[string]interface{}{
		"type":   typeParam,
		"search": searchParam,
		"total":  pagedPage.Total,
	})

	json.NewEncoder(w).Encode(pagination.NewPagedResponse(result, pagedPage))
}
