package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	firebaseAuth "firebase.google.com/go/v4/auth"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/security"
	"bikko-app/internal/service"

	"github.com/gin-gonic/gin"
)

type fakeAuthUserRepo struct {
	byPhone         *domain.User
	byEmail         *domain.User
	create          func(ctx context.Context, user *domain.User) (*domain.User, error)
	upgradeToBikker func(ctx context.Context, userID string, bikker *domain.Bikker) (*domain.User, error)
}

func (f *fakeAuthUserRepo) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	if f.create != nil {
		return f.create(ctx, user)
	}
	user.ID = "user-new-1"
	return user, nil
}

func (f *fakeAuthUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if f.byEmail != nil && f.byEmail.Email == email {
		return f.byEmail, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeAuthUserRepo) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	if f.byPhone != nil {
		canonical, _ := domain.NormalizePhone(f.byPhone.Phone)
		if canonical == phone {
			return f.byPhone, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeAuthUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeAuthUserRepo) UpdateUser(ctx context.Context, id string, name, email, phone string) error {
	return nil
}

func (f *fakeAuthUserRepo) UpdateProfilePhoto(ctx context.Context, id string, photoURL string) error {
	return nil
}

func (f *fakeAuthUserRepo) DeleteUser(ctx context.Context, id string) error {
	return nil
}

func (f *fakeAuthUserRepo) UpdateDeviceToken(ctx context.Context, id string, token *string) error {
	return nil
}

func (f *fakeAuthUserRepo) GetDeviceToken(ctx context.Context, id string) (string, error) {
	return "", nil
}

func (f *fakeAuthUserRepo) UpgradeToBikker(ctx context.Context, userID string, bikker *domain.Bikker) (*domain.User, error) {
	if f.upgradeToBikker != nil {
		return f.upgradeToBikker(ctx, userID, bikker)
	}
	if f.byPhone != nil && f.byPhone.ID == userID {
		f.byPhone.IsBikker = true
		f.byPhone.Bikker = bikker
		return f.byPhone, nil
	}
	return &domain.User{
		ID:       userID,
		FullName: "Upgraded User",
		IsBikker: true,
		Bikker:   bikker,
	}, nil
}

func TestAuthController_RegisterDuplicateDetection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns 409 Conflict when phone already exists", func(t *testing.T) {
		repo := &fakeAuthUserRepo{
			byPhone: &domain.User{
				ID:       "existing-1",
				FullName: "Existing User",
				Phone:    "+5511999990000",
			},
		}
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), controllerPhoneVerifier{})
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/register", ctrl.Register)

		body, _ := json.Marshal(map[string]interface{}{
			"full_name": "New User",
			"email":     "new@bikko.com",
			"phone":     "+5511999990000",
			"id_token":  "verified_11999990000",
		})
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["code"] != "USER_ALREADY_EXISTS" || resp["error"] != "user_already_exists" {
			t.Errorf("unexpected conflict payload: %+v", resp)
		}
	})

	t.Run("returns 409 Conflict when email already exists", func(t *testing.T) {
		repo := &fakeAuthUserRepo{
			byEmail: &domain.User{
				ID:       "existing-2",
				FullName: "Existing Email User",
				Email:    "dup@bikko.com",
			},
		}
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), controllerPhoneVerifier{})
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/register", ctrl.Register)

		body, _ := json.Marshal(map[string]interface{}{
			"full_name": "New User",
			"email":     "dup@bikko.com",
			"phone":     "+5511988887777",
			"id_token":  "verified_11988887777",
		})
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d", w.Code)
		}
	})

	t.Run("returns 409 Conflict when CreateUser returns unique constraint 23505 error", func(t *testing.T) {
		repo := &fakeAuthUserRepo{
			create: func(ctx context.Context, user *domain.User) (*domain.User, error) {
				return nil, domain.ErrConflict
			},
		}
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), controllerPhoneVerifier{})
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/register", ctrl.Register)

		body, _ := json.Marshal(map[string]interface{}{
			"full_name": "New User",
			"email":     "unique@bikko.com",
			"id_token":  "verified_11999990000",
		})
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d", w.Code)
		}
	})
}

