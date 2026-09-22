package domain

import (
	"context"
	"time"
)

type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IconURL     string    `json:"icon_url"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	Services    []Service `json:"services,omitempty"`
}

type CategoryRepository interface {
	GetAllActive(ctx context.Context) ([]Category, error)
	GetByID(ctx context.Context, id string) (*Category, error)
	SearchCategories(ctx context.Context, query string) ([]Category, error)
}
