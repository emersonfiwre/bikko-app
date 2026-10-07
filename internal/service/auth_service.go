package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	firebaseAuth "firebase.google.com/go/v4/auth"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/security"
	"bikko-app/internal/model"
)

var (
	ErrUnconfirmedEmail     = errors.New("unconfirmed_email")
	ErrInvalidCredentials   = errors.New("invalid_credentials")
	ErrUserNotFound         = errors.New("user_not_found")
	ErrInvalidFirebaseToken = errors.New("invalid_firebase_token")
)

type AuthService interface {
	Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error)
	Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error)
	VerifyIDToken(ctx context.Context, idToken string) (*firebaseAuth.Token, error)
}

type authService struct {
	repo         domain.UserRepository
	jwtService   security.JWTService
	firebaseAuth *firebaseAuth.Client
}

func NewAuthService(repo domain.UserRepository, jwtService security.JWTService, fbAuth *firebaseAuth.Client) AuthService {
	return &authService{
		repo:         repo,
		jwtService:   jwtService,
		firebaseAuth: fbAuth,
	}
}

func (s *authService) VerifyIDToken(ctx context.Context, idToken string) (*firebaseAuth.Token, error) {
	if s.firebaseAuth != nil {
		return s.firebaseAuth.VerifyIDToken(ctx, idToken)
	}

	// Fallback for development / mock test when Firebase credentials are not provided
	if strings.HasPrefix(idToken, "mock_") || idToken == "test_token" {
		return &firebaseAuth.Token{
			UID: "mock_firebase_uid",
			Claims: map[string]interface{}{
				
			},
		}, nil
	}

	return nil, ErrInvalidFirebaseToken
}

func (s *authService) Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error) {
	name := req.FullName
	if name == "" {
		name = req.Name
	}
	if name == "" {
		return nil, errors.New("o nome é obrigatório")
	}

	phone := req.Phone

	// If Firebase IDToken is provided, verify phone with Firebase Admin SDK
	if req.IDToken != "" {
		token, err := s.VerifyIDToken(ctx, req.IDToken)
		if err != nil {
			return nil, fmt.Errorf("falha ao verificar autenticação de telefone: %w", err)
		}
		if tokenPhone, ok := token.Claims["phone_number"].(string); ok && tokenPhone != "" {
			phone = tokenPhone
		}
	}

	user := &domain.User{
		FullName: name,
		Email:    req.Email,
		Phone:    phone,
		CPF:      req.CPF,
	}

	createdUser, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	token, err := s.jwtService.GenerateToken(createdUser.ID)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		Message: "Usuário registrado com sucesso",
		Token:   token,
		User: model.User{
			ID:    createdUser.ID,
			Name:  createdUser.FullName,
			Email: createdUser.Email,
			Phone: createdUser.Phone,
		},
	}, nil
}

func (s *authService) Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error) {
	// If Firebase IDToken is provided, perform Phone Auth verification
	if req.IDToken != "" {
		token, err := s.VerifyIDToken(ctx, req.IDToken)
		if err != nil {
			return nil, ErrInvalidCredentials
		}

		phone := req.Phone
		if tokenPhone, ok := token.Claims["phone_number"].(string); ok && tokenPhone != "" {
			phone = tokenPhone
		}

		if phone == "" {
			return nil, ErrInvalidCredentials
		}

		user, err := s.repo.GetByPhone(ctx, phone)
		if err != nil {
			return nil, ErrUserNotFound
		}

		tokenStr, err := s.jwtService.GenerateToken(user.ID)
		if err != nil {
			return nil, err
		}

		return &model.AuthResponse{
			Message: "Login realizado com sucesso",
			Token:   tokenStr,
			User: model.User{
				ID:    user.ID,
				Name:  user.FullName,
				Email: user.Email,
				Phone: user.Phone,
			},
		}, nil
	}

	return nil, ErrInvalidCredentials
}