func TestAuthController_CheckPhone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns a generic response for an unregistered number", func(t *testing.T) {
		repo := &fakeAuthUserRepo{}
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), controllerPhoneVerifier{})
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/check-phone", ctrl.CheckPhone)

		body, _ := json.Marshal(map[string]interface{}{
			"phone":    "11977776666",
			"app_type": "client",
		})
		req, _ := http.NewRequest(http.MethodPost, "/auth/check-phone", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["exists"] != true || resp["is_bikker"] != false {
			t.Errorf("unexpected response body: %+v", resp)
		}
	})

	t.Run("returns 200 OK for Client number on client app", func(t *testing.T) {
		repo := &fakeAuthUserRepo{
			byPhone: &domain.User{
				ID:       "client-1",
				FullName: "Cliente Teste",
				Phone:    "11999990000",
				IsBikker: false,
			},
		}
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), controllerPhoneVerifier{})
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/check-phone", ctrl.CheckPhone)

		body, _ := json.Marshal(map[string]interface{}{
			"phone":    "11999990000",
			"app_type": "client",
		})
		req, _ := http.NewRequest(http.MethodPost, "/auth/check-phone", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["exists"] != true || resp["is_bikker"] != false {
			t.Errorf("unexpected response body: %+v", resp)
		}
	})

	t.Run("does not reveal client role to provider app", func(t *testing.T) {
		repo := &fakeAuthUserRepo{
			byPhone: &domain.User{
				ID:       "client-1",
				FullName: "Cliente Teste",
				Phone:    "11999990000",
				IsBikker: false,
			},
		}
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), controllerPhoneVerifier{})
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/check-phone", ctrl.CheckPhone)

		body, _ := json.Marshal(map[string]interface{}{
			"phone":    "11999990000",
			"app_type": "bikker",
		})
		req, _ := http.NewRequest(http.MethodPost, "/auth/check-phone", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["exists"] != true || resp["is_bikker"] != false {
			t.Errorf("unexpected response body: %+v", resp)
		}
	})

	t.Run("does not reveal provider role on either app", func(t *testing.T) {
		repo := &fakeAuthUserRepo{
			byPhone: &domain.User{
				ID:       "bikker-1",
				FullName: "Bikker Profissional",
				Phone:    "11988880000",
				IsBikker: true,
			},
		}
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), controllerPhoneVerifier{})
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/check-phone", ctrl.CheckPhone)

		// On bikker app
		bodyBikker, _ := json.Marshal(map[string]interface{}{
			"phone":    "11988880000",
			"app_type": "bikker",
		})
		reqBikker, _ := http.NewRequest(http.MethodPost, "/auth/check-phone", bytes.NewBuffer(bodyBikker))
		reqBikker.Header.Set("Content-Type", "application/json")
		wBikker := httptest.NewRecorder()
		r.ServeHTTP(wBikker, reqBikker)

		if wBikker.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on bikker app, got %d: %s", wBikker.Code, wBikker.Body.String())
		}

		var respBikker map[string]interface{}
		_ = json.Unmarshal(wBikker.Body.Bytes(), &respBikker)
		if respBikker["exists"] != true || respBikker["is_bikker"] != false {
			t.Errorf("unexpected bikker response: %+v", respBikker)
		}

		// On client app
		bodyClient, _ := json.Marshal(map[string]interface{}{
			"phone":    "11988880000",
			"app_type": "client",
		})
		reqClient, _ := http.NewRequest(http.MethodPost, "/auth/check-phone", bytes.NewBuffer(bodyClient))
		reqClient.Header.Set("Content-Type", "application/json")
		wClient := httptest.NewRecorder()
		r.ServeHTTP(wClient, reqClient)

		if wClient.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on client app, got %d: %s", wClient.Code, wClient.Body.String())
		}

		var respClient map[string]interface{}
		_ = json.Unmarshal(wClient.Body.Bytes(), &respClient)
		if respClient["exists"] != true || respClient["is_bikker"] != false {
			t.Errorf("unexpected client response: %+v", respClient)
		}
	})
}

func TestAuthController_RegisterUpgradeToBikker(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns 201 Created and upgrades existing client to bikker", func(t *testing.T) {
		repo := &fakeAuthUserRepo{
			byPhone: &domain.User{
				ID:       "client-existing-1",
				FullName: "Cliente Existente",
				Phone:    "11999990000",
				Email:    "cliente@bikko.com",
				IsBikker: false,
			},
		}
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), controllerPhoneVerifier{})
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/register", ctrl.Register)

		body, _ := json.Marshal(map[string]interface{}{
			"full_name":        "Cliente Existente",
			"phone":            "11999990000",
			"id_token":         "verified_11999990000",
			"latitude":         -23.56,
			"longitude":        -46.69,
			"is_bikker":        true,
			"profession":       "Pintor Profissional",
			"experience_years": "5 anos",
			"description":      "Serviços de pintura e acabamento",
			"location_name":    "São Paulo, SP",
		})
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		userMap, ok := resp["user"].(map[string]interface{})
		if !ok || userMap["is_bikker"] != true {
			t.Errorf("expected user.is_bikker to be true, got %+v", resp)
		}
	})
}

func TestAuthController_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns 403 CLIENT_NEEDS_BIKKER_REGISTRATION when client logs into bikker app", func(t *testing.T) {
		repo := &fakeAuthUserRepo{
			byPhone: &domain.User{
				ID:       "client-1",
				FullName: "Cliente Somente",
				Phone:    "11999990000",
				IsBikker: false,
			},
		}
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), controllerPhoneVerifier{})
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/login", ctrl.Login)

		body, _ := json.Marshal(map[string]interface{}{
			"id_token": "verified_11999990000",
			"phone":    "11999990000",
			"app_type": "bikker",
		})
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["code"] != "CLIENT_NEEDS_BIKKER_REGISTRATION" {
			t.Errorf("expected code CLIENT_NEEDS_BIKKER_REGISTRATION, got %+v", resp)
		}
	})
}

type controllerPhoneVerifier struct{}

func (controllerPhoneVerifier) VerifyIDToken(ctx context.Context, token string) (*firebaseAuth.Token, error) {
	if !strings.HasPrefix(token, "verified_") {
		return nil, errors.New("invalid test proof")
	}
	return &firebaseAuth.Token{UID: "verified-user", Claims: map[string]interface{}{"phone_number": "+55" + strings.TrimPrefix(token, "verified_")}}, nil
}
func TestAuthControllerUnverifiedRegistrationCannotUpgradeClient(t *testing.T) {
	repo := &fakeAuthUserRepo{byPhone: &domain.User{ID: "victim", Phone: "11999990000"}}
	ctrl := NewAuthController(service.NewAuthService(repo, security.NewJWTService("test-secret"), nil))
	r := gin.New()
	r.POST("/auth/register", ctrl.Register)
	req := httptest.NewRequest("POST", "/auth/register", strings.NewReader(`{"full_name":"Caller","phone":"11999990000","is_bikker":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized || repo.byPhone.IsBikker {
		t.Fatal(w.Code, w.Body.String())
	}
}
