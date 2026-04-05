package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbs(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "正数",
			input:    10,
			expected: 10,
		},
		{
			name:     "负数",
			input:    -10,
			expected: 10,
		},
		{
			name:     "零",
			input:    0,
			expected: 0,
		},
		{
			name:     "大负数",
			input:    -999999,
			expected: 999999,
		},
		{
			name:     "大正数",
			input:    999999,
			expected: 999999,
		},
		{
			name:     "MinInt溢出",
			input:    math.MinInt,
			expected: math.MaxInt, // math.Abs(float64(MinInt)) 因精度丢失返回 MaxInt
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := abs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		data         interface{}
		expectedBody string
	}{
		{
			name:         "正常数据结构",
			status:       http.StatusOK,
			data:         map[string]string{"message": "success"},
			expectedBody: `{"message":"success"}` + "\n",
		},
		{
			name:         "空map",
			status:       http.StatusOK,
			data:         map[string]string{},
			expectedBody: "{}" + "\n",
		},
		{
			name:         "数组数据",
			status:       http.StatusOK,
			data:         []int{1, 2, 3},
			expectedBody: "[1,2,3]" + "\n",
		},
		{
			name:         "不同状态码",
			status:       http.StatusCreated,
			data:         map[string]string{"id": "123"},
			expectedBody: `{"id":"123"}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeJSON(rr, tt.status, tt.data)

			assert.Equal(t, tt.status, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		message      string
		expectedBody map[string]string
	}{
		{
			name:         "400错误",
			status:       http.StatusBadRequest,
			message:      "invalid request",
			expectedBody: map[string]string{"error": "invalid request"},
		},
		{
			name:         "500错误",
			status:       http.StatusInternalServerError,
			message:      "internal server error",
			expectedBody: map[string]string{"error": "internal server error"},
		},
		{
			name:         "404错误",
			status:       http.StatusNotFound,
			message:      "resource not found",
			expectedBody: map[string]string{"error": "resource not found"},
		},
		{
			name:         "空错误消息",
			status:       http.StatusBadRequest,
			message:      "",
			expectedBody: map[string]string{"error": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeError(rr, tt.status, tt.message)

			assert.Equal(t, tt.status, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			var response map[string]string
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, response)
		})
	}
}
