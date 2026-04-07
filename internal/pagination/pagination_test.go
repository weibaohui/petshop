package pagination

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name     string
		query    url.Values
		expected *Page
	}{
		{
			name:     "无参请求时返回默认值",
			query:    url.Values{},
			expected: &Page{Page: 1, PageSize: DefaultPageSize},
		},
		{
			name:     "page为0时回退到1",
			query:    url.Values{"page": []string{"0"}},
			expected: &Page{Page: 1, PageSize: DefaultPageSize},
		},
		{
			name:     "page为负数时回退到1",
			query:    url.Values{"page": []string{"-1"}},
			expected: &Page{Page: 1, PageSize: DefaultPageSize},
		},
		{
			name:     "pageSize超过MaxPageSize时被截断为100",
			query:    url.Values{"pageSize": []string{"200"}},
			expected: &Page{Page: 1, PageSize: MaxPageSize},
		},
		{
			name:     "pageSize为0时回退到DefaultPageSize",
			query:    url.Values{"pageSize": []string{"0"}},
			expected: &Page{Page: 1, PageSize: DefaultPageSize},
		},
		{
			name:     "pageSize为负数时回退到DefaultPageSize",
			query:    url.Values{"pageSize": []string{"-5"}},
			expected: &Page{Page: 1, PageSize: DefaultPageSize},
		},
		{
			name:     "正常的page与pageSize组合",
			query:    url.Values{"page": []string{"3"}, "pageSize": []string{"20"}},
			expected: &Page{Page: 3, PageSize: 20},
		},
		{
			name:     "page为非法字符串时回退到默认值",
			query:    url.Values{"page": []string{"abc"}},
			expected: &Page{Page: 1, PageSize: DefaultPageSize},
		},
		{
			name:     "pageSize为非法字符串时回退到默认值",
			query:    url.Values{"pageSize": []string{"abc"}},
			expected: &Page{Page: 1, PageSize: DefaultPageSize},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL: &url.URL{
					RawQuery: tt.query.Encode(),
				},
			}
			result := ParsePagination(req)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParsePagination() = %+v, expected %+v", result, tt.expected)
			}
		})
	}
}

func TestNewPagedResponse(t *testing.T) {
	tests := []struct {
		name         string
		data         interface{}
		page         *Page
		expectedData interface{}
		expectedPage Page
	}{
		{
			name:         "创建带数据的分页响应",
			data:         []int{1, 2, 3, 4, 5},
			page:         &Page{Page: 1, PageSize: 10, Total: 5, TotalPages: 1},
			expectedData: []int{1, 2, 3, 4, 5},
			expectedPage: Page{Page: 1, PageSize: 10, Total: 5, TotalPages: 1},
		},
		{
			name:         "创建空数据分页响应",
			data:         []int{},
			page:         &Page{Page: 1, PageSize: 10, Total: 0, TotalPages: 0},
			expectedData: []int{},
			expectedPage: Page{Page: 1, PageSize: 10, Total: 0, TotalPages: 0},
		},
		{
			name:         "创建多页分页响应",
			data:         []string{"a", "b", "c"},
			page:         &Page{Page: 2, PageSize: 10, Total: 25, TotalPages: 3},
			expectedData: []string{"a", "b", "c"},
			expectedPage: Page{Page: 2, PageSize: 10, Total: 25, TotalPages: 3},
		},
		{
			name:         "nil数据创建分页响应",
			data:         nil,
			page:         &Page{Page: 1, PageSize: 10, Total: 0, TotalPages: 0},
			expectedData: nil,
			expectedPage: Page{Page: 1, PageSize: 10, Total: 0, TotalPages: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewPagedResponse(tt.data, tt.page)
			if !reflect.DeepEqual(result.Data, tt.expectedData) {
				t.Errorf("Data = %v, expected %v", result.Data, tt.expectedData)
			}
			if !reflect.DeepEqual(result.Pagination, tt.expectedPage) {
				t.Errorf("Pagination = %+v, expected %+v", result.Pagination, tt.expectedPage)
			}
		})
	}
}

