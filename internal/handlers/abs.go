package handlers

import (
	"encoding/json"
	"math"
	"net/http"
)

// abs returns the absolute value of an integer
func abs(x int) int {
	return int(math.Abs(float64(x)))
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
