package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	ordersRepo  repository.ProfileRepository
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

func NewProfileService(userRepo domain.UserRepository, ordersRepo repository.ProfileRepository, opts ...ProfileServiceOption) ProfileService {
	ps := &profileService{
		userRepo:   userRepo,
		ordersRepo: ordersRepo,
	}
	for _, opt := range opts {
		opt(ps)
	}
	return ps
}

func NewProfileServiceWithAuth(userRepo domain.UserRepository, ordersRepo repository.ProfileRepository, authService AuthService, jwtService security.JWTService) ProfileService {
	return &profileService{
		userRepo:    userRepo,
		ordersRepo:  ordersRepo,
		authService: authService,
		jwtService:  jwtService,
	}
}

func (s *profileService) GetProfile(ctx context.Context, userID string) (*model.ProfileResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("user id required")
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("profile: load user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	orders := []model.ProfileOrderItem{}
	if s.ordersRepo != nil {
		data, err := s.ordersRepo.GetProfileByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("profile: load orders: %w", err)
		}
		if data != nil && data.Orders != nil {
			orders = data.Orders
		}
	}
	result := &model.ProfileResponse{ID: user.ID, ImageProfile: user.ProfilePhotoURL, Name: user.FullName, Email: user.Email, Phone: user.Phone, Rating: fmt.Sprintf("%.1f", user.Rating), TotalRatings: user.TotalRatings, Orders: orders}
	result.LocationName = user.LocationName
	if user.Latitude != nil {
		result.Latitude = *user.Latitude
	}
	if user.Longitude != nil {
		result.Longitude = *user.Longitude
	}
	if user.Bikker != nil {
		b := user.Bikker
		result.Profession, result.ExperienceYears, result.Description = b.Profession, b.ExperienceYears, b.Description
		result.LocationName, result.Latitude, result.Longitude = b.LocationName, b.Latitude, b.Longitude
		active := b.IsActive
		result.IsActive = &active
	}
	return result, nil
}

func (s *profileService) DeleteAccount(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user id required")
	}
	return s.userRepo.DeleteUser(ctx, userID)
}

func normalizePhoneDigits(raw string) string {
	phone, _ := domain.NormalizePhone(raw)
	return phone
}

func (s *profileService) UpdateProfileWithToken(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user id required")
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("profile: load identity: %w", err)
	}
	if user == nil {
		return "", ErrUserNotFound
	}
	canonicalPhone := user.Phone
	if strings.TrimSpace(phone) != "" {
		canonicalPhone, err = domain.NormalizePhone(phone)
		if err != nil {
			return "", ErrInvalidCredentials
		}
	}
	changed := normalizePhoneDigits(canonicalPhone) != normalizePhoneDigits(user.Phone)
	if changed {
		if firebaseToken == "" || s.authService == nil {
			return "", ErrPhoneChangeRequiresFirebaseToken
		}
		token, err := s.authService.VerifyIDToken(ctx, firebaseToken)
		if err != nil || token == nil {
			return "", ErrInvalidFirebaseToken
		}
		claimedPhone, ok := token.Claims["phone_number"].(string)
		if !ok {
			return "", ErrInvalidFirebaseToken
		}
		verifiedPhone, err := domain.NormalizePhone(claimedPhone)
		if err != nil || verifiedPhone != canonicalPhone {
			return "", ErrInvalidFirebaseToken
		}
		existing, err := s.userRepo.GetByPhone(ctx, canonicalPhone)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return "", fmt.Errorf("profile: check phone: %w", err)
		}
		if existing != nil && existing.ID != userID {
			return "", ErrPhoneAlreadyInUse
		}
	}
	if err := s.userRepo.UpdateUser(ctx, userID, name, email, canonicalPhone); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return "", ErrPhoneAlreadyInUse
		}
		return "", fmt.Errorf("profile: update identity: %w", err)
	}
	if changed && s.jwtService != nil {
		return s.jwtService.GenerateToken(userID)
	}
	return "", nil
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
