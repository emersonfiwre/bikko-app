package domain

import (
	"context"
	"time"
)

type Review struct {
	ID           string    `json:"id"`
	OrderID      string    `json:"order_id"`
	ServiceID    string    `json:"service_id"`
	ReviewerID   string    `json:"reviewer_id"`
	BikkerID     string    `json:"bikker_id"`
	Rating       int       `json:"rating"` // 1 to 5
	Comment      string    `json:"comment"`
	LikeCount    int       `json:"like_count"`
	DislikeCount int       `json:"dislike_count"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateReviewInput struct {
	OrderID    string `json:"order_id" binding:"required"`
	ReviewerID string `json:"reviewer_id" binding:"required"`
	Rating     int    `json:"rating" binding:"required,min=1,max=5"`
	Comment    string `json:"comment"`
}

type ReviewRepository interface {
	CreateReviewInTx(ctx context.Context, review *Review) error
}
