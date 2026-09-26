package model

type Business struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Address     string  `json:"address"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	PostalCode  string  `json:"postal_code"`
	Lat         float64 `json:"latitude"`
	Lng         float64 `json:"longitude"`
	AvgRating   float64 `json:"avg_rating"`
	ReviewCount int     `json:"review_count"`
	IsOpen      bool    `json:"is_open"`
}
