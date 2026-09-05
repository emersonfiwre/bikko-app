package repository

import (
	"context"

	"bikko-app/internal/model"
)

type ProfileRepository interface {
	GetProfileByID(ctx context.Context, id string) (*model.ProfileResponse, error)
}

type mockProfileRepository struct{}

func NewMockProfileRepository() ProfileRepository {
	return &mockProfileRepository{}
}

func (r *mockProfileRepository) GetProfileByID(ctx context.Context, id string) (*model.ProfileResponse, error) {
	return &model.ProfileResponse{
		ID:           id,
		Name:         "Emerson Torres",
		Email:        "emerson@bikko.com.br",
		Phone:        "(11) 99999-9999",
		Rating:       "4.9",
		TotalRatings: 18,
	}, nil
}
