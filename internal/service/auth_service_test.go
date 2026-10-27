package service

import (
	"context"
	"errors"
	firebaseAuth "firebase.google.com/go/v4/auth"
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
	return nil, domain.ErrNotFound
}

func (m *mockAuthUserRepo) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Phone == phone || u.Phone == "+55"+phone || "+55"+u.Phone == phone {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockAuthUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockAuthUserRepo) UpdateUser(ctx context.Context, id string, name, email, phone string) error {
	for key, u := range m.users {
		if u.ID == id {
			delete(m.users, key)
			u.FullName = name
			u.Email = email
			if phone != "" {
				u.Phone = phone
			}
			m.users[u.Phone] = u
			return nil
		}
	}
	return domain.ErrNotFound
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
	return nil, domain.ErrNotFound
}

type verifiedPhoneToken struct {
	phone   string
	noPhone bool
}

func (v verifiedPhoneToken) VerifyIDToken(context.Context, string) (*firebaseAuth.Token, error) {
	claims := map[string]interface{}{}
	if !v.noPhone {
		claims["phone_number"] = v.phone
	}
	return &firebaseAuth.Token{UID: "verified-firebase-user", Claims: claims}, nil
}
func testRegistration() model.RegisterRequest {
	lat, lon := -23.5614, -46.6914
	return model.RegisterRequest{FullName: "Verified", Email: "verified@example.com", Phone: "11999990000", IDToken: "firebase-valid", IsBikker: true, Profession: "Eletricista", Latitude: &lat, Longitude: &lon}
}
func TestRegistrationRequiresVerifiedPhoneBeforeUpgrade(t *testing.T) {
	t.Setenv("ENV", "production")
	repo := newMockAuthUserRepo()
	repo.users["11999990000"] = &domain.User{ID: "victim", Phone: "11999990000"}
	svc := NewAuthService(repo, security.NewJWTService("test-secret"), nil)
	req := testRegistration()
	req.IDToken = ""
	if _, err := svc.Register(context.Background(), req); !errors.Is(err, ErrInvalidFirebaseToken) {
		t.Fatalf("expected proof required: %v", err)
	}
	if repo.users[req.Phone].IsBikker {
		t.Fatal("unverified caller changed victim account")
	}
}
func TestRegistrationRejectsMissingOrMismatchingPhoneClaims(t *testing.T) {
	for _, tc := range []struct {
		name     string
		verifier verifiedPhoneToken
	}{{"missing claim", verifiedPhoneToken{noPhone: true}}, {"other phone", verifiedPhoneToken{phone: "+5511988880000"}}} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockAuthUserRepo()
			svc := NewAuthService(repo, security.NewJWTService("test-secret"), tc.verifier)
			if _, err := svc.Register(context.Background(), testRegistration()); err == nil {
				t.Fatal("registration accepted an unrelated Firebase identity")
			}
			if len(repo.users) != 0 {
				t.Fatal("invalid identity mutated repository")
			}
		})
	}
}
func TestVerifiedUpgradeReturnsExistingIdentity(t *testing.T) {
	repo := newMockAuthUserRepo()
	repo.users["11999990000"] = &domain.User{ID: "existing", Phone: "11999990000"}
	jwt := security.NewJWTService("test-secret")
	svc := NewAuthService(repo, jwt, verifiedPhoneToken{phone: "+5511999990000"})
	response, err := svc.Register(context.Background(), testRegistration())
	if err != nil {
		t.Fatal(err)
	}
	id, err := jwt.ValidateToken(response.Token)
	if err != nil || id != "existing" || !response.User.IsBikker {
		t.Fatal(response, err)
	}
}
func TestLocalAccountsAreExplicitAndBoundToOnlyTheirOwnPhones(t *testing.T) {
	for _, env := range []string{"", "production", "development"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)
			t.Setenv("GIN_MODE", "debug")
			repo := newMockAuthUserRepo()
			repo.users["11999990000"] = &domain.User{ID: "lucas", Phone: "11999990000"}
			repo.users["11988880000"] = &domain.User{ID: "rafael", Phone: "11988880000", IsBikker: true}
			svc := NewAuthService(repo, security.NewJWTService("test-secret"), nil, WithLocalTestAccounts(true))
			response, err := svc.Login(context.Background(), model.LoginRequest{IDToken: "local_test_11999990000", Phone: "11999990000"})
			if env == "development" {
				if err != nil || response.User.ID != "lucas" {
					t.Fatal(response, err)
				}
			} else if err == nil {
				t.Fatal("test identity enabled outside explicit development")
			}
			for _, token := range []string{"mock_token_123", "test_token", "local_test_11987654321", "local_test_11988880000"} {
				if _, err := svc.Login(context.Background(), model.LoginRequest{IDToken: token, Phone: "11999990000"}); err == nil {
					t.Fatalf("accepted %s for Lucas", token)
				}
			}
		})
	}
	t.Setenv("ENV", "development")
	svc := NewAuthService(newMockAuthUserRepo(), security.NewJWTService("test-secret"), nil)
	if _, err := svc.VerifyIDToken(context.Background(), "local_test_11999990000"); err == nil {
		t.Fatal("local identity enabled without option")
	}
}
func TestCheckPhoneDoesNotSeedOrHideDatabaseFailure(t *testing.T) {
	repo := newMockAuthUserRepo()
	svc := NewAuthService(repo, security.NewJWTService("test-secret"), nil)
	if _, err := svc.CheckPhone(context.Background(), model.CheckPhoneRequest{Phone: "11999990000"}); !errors.Is(err, ErrUserNotFound) {
		t.Fatal(err)
	}
	if len(repo.users) != 0 {
		t.Fatal("phone discovery must be read-only")
	}
}
func TestLoginUsesCanonicalVerifiedPhone(t *testing.T) {
	repo := newMockAuthUserRepo()
	repo.users["11999990000"] = &domain.User{ID: "lucas", Phone: "11999990000"}
	svc := NewAuthService(repo, security.NewJWTService("test-secret"), verifiedPhoneToken{phone: "+55 (11) 99999-0000"})
	result, err := svc.Login(context.Background(), model.LoginRequest{IDToken: "valid", Phone: "(11) 99999-0000"})
	if err != nil || result.User.ID != "lucas" {
		t.Fatal(result, err)
	}
	if _, err := svc.Login(context.Background(), model.LoginRequest{IDToken: "valid", AppType: "bikker"}); !errors.Is(err, ErrClientNeedsBikkerRegistration) {
		t.Fatal(err)
	}
}
