package service

import (
	"context"
	"errors"
	"os"
	"testing"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/security"
	"bikko-app/internal/model"
)

type mockAuthUserRepo struct {
	users           map[string]*domain.User
	createFn        func(ctx context.Context, user *domain.User) (*domain.User, error)
	upgradeToBikker func(ctx context.Context, userID string, bikker *domain.Bikker) (*domain.User, error)
}

func newMockAuthUserRepo() *mockAuthUserRepo {
	return &mockAuthUserRepo{
		users: make(map[string]*domain.User),
	}
}

func (m *mockAuthUserRepo) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	if user.ID == "" {
		user.ID = "user-" + user.Phone
	}
	m.users[user.Phone] = user
	return user, nil
}

func (m *mockAuthUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockAuthUserRepo) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Phone == phone || u.Phone == "+55"+phone || "+55"+u.Phone == phone {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockAuthUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockAuthUserRepo) UpdateUser(ctx context.Context, id string, name, email, phone string) error {
	return nil
}

func (m *mockAuthUserRepo) UpdateProfilePhoto(ctx context.Context, id string, photoURL string) error {
	return nil
}

func (m *mockAuthUserRepo) DeleteUser(ctx context.Context, id string) error {
	return nil
}

func (m *mockAuthUserRepo) UpdateDeviceToken(ctx context.Context, id string, token *string) error {
	return nil
}

func (m *mockAuthUserRepo) GetDeviceToken(ctx context.Context, id string) (string, error) {
	return "", nil
}