func TestGetPageAndLimit(t *testing.T) {
	tests := []struct {
		name         string
		query        url.Values
		expectedPage int
		expectedLimit int
	}{
		{
			name:          "无参请求时返回默认值",
			query:         url.Values{},
			expectedPage:  1,
			expectedLimit: DefaultPageSize,
		},
		{
			name:          "正常page和limit",
			query:         url.Values{"page": []string{"3"}, "limit": []string{"20"}},
			expectedPage:  3,
			expectedLimit: 20,
		},
		{
			name:          "page为0时回退到1",
			query:         url.Values{"page": []string{"0"}},
			expectedPage:  1,
			expectedLimit: DefaultPageSize,
		},
		{
			name:          "page为负数时回退到1",
			query:         url.Values{"page": []string{"-1"}},
			expectedPage:  1,
			expectedLimit: DefaultPageSize,
		},
		{
			name:          "limit超过MaxPageSize时被截断",
			query:         url.Values{"limit": []string{"200"}},
			expectedPage:  1,
			expectedLimit: MaxPageSize,
		},
		{
			name:          "limit为0时回退到DefaultPageSize",
			query:         url.Values{"limit": []string{"0"}},
			expectedPage:  1,
			expectedLimit: DefaultPageSize,
		},
		{
			name:          "limit为负数时回退到DefaultPageSize",
			query:         url.Values{"limit": []string{"-5"}},
			expectedPage:  1,
			expectedLimit: DefaultPageSize,
		},
		{
			name:          "page为非法字符串时回退到默认值",
			query:         url.Values{"page": []string{"abc"}},
			expectedPage:  1,
			expectedLimit: DefaultPageSize,
		},
		{
			name:          "limit为非法字符串时回退到默认值",
			query:         url.Values{"limit": []string{"abc"}},
			expectedPage:  1,
			expectedLimit: DefaultPageSize,
		},
		{
			name:          "pageSize参数被忽略，只使用limit",
			query:         url.Values{"pageSize": []string{"50"}},
			expectedPage:  1,
			expectedLimit: DefaultPageSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL: &url.URL{
					RawQuery: tt.query.Encode(),
				},
			}
			page, limit := GetPageAndLimit(req)
			if page != tt.expectedPage {
				t.Errorf("Page = %d, expected %d", page, tt.expectedPage)
			}
			if limit != tt.expectedLimit {
				t.Errorf("Limit = %d, expected %d", limit, tt.expectedLimit)
			}
		})
	}
}

