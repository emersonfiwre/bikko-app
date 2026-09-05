package domain

import (
	"context"
	"time"
)

type User struct {
	ID                   string    `json:"id"`
	FullName             string    `json:"full_name"`
	Email                string    `json:"email"`
	Phone                string    `json:"phone,omitempty"`
	CPF                  string    `json:"cpf,omitempty"`
	PasswordHash         string    `json:"-"`
	ProfilePhotoURL      string    `json:"profile_photo_url,omitempty"`
	Rating               float64   `json:"rating"`
	TotalRatings         int       `json:"total_ratings"`
	IsBikker             bool      `json:"is_bikker"`
	NotificationsEnabled bool      `json:"notifications_enabled"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type Bikker struct {
	ID              string    `json:"id"` // References User.ID (1:1)
	Profession      string    `json:"profession"`
	ExperienceYears string    `json:"experience_years"`
	Description     string    `json:"description"`
	LocationName    string    `json:"location_name"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	Rating          float64   `json:"rating"`
	TotalReviews    int       `json:"total_reviews"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *User, password string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
}

type BikkerRepository interface {
	GetByID(ctx context.Context, id string) (*Bikker, error)
	UpdateRatingAndReviews(ctx context.Context, bikkerID string, rating float64, totalReviews int) error
}
