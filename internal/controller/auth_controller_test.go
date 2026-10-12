package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/security"
	"bikko-app/internal/service"

	"github.com/gin-gonic/gin"
)

type fakeAuthUserRepo struct {
	byPhone *domain.User
	byEmail *domain.User
	create  func(ctx context.Context, user *domain.User) (*domain.User, error)
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
	return nil, errors.New("not found")
}

func (f *fakeAuthUserRepo) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	if f.byPhone != nil && f.byPhone.Phone == phone {
		return f.byPhone, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeAuthUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, errors.New("not found")
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
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), nil)
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/register", ctrl.Register)

		body, _ := json.Marshal(map[string]interface{}{
			"full_name": "New User",
			"email":     "new@bikko.com",
			"phone":     "+5511999990000",
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
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), nil)
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/register", ctrl.Register)

		body, _ := json.Marshal(map[string]interface{}{
			"full_name": "New User",
			"email":     "dup@bikko.com",
			"phone":     "+5511988887777",
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
				return nil, errors.New("ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)")
			},
		}
		authSvc := service.NewAuthService(repo, security.NewJWTService("secret"), nil)
		ctrl := NewAuthController(authSvc)

		r := gin.New()
		r.POST("/auth/register", ctrl.Register)

		body, _ := json.Marshal(map[string]interface{}{
			"full_name": "New User",
			"email":     "unique@bikko.com",
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