func TestSetPaginationHeaders(t *testing.T) {
	tests := []struct {
		name             string
		page             int
		limit            int
		total            int
		expectedHeaders  map[string]string
	}{
		{
			name:  "正常分页设置响应头",
			page:  2,
			limit: 10,
			total: 50,
			expectedHeaders: map[string]string{
				"X-Page":        "2",
				"X-Limit":       "10",
				"X-Total":       "50",
				"X-Total-Pages": "5",
			},
		},
		{
			name:  "空数据设置响应头",
			page:  1,
			limit: 10,
			total: 0,
			expectedHeaders: map[string]string{
				"X-Page":        "1",
				"X-Limit":       "10",
				"X-Total":       "0",
				"X-Total-Pages": "0",
			},
		},
		{
			name:  "单页数据设置响应头",
			page:  1,
			limit: 10,
			total: 5,
			expectedHeaders: map[string]string{
				"X-Page":        "1",
				"X-Limit":       "10",
				"X-Total":       "5",
				"X-Total-Pages": "1",
			},
		},
		{
			name:  "不满一页的数据设置响应头",
			page:  1,
			limit: 10,
			total: 25,
			expectedHeaders: map[string]string{
				"X-Page":        "1",
				"X-Limit":       "10",
				"X-Total":       "25",
				"X-Total-Pages": "3",
			},
		},
		{
			name:  "负数page设置响应头",
			page:  -1,
			limit: 10,
			total: 50,
			expectedHeaders: map[string]string{
				"X-Page":        "-1",
				"X-Limit":       "10",
				"X-Total":       "50",
				"X-Total-Pages": "5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := httptest.NewRecorder()
			SetPaginationHeaders(rw, tt.page, tt.limit, tt.total)

			headers := rw.Header()
			for key, expectedValue := range tt.expectedHeaders {
				if actualValue := headers.Get(key); actualValue != expectedValue {
					t.Errorf("Header %s = %s, expected %s", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestPaginate(t *testing.T) {
	makeItems := func(count int) []interface{} {
		items := make([]interface{}, count)
		for i := 0; i < count; i++ {
			items[i] = i + 1
		}
		return items
	}

	tests := []struct {
		name          string
		items         []interface{}
		page          int
		pageSize      int
		expectedPage  *Page
		expectedItems []interface{}
	}{
		{
			name:          "空切片返回空结果与正确Page元数据",
			items:         []interface{}{},
			page:          1,
			pageSize:      10,
			expectedPage:  &Page{Page: 1, PageSize: 10, Total: 0, TotalPages: 0},
			expectedItems: []interface{}{},
		},
		{
			name:          "单页完整返回",
			items:         makeItems(5),
			page:          1,
			pageSize:      10,
			expectedPage:  &Page{Page: 1, PageSize: 10, Total: 5, TotalPages: 1},
			expectedItems: []interface{}{1, 2, 3, 4, 5},
		},
		{
			name:          "多页时正确切片并计算TotalPages-第一页",
			items:         makeItems(25),
			page:          1,
			pageSize:      10,
			expectedPage:  &Page{Page: 1, PageSize: 10, Total: 25, TotalPages: 3},
			expectedItems: []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			name:          "多页时正确切片并计算TotalPages-第二页",
			items:         makeItems(25),
			page:          2,
			pageSize:      10,
			expectedPage:  &Page{Page: 2, PageSize: 10, Total: 25, TotalPages: 3},
			expectedItems: []interface{}{11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		},
		{
			name:          "多页时正确切片并计算TotalPages-最后一页",
			items:         makeItems(25),
			page:          3,
			pageSize:      10,
			expectedPage:  &Page{Page: 3, PageSize: 10, Total: 25, TotalPages: 3},
			expectedItems: []interface{}{21, 22, 23, 24, 25},
		},
		{
			name:          "page超过总页数时回退到最后一页",
			items:         makeItems(25),
			page:          10,
			pageSize:      10,
			expectedPage:  &Page{Page: 3, PageSize: 10, Total: 25, TotalPages: 3},
			expectedItems: []interface{}{21, 22, 23, 24, 25},
		},
		{
			name:          "page为0时回退到第1页",
			items:         makeItems(25),
			page:          0,
			pageSize:      10,
			expectedPage:  &Page{Page: 1, PageSize: 10, Total: 25, TotalPages: 3},
			expectedItems: []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			name:          "page为负数时回退到第1页",
			items:         makeItems(25),
			page:          -1,
			pageSize:      10,
			expectedPage:  &Page{Page: 1, PageSize: 10, Total: 25, TotalPages: 3},
			expectedItems: []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			name:          "start >= total时返回空切片",
			items:         []interface{}{},
			page:          2,
			pageSize:      10,
			expectedPage:  &Page{Page: 2, PageSize: 10, Total: 0, TotalPages: 0},
			expectedItems: []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, items := Paginate(tt.items, tt.page, tt.pageSize)
			if !reflect.DeepEqual(page, tt.expectedPage) {
				t.Errorf("Page = %+v, expected %+v", page, tt.expectedPage)
			}
			if !reflect.DeepEqual(items, tt.expectedItems) {
				t.Errorf("Items = %v, expected %v", items, tt.expectedItems)
			}
		})
	}
}
