package model

import "time"

type Review struct {
	ID         string    `json:"id"`
	BusinessID string    `json:"business_id"`
	UserID     string    `json:"user_id"`
	UserName   string    `json:"user_name"`
	Rating     int       `json:"rating"`
	Text       string    `json:"text"`
	Useful     int       `json:"useful"`
	Funny      int       `json:"funny"`
	Cool       int       `json:"cool"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateReviewRequest is the expected POST /reviews body.
type CreateReviewRequest struct {
	BusinessID string `json:"business_id"`
	// UserID     string `json:"user_id"` // stubbed for now, will come from JWT later
	Rating int    `json:"rating"`
	Text   string `json:"text"`
}
