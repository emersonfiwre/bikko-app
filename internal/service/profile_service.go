package service

import (
	"context"
	"fmt"

	"bikko-app/internal/domain"
	"bikko-app/internal/model"
	"bikko-app/internal/repository"
)

type ProfileService interface {
	GetProfile(ctx context.Context, userID string) (*model.ProfileResponse, error)
	DeleteAccount(ctx context.Context, userID string) error
}

type profileService struct {
	userRepo domain.UserRepository
	mockRepo repository.ProfileRepository
}

func NewProfileService(userRepo domain.UserRepository, mockRepo repository.ProfileRepository) ProfileService {
	return &profileService{
		userRepo: userRepo,
		mockRepo: mockRepo, // Kept to fetch mock orders temporarily
	}
}

func (s *profileService) GetProfile(ctx context.Context, userID string) (*model.ProfileResponse, error) {
	if userID == "" {
		userID = "user_123"
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		// Fallback for mock backward compatibility if the user doesn't exist in DB
		return s.mockRepo.GetProfileByID(ctx, userID)
	}

	mockData, _ := s.mockRepo.GetProfileByID(ctx, userID)
	orders := []model.ProfileOrderItem{}
	if mockData != nil {
		orders = mockData.Orders
	}

	rating := fmt.Sprintf("%.1f", user.Rating)
	if user.Rating == 0 {
		rating = "N/A"
	}

	return &model.ProfileResponse{
		ID:           user.ID,
		ImageProfile: user.ProfilePhotoURL,
		Name:         user.FullName,
		Email:        user.Email,
		Phone:        user.Phone,
		Rating:       rating,
		TotalRatings: user.TotalRatings,
		Orders:       orders,
	}, nil
}

func (s *profileService) DeleteAccount(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user id required")
	}
	return s.userRepo.DeleteUser(ctx, userID)
}

