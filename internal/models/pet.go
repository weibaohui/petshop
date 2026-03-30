package models

type Pet struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Price float64 `json:"price"`
}