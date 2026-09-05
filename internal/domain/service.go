package domain

import (
	"context"
	"time"
)

type ServicePhoto struct {
	ID           string `json:"id"`
	ServiceID    string `json:"service_id"`
	PhotoURL     string `json:"photo_url"`
	DisplayOrder int    `json:"display_order"`
}

type Service struct {
	ID             string         `json:"id"`
	BikkerID       string         `json:"bikker_id"`
	CategoryID     string         `json:"category_id"`
	Category       *Category      `json:"category,omitempty"` // Nested Category for Data Shaping (solves N+1 in Android)
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	ThumbnailURL   string         `json:"thumbnail_url"`
	Latitude       float64        `json:"latitude"`
	Longitude      float64        `json:"longitude"`
	ReviewsAverage float64        `json:"reviews_average"`
	TotalReviews   int            `json:"total_reviews"`
	IsActive       bool           `json:"is_active"`
	Photos         []ServicePhoto `json:"photos,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type ServiceRepository interface {
	GetByID(ctx context.Context, id string) (*Service, error)
	GetServicesNear(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]Service, error)
	GetFeaturedAdviceServices(ctx context.Context, limit int) ([]Service, error)
	GetActiveServicesByBikkerID(ctx context.Context, bikkerID string) ([]Service, error)
	SearchServices(ctx context.Context, query string) ([]Service, error)
}
