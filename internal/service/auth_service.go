package service

import (
	"context"

	"bikko-app/internal/model"
	"bikko-app/internal/repository"
)

type AuthService interface {
	Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error)
	Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error)
}

type authService struct {
	repo repository.AuthRepository
}

func NewAuthService(repo repository.AuthRepository) AuthService {
	return &authService{repo: repo}
}

func (s *authService) Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error) {
	user := &model.User{
		Name:  req.Name,
		Email: req.Email,
		Phone: req.Phone,
	}

	createdUser, err := s.repo.CreateUser(ctx, user, req.Password)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		Message: "Usuário registrado com sucesso",
		Token:   "mocked_jwt_token_123456",
		User:    *createdUser,
	}, nil
}

func (s *authService) Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		Message: "Login realizado com sucesso",
		Token:   "mocked_jwt_token_123456",
		User:    *user,
	}, nil
}
