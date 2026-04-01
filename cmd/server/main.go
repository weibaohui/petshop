package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"petshop/internal/handlers"
)

func main() {
	fmt.Println("Project: petshop")

	http.HandleFunc("/api/pets", handlers.ListPets)
	http.HandleFunc("/api/pet", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetPet(w, r)
		case http.MethodPut:
			handlers.UpdatePet(w, r)
		case http.MethodDelete:
			handlers.DeletePet(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Allow", "GET, PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/pet/search", handlers.SearchPets)
	http.HandleFunc("/api/pet/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
		// parts should be like ["", "api", "pet", "1"] or ["", "api", "pet", "1", "photos"]
		if len(parts) >= 5 && parts[4] == "photos" {
			handlers.PetPhotoHandler(w, r)
			return
		}
		if len(parts) != 4 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			handlers.GetPet(w, r)
		case http.MethodPut:
			handlers.UpdatePet(w, r)
		case http.MethodDelete:
			handlers.DeletePet(w, r)
		default:
			w.Header().Set("Allow", "GET, PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// ==================== 商品管理 ====================
	http.HandleFunc("/api/admin/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListProducts(w, r)
		case http.MethodPost:
			handlers.CreateProduct(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/admin/product", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetProduct(w, r)
		case http.MethodPut:
			handlers.UpdateProduct(w, r)
		case http.MethodDelete:
			handlers.DeleteProduct(w, r)
		default:
			w.Header().Set("Allow", "GET, PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// ==================== 库存管理 ====================
	http.HandleFunc("/api/admin/inventory/logs", handlers.ListInventoryLogs)
	http.HandleFunc("/api/admin/inventory/alerts", handlers.GetInventoryAlerts)
	http.HandleFunc("/api/admin/inventory/adjust", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.AdjustInventory(w, r)
		} else {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// ==================== 订单管理 ====================
	http.HandleFunc("/api/admin/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListOrders(w, r)
		} else {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/admin/order", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetOrder(w, r)
		case http.MethodPut:
			handlers.UpdateOrderStatus(w, r)
		default:
			w.Header().Set("Allow", "GET, PUT")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/admin/order/refund", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.ProcessRefund(w, r)
		} else {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// ==================== 用户管理 ====================
	http.HandleFunc("/api/admin/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.ListUsers(w, r)
		} else {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/admin/user", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetUser(w, r)
		case http.MethodPut:
			handlers.UpdateUserStatus(w, r)
		default:
			w.Header().Set("Allow", "GET, PUT")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/admin/user/reset-password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.ResetUserPassword(w, r)
		} else {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// ==================== 销售统计 ====================
	http.HandleFunc("/api/admin/stats/sales", handlers.GetSalesStats)
	http.HandleFunc("/api/admin/stats/hot-products", handlers.GetHotProducts)

	// ==================== 系统配置 ====================
	// 轮播图管理
	http.HandleFunc("/api/admin/carousels", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListCarousels(w, r)
		case http.MethodPost:
			handlers.CreateCarousel(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/admin/carousel", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateCarousel(w, r)
		case http.MethodDelete:
			handlers.DeleteCarousel(w, r)
		default:
			w.Header().Set("Allow", "PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 公告管理
	http.HandleFunc("/api/admin/announcements", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListAnnouncements(w, r)
		case http.MethodPost:
			handlers.CreateAnnouncement(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/admin/announcement", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateAnnouncement(w, r)
		case http.MethodDelete:
			handlers.DeleteAnnouncement(w, r)
		default:
			w.Header().Set("Allow", "PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 系统参数配置
	http.HandleFunc("/api/admin/configs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetSystemConfigs(w, r)
		} else {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/admin/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.SetSystemConfig(w, r)
		} else {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func run() error {
	// Application initialization
	return nil
}
