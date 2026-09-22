package domain

import (
	"context"
	"time"
)

type ServiceReviewProfile struct {
	ImageProfile string `json:"image_profile"`
	Name         string `json:"name"`
}

type ServiceReviewItem struct {
	ReviewProfile ServiceReviewProfile `json:"review_profile"`
	Rating        int                  `json:"rating"`
	Comment       string               `json:"comment"`
	Date          string               `json:"date"`
	Like          int                  `json:"like"`
	Dislike       int                  `json:"dislike"`
	ProviderTitle string               `json:"provider_title,omitempty"`
	ServiceTag    string               `json:"service_tag,omitempty"`
}

type ServicePhoto struct {
	ID           string `json:"id"`
	ServiceID    string `json:"service_id"`
	PhotoURL     string `json:"photo_url"`
	DisplayOrder int    `json:"display_order"`
}

type Service struct {
	ID             string              `json:"id"`
	BikkerID       string              `json:"bikker_id"`
	CategoryID     string              `json:"category_id"`
	Category       *Category           `json:"category,omitempty"` // Nested Category for Data Shaping (solves N+1 in Android)
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	ThumbnailURL   string              `json:"thumbnail_url"`
	Latitude       float64             `json:"latitude"`
	Longitude      float64             `json:"longitude"`
	ReviewsAverage float64             `json:"reviews_average"`
	TotalReviews   int                 `json:"total_reviews"`
	IsActive       bool                `json:"is_active"`
	Photos           []string            `json:"photos,omitempty"`
	Reviews          []ServiceReviewItem `json:"review,omitempty"`
	ProviderName     string              `json:"provider_name,omitempty"`
	ProviderTitle    string              `json:"provider_title,omitempty"`
	ProviderPhotoURL string              `json:"provider_photo_url,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

type ServiceRepository interface {
	GetByID(ctx context.Context, id string) (*Service, error)
	GetServiceByID(ctx context.Context, id string) (*Service, error)
	GetServicesNear(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]Service, error)
	GetFeaturedAdviceServices(ctx context.Context, limit int) ([]Service, error)
	GetActiveServicesByBikkerID(ctx context.Context, bikkerID string) ([]Service, error)
	GetServicesByCategoryID(ctx context.Context, categoryID string) ([]Service, error)
	SearchServices(ctx context.Context, query string) ([]Service, error)
}
