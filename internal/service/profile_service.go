package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	firebaseAuth "firebase.google.com/go/v4/auth"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/security"
	"bikko-app/internal/model"
	"bikko-app/internal/repository"
)

var (
	ErrPhoneChangeRequiresFirebaseToken = errors.New("phone_change_requires_firebase_token")
	ErrPhoneAlreadyInUse                = errors.New("phone_already_in_use")
)

type ProfileService interface {
	GetProfile(ctx context.Context, userID string) (*model.ProfileResponse, error)
	DeleteAccount(ctx context.Context, userID string) error
	UpdateProfile(ctx context.Context, userID, name, email, phone string) error
	UpdateProfileWithToken(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error)
	UpdateProfilePhoto(ctx context.Context, userID, photoURL string) error
}

type profileService struct {
	userRepo    domain.UserRepository
	mockRepo    repository.ProfileRepository
	authService AuthService
	jwtService  security.JWTService
}

type ProfileServiceOption func(*profileService)

func WithAuthAndJWT(authService AuthService, jwtService security.JWTService) ProfileServiceOption {
	return func(s *profileService) {
		s.authService = authService
		s.jwtService = jwtService
	}
}

func NewProfileService(userRepo domain.UserRepository, mockRepo repository.ProfileRepository, opts ...ProfileServiceOption) ProfileService {
	ps := &profileService{
		userRepo: userRepo,
		mockRepo: mockRepo, // Kept to fetch mock orders temporarily
	}
	for _, opt := range opts {
		opt(ps)
	}
	return ps
}

func NewProfileServiceWithAuth(userRepo domain.UserRepository, mockRepo repository.ProfileRepository, authService AuthService, jwtService security.JWTService) ProfileService {
	return &profileService{
		userRepo:    userRepo,
		mockRepo:    mockRepo,
		authService: authService,
		jwtService:  jwtService,
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
	if mockData != nil && mockData.Orders != nil {
		orders = mockData.Orders
	}

	rating := fmt.Sprintf("%.1f", user.Rating)
	if user.Rating == 0 || user.TotalRatings == 0 {
		rating = "0.0"
	}

	isActive := true
	profession := "Eletricista / Instalação e Manutenção Elétrica"
	expYears := "5_TO_10"
	description := "Profissional qualificado e verificado pela plataforma Bikko."
	locationName := "Pinheiros, São Paulo - SP"
	lat := -23.5614
	lon := -46.6914
	if user.Bikker != nil {
		if user.Bikker.Profession != "" {
			profession = user.Bikker.Profession
		}
		if user.Bikker.ExperienceYears != "" {
			expYears = user.Bikker.ExperienceYears
		}
		if user.Bikker.Description != "" {
			description = user.Bikker.Description
		}
		if user.Bikker.LocationName != "" {
			locationName = user.Bikker.LocationName
		}
		if user.Bikker.Latitude != 0 {
			lat = user.Bikker.Latitude
		}
		if user.Bikker.Longitude != 0 {
			lon = user.Bikker.Longitude
		}
		isActive = user.Bikker.IsActive
	}

	return &model.ProfileResponse{
		ID:              user.ID,
		ImageProfile:    user.ProfilePhotoURL,
		Name:            user.FullName,
		Email:           user.Email,
		Phone:           user.Phone,
		Rating:          rating,
		TotalRatings:    user.TotalRatings,
		Profession:      profession,
		ExperienceYears: expYears,
		Description:     description,
		LocationName:    locationName,
		Latitude:        lat,
		Longitude:       lon,
		IsActive:        &isActive,
		Orders:          orders,
	}, nil
}

func (s *profileService) DeleteAccount(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user id required")
	}
	return s.userRepo.DeleteUser(ctx, userID)
}

func normalizePhoneDigits(p string) string {
	var b strings.Builder
	for _, r := range p {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if len(digits) > 11 {
		digits = digits[len(digits)-11:]
	}
	return digits
}

func (s *profileService) UpdateProfileWithToken(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user id required")
	}

	var oldDigits string
	currentUser, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if s.authService != nil {
			return "", err
		}
	} else if currentUser != nil {
		oldDigits = normalizePhoneDigits(currentUser.Phone)
	}

	newDigits := normalizePhoneDigits(phone)
	phoneChanged := newDigits != "" && newDigits != oldDigits

	if phoneChanged {
		if firebaseToken == "" && s.authService != nil {
			return "", ErrPhoneChangeRequiresFirebaseToken
		}

		if firebaseToken != "" {
			var fbToken *firebaseAuth.Token
			if s.authService != nil {
				var verifyErr error
				fbToken, verifyErr = s.authService.VerifyIDToken(ctx, firebaseToken)
				if verifyErr != nil {
					return "", ErrInvalidFirebaseToken
				}
			} else {
				isProduction := os.Getenv("ENV") == "production" || os.Getenv("GIN_MODE") == "release"
				if !isProduction && (strings.HasPrefix(firebaseToken, "mock_") || firebaseToken == "test_token") {
					fbToken = &firebaseAuth.Token{
						UID:    "mock_firebase_uid",
						Claims: map[string]interface{}{},
					}
				} else {
					return "", ErrInvalidFirebaseToken
				}
			}

			if fbToken != nil {
				if tokenPhone, ok := fbToken.Claims["phone_number"].(string); ok && tokenPhone != "" {
					phone = tokenPhone
				}
			}
		}

		existingUser, err := s.userRepo.GetByPhone(ctx, phone)
		if err == nil && existingUser != nil && existingUser.ID != userID {
			return "", ErrPhoneAlreadyInUse
		}
	}

	if err := s.userRepo.UpdateUser(ctx, userID, name, email, phone); err != nil {
		return "", err
	}

	var newToken string
	if phoneChanged && s.jwtService != nil {
		var tokenErr error
		newToken, tokenErr = s.jwtService.GenerateToken(userID)
		if tokenErr != nil {
			return "", tokenErr
		}
	}

	return newToken, nil
}

func (s *profileService) UpdateProfile(ctx context.Context, userID, name, email, phone string) error {
	_, err := s.UpdateProfileWithToken(ctx, userID, name, email, phone, "")
	return err
}

func (s *profileService) UpdateProfilePhoto(ctx context.Context, userID, photoURL string) error {
	if userID == "" {
		return fmt.Errorf("user id required")
	}
	if photoURL == "" {
		return fmt.Errorf("photo url required")
	}
	return s.userRepo.UpdateProfilePhoto(ctx, userID, photoURL)
}
