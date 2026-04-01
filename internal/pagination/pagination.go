package pagination

import (
	"net/http"
	"strconv"
)

// Page represents pagination parameters
type Page struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// DefaultPageSize is the default number of items per page
const DefaultPageSize = 10

// MaxPageSize is the maximum allowed items per page
const MaxPageSize = 100

// ParsePagination parses pagination parameters from request
func ParsePagination(r *http.Request) *Page {
	page := &Page{
		Page:     1,
		PageSize: DefaultPageSize,
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page.Page = p
		}
	}

	if pageSizeStr := r.URL.Query().Get("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			if ps > MaxPageSize {
				ps = MaxPageSize
			}
			page.PageSize = ps
		}
	}

	return page
}

// Paginate calculates pagination metadata for a list
func Paginate(items []interface{}, page, pageSize int) (*Page, []interface{}) {
	total := len(items)
	totalPages := (total + pageSize - 1) / pageSize

	if page < 1 {
		page = 1
	}
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return &Page{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		}, []interface{}{}
	}

	if end > total {
		end = total
	}

	return &Page{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}, items[start:end]
}

// PagedResponse is a generic paginated response
type PagedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Page        `json:"pagination"`
}

// NewPagedResponse creates a new paginated response
func NewPagedResponse(data interface{}, page *Page) PagedResponse {
	return PagedResponse{
		Data:       data,
		Pagination: *page,
	}
}
