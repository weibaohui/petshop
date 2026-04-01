package models

type Pet struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	PhotoUrls []string `json:"photoUrls"`
	Status   string   `json:"status"`
}