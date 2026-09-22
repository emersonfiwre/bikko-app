package domain

import (
	"context"
)

type NearBikkerItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	ImageURL string `json:"image_url"`
	Distance string `json:"distance"`
}

type BikkerProfileResponse struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Profession      string              `json:"profession"`
	ProfilePhotoURL string              `json:"profile_photo_url"`
	Rating          string              `json:"rating"`
	TotalReviews    int                 `json:"total_reviews"`
	ExperienceYears string              `json:"experience_years"`
	Location        string              `json:"location"`
	Description     string              `json:"description"`
	OfferedServices []Service           `json:"offered_services"`
	Reviews         []ServiceReviewItem `json:"reviews"`
}

type BikkerRepository interface {
	GetByID(ctx context.Context, id string) (*BikkerProfileResponse, error)
	GetNearBikkers(ctx context.Context, lat, lon float64, limit int) ([]NearBikkerItem, error)
	GetAll(ctx context.Context, limit int) ([]BikkerProfileResponse, error)
}
