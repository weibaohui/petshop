package models

type Pet struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Photos   []string `json:"photos"`
	Status   string   `json:"status"`
}