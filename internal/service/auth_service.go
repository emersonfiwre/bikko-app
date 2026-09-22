package service

import (
	"context"
	"errors"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/security"
	"bikko-app/internal/model"
)

var (
	ErrUnconfirmedEmail   = errors.New("unconfirmed_email")
	ErrInvalidCredentials = errors.New("invalid_credentials")
)

type AuthService interface {
	Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error)
	Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error)
}

type authService struct {
	repo       domain.UserRepository
	jwtService security.JWTService
}

func NewAuthService(repo domain.UserRepository, jwtService security.JWTService) AuthService {
	return &authService{
		repo:       repo,
		jwtService: jwtService,
	}
}

func (s *authService) Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error) {
	// Hash password
	hashedPassword, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		FullName: req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
	}

	createdUser, err := s.repo.CreateUser(ctx, user, hashedPassword)
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
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !security.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	token, err := s.jwtService.GenerateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		Message: "Login realizado com sucesso",
		Token:   token,
		User: model.User{
			ID:    user.ID,
			Name:  user.FullName,
			Email: user.Email,
			Phone: user.Phone,
		},
	}, nil
}

