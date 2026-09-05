package service

import (
	"context"

	"bikko-app/internal/model"
	"bikko-app/internal/repository"
)

type ProfileService interface {
	GetProfile(ctx context.Context, userID string) (*model.ProfileResponse, error)
}

type profileService struct {
	repo repository.ProfileRepository
}

func NewProfileService(repo repository.ProfileRepository) ProfileService {
	return &profileService{repo: repo}
}

func (s *profileService) GetProfile(ctx context.Context, userID string) (*model.ProfileResponse, error) {
	if userID == "" {
		userID = "user_123"
	}
	return s.repo.GetProfileByID(ctx, userID)
}
