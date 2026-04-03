package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"time"

	"petshop/internal/models"
)

// Sales statistics functions

// GetSalesStats handles GET /api/admin/stats/sales and returns sales statistics.
// Query param 'period' can be: day (last 7 days), week (last 4 weeks), month (last 6 months).
func GetSalesStats(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	dataMu.RLock()
	defer dataMu.RUnlock()

	stats := make([]models.SalesStat, 0)

	switch period {
	case "day":
		// 最近7天日报
		for i := 6; i >= 0; i-- {
			date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			stat := calculateDayStat(date)
			stats = append(stats, stat)
		}
	case "week":
		// 最近4周周报
		for i := 3; i >= 0; i-- {
			weekStart := getWeekStart(time.Now().AddDate(0, 0, -i*7))
			weekEnd := weekStart.AddDate(0, 0, 6)
			stat := calculatePeriodStat(weekStart, weekEnd)
			stat.Date = weekStart.Format("2006-01-02")
			stats = append(stats, stat)
		}
	case "month":
		// 最近6月月报
		for i := 5; i >= 0; i-- {
			month := time.Now().AddDate(0, -i, 0)
			monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.Local)
			monthEnd := monthStart.AddDate(0, 1, -1)
			stat := calculatePeriodStat(monthStart, monthEnd)
			stat.Date = monthStart.Format("2006-01")
			stats = append(stats, stat)
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid period"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// calculateDayStat calculates sales statistics for a given date.
func calculateDayStat(date string) models.SalesStat {
	stat := models.SalesStat{Date: date}

	for _, o := range orders {
		if o.Status != "cancelled" && o.Status != "refunded" {
			orderDate := o.CreatedAt.Format("2006-01-02")
			if orderDate == date {
				stat.TotalSales += o.TotalAmount
				stat.OrderCount++
			}
		}
	}
	return stat
}

// calculatePeriodStat calculates sales statistics for a given date range.
func calculatePeriodStat(start, end time.Time) models.SalesStat {
	stat := models.SalesStat{Date: start.Format("2006-01-02")}

	for _, o := range orders {
		if o.Status != "cancelled" && o.Status != "refunded" {
			if o.CreatedAt.After(start) && o.CreatedAt.Before(end.AddDate(0, 0, 1)) {
				stat.TotalSales += o.TotalAmount
				stat.OrderCount++
			}
		}
	}
	return stat
}

// getWeekStart returns the Monday of the week for the given time.
func getWeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, time.Local)
}

// GetHotProducts handles GET /api/admin/stats/hot-products and returns top selling products.
// Query param 'limit' sets the maximum number of products to return (default 10).
func GetHotProducts(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	dataMu.RLock()
	defer dataMu.RUnlock()

	productSales := make(map[int64]struct {
		name   string
		count  int
		amount float64
	})

	for _, o := range orders {
		if o.Status != "cancelled" && o.Status != "refunded" {
			for _, item := range o.Products {
				if ps, ok := productSales[item.ProductID]; ok {
					ps.count += item.Quantity
					ps.amount += item.Subtotal
					productSales[item.ProductID] = ps
				} else {
					productSales[item.ProductID] = struct {
						name   string
						count  int
						amount float64
					}{
						name:   item.ProductName,
						count:  item.Quantity,
						amount: item.Subtotal,
					}
				}
			}
		}
	}

	hotList := make([]models.HotProduct, 0, len(productSales))
	for pid, ps := range productSales {
		hotList = append(hotList, models.HotProduct{
			ProductID:   pid,
			ProductName: ps.name,
			SalesCount:  ps.count,
			SalesAmount: ps.amount,
		})
	}

	// 排序（使用标准库 sort.Slice）
	sort.Slice(hotList, func(i, j int) bool {
		return hotList[i].SalesCount > hotList[j].SalesCount
	})

	if len(hotList) > limit {
		hotList = hotList[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hotList)
}
