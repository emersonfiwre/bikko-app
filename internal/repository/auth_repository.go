package repository

import (
	"context"
	"errors"

	"bikko-app/internal/model"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *model.User, password string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
}

type mockAuthRepository struct{}

func NewMockAuthRepository() AuthRepository {
	return &mockAuthRepository{}
}

func (r *mockAuthRepository) CreateUser(ctx context.Context, user *model.User, password string) (*model.User, error) {
	user.ID = "user_123"
	return user, nil
}

func (r *mockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	if email == "unconfirmed@bikko.com.br" {
		return nil, errors.New("unconfirmed_email")
	}
	if email == "error@bikko.com.br" {
		return nil, errors.New("usuário não encontrado")
	}

	return &model.User{
		ID:    "user_123",
		Name:  "Usuário Teste",
		Email: email,
		Phone: "(11) 99999-9999",
	}, nil
}