func (m *mockAuthUserRepo) UpgradeToBikker(ctx context.Context, userID string, bikker *domain.Bikker) (*domain.User, error) {
	if m.upgradeToBikker != nil {
		return m.upgradeToBikker(ctx, userID, bikker)
	}
	for _, u := range m.users {
		if u.ID == userID {
			u.IsBikker = true
			u.Bikker = bikker
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func TestAuthService_CheckPhone(t *testing.T) {
	_ = os.Setenv("ENV", "development")
	_ = os.Setenv("GIN_MODE", "debug")
	jwtSvc := security.NewJWTService("secret")

	t.Run("returns ErrInvalidCredentials when phone is empty", func(t *testing.T) {
		repo := newMockAuthUserRepo()
		svc := NewAuthService(repo, jwtSvc, nil)

		_, err := svc.CheckPhone(context.Background(), model.CheckPhoneRequest{Phone: ""})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("returns ErrUserNotFound for unregistered phone", func(t *testing.T) {
		repo := newMockAuthUserRepo()
		svc := NewAuthService(repo, jwtSvc, nil)

		_, err := svc.CheckPhone(context.Background(), model.CheckPhoneRequest{
			Phone:   "11911112222",
			AppType: "client",
		})
		if !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("auto-seeds test client 11999990000 and allows login on client app", func(t *testing.T) {
		repo := newMockAuthUserRepo()
		svc := NewAuthService(repo, jwtSvc, nil)

		resp, err := svc.CheckPhone(context.Background(), model.CheckPhoneRequest{
			Phone:   "11999990000",
			AppType: "client",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !resp.Exists || resp.IsBikker {
			t.Fatalf("expected Exists=true and IsBikker=false, got %+v", resp)
		}
	})

	t.Run("auto-seeds test client 11999990000 and blocks on bikker app with ErrClientNeedsBikkerRegistration", func(t *testing.T) {
		repo := newMockAuthUserRepo()
		svc := NewAuthService(repo, jwtSvc, nil)

		resp, err := svc.CheckPhone(context.Background(), model.CheckPhoneRequest{
			Phone:   "11999990000",
			AppType: "bikker",
		})
		if !errors.Is(err, ErrClientNeedsBikkerRegistration) {
			t.Fatalf("expected ErrClientNeedsBikkerRegistration, got %v", err)
		}
		if resp == nil || !resp.Exists || resp.IsBikker {
			t.Fatalf("expected response with Exists=true and IsBikker=false, got %+v", resp)
		}
	})

	t.Run("auto-seeds test bikker 11988880000 and allows on both bikker and client apps", func(t *testing.T) {
		repo := newMockAuthUserRepo()
		svc := NewAuthService(repo, jwtSvc, nil)

		// Bikker app check
		respBikker, err := svc.CheckPhone(context.Background(), model.CheckPhoneRequest{
			Phone:   "11988880000",
			AppType: "bikker",
		})
		if err != nil {
			t.Fatalf("expected no error on bikker app, got %v", err)
		}
		if !respBikker.Exists || !respBikker.IsBikker {
			t.Fatalf("expected Exists=true and IsBikker=true, got %+v", respBikker)
		}

		// Client app check
		respClient, err := svc.CheckPhone(context.Background(), model.CheckPhoneRequest{
			Phone:   "11988880000",
			AppType: "client",
		})
		if err != nil {
			t.Fatalf("expected no error on client app, got %v", err)
		}
		if !respClient.Exists || !respClient.IsBikker {
			t.Fatalf("expected Exists=true and IsBikker=true, got %+v", respClient)
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	_ = os.Setenv("ENV", "development")
	_ = os.Setenv("GIN_MODE", "debug")
	jwtSvc := security.NewJWTService("secret")

	t.Run("client user is blocked when logging into bikker app", func(t *testing.T) {
		repo := newMockAuthUserRepo()
		repo.users["11999990000"] = &domain.User{
			ID:       "client-1",
			FullName: "Cliente Teste",
			Phone:    "11999990000",
			IsBikker: false,
		}
		svc := NewAuthService(repo, jwtSvc, nil)

		_, err := svc.Login(context.Background(), model.LoginRequest{
			IDToken: "test_token",
			Phone:   "11999990000",
			AppType: "bikker",
		})
		if !errors.Is(err, ErrClientNeedsBikkerRegistration) {
			t.Fatalf("expected ErrClientNeedsBikkerRegistration, got %v", err)
		}
	})

	t.Run("client user can log into client app", func(t *testing.T) {
		repo := newMockAuthUserRepo()
		repo.users["11999990000"] = &domain.User{
			ID:       "client-1",
			FullName: "Cliente Teste",
			Phone:    "11999990000",
			IsBikker: false,
		}
		svc := NewAuthService(repo, jwtSvc, nil)

		resp, err := svc.Login(context.Background(), model.LoginRequest{
			IDToken: "test_token",
			Phone:   "11999990000",
			AppType: "client",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.Token == "" || resp.User.ID != "client-1" {
			t.Fatalf("unexpected login response: %+v", resp)
		}
	})

	t.Run("bikker user can log into both bikker and client apps", func(t *testing.T) {
		repo := newMockAuthUserRepo()
		repo.users["11988880000"] = &domain.User{
			ID:       "bikker-1",
			FullName: "Bikker Profissional",
			Phone:    "11988880000",
			IsBikker: true,
		}
		svc := NewAuthService(repo, jwtSvc, nil)

		respBikker, err := svc.Login(context.Background(), model.LoginRequest{
			IDToken: "test_token",
			Phone:   "11988880000",
			AppType: "bikker",
		})
		if err != nil {
			t.Fatalf("expected bikker login to succeed, got %v", err)
		}
		if respBikker.Token == "" || !respBikker.User.IsBikker {
			t.Fatalf("unexpected bikker response: %+v", respBikker)
		}

		respClient, err := svc.Login(context.Background(), model.LoginRequest{
			IDToken: "test_token",
			Phone:   "11988880000",
			AppType: "client",
		})
		if err != nil {
			t.Fatalf("expected client login to succeed for bikker, got %v", err)
		}
		if respClient.Token == "" {
			t.Fatalf("unexpected client response: %+v", respClient)
		}
	})
}

func TestAuthService_RegisterUpgradeToBikker(t *testing.T) {
	jwtSvc := security.NewJWTService("secret")

	t.Run("upgrades existing client to bikker when is_bikker=true", func(t *testing.T) {
		repo := newMockAuthUserRepo()
		repo.users["11999990000"] = &domain.User{
			ID:       "client-to-upgrade",
			FullName: "Cliente Existente",
			Phone:    "11999990000",
			Email:    "cliente@bikko.com",
			IsBikker: false,
		}
		svc := NewAuthService(repo, jwtSvc, nil)

		lat := -23.55
		lon := -46.63
		resp, err := svc.Register(context.Background(), model.RegisterRequest{
			FullName:        "Cliente Existente",
			Phone:           "11999990000",
			IsBikker:        true,
			Profession:      "Encanador Residencial",
			ExperienceYears: "3 a 5 anos",
			Description:     "Especialista em vazamentos e encanamento predial.",
			LocationName:    "São Paulo, SP",
			Latitude:        &lat,
			Longitude:       &lon,
		})
		if err != nil {
			t.Fatalf("expected registration/upgrade to succeed, got %v", err)
		}

		if resp.Message != "Cadastro de Bikker vinculado com sucesso" {
			t.Errorf("unexpected message: %s", resp.Message)
		}
		if !resp.User.IsBikker {
			t.Errorf("expected User.IsBikker to be true")
		}
		if repo.users["11999990000"].IsBikker != true {
			t.Errorf("expected repo user to be upgraded to bikker")
		}
	})

	t.Run("returns ErrUserAlreadyExists when existing client registers again as client", func(t *testing.T) {
		repo := newMockAuthUserRepo()
		repo.users["11999990000"] = &domain.User{
			ID:       "client-1",
			FullName: "Cliente Existente",
			Phone:    "11999990000",
			IsBikker: false,
		}
		svc := NewAuthService(repo, jwtSvc, nil)

		_, err := svc.Register(context.Background(), model.RegisterRequest{
			FullName: "Cliente Existente",
			Phone:    "11999990000",
			IsBikker: false,
		})
		if !errors.Is(err, ErrUserAlreadyExists) {
			t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
		}
	})

	t.Run("returns ErrUserAlreadyExists when existing bikker registers again", func(t *testing.T) {
		repo := newMockAuthUserRepo()
		repo.users["11988880000"] = &domain.User{
			ID:       "bikker-1",
			FullName: "Bikker Existente",
			Phone:    "11988880000",
			IsBikker: true,
		}
		svc := NewAuthService(repo, jwtSvc, nil)

		_, err := svc.Register(context.Background(), model.RegisterRequest{
			FullName:   "Bikker Existente",
			Phone:      "11988880000",
			IsBikker:   true,
			Profession: "Eletricista",
		})
		if !errors.Is(err, ErrUserAlreadyExists) {
			t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
		}
	})
}
