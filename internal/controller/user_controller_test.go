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

	"github.com/gin-gonic/gin"
)

type mockUserRepoForController struct {
	updateDeviceTokenFunc func(ctx context.Context, id string, token *string) error
}

func (m *mockUserRepoForController) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserRepoForController) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserRepoForController) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserRepoForController) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserRepoForController) UpdateUser(ctx context.Context, id string, name, email, phone string) error {
	return errors.New("not implemented")
}

func (m *mockUserRepoForController) DeleteUser(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

func (m *mockUserRepoForController) UpdateDeviceToken(ctx context.Context, id string, token *string) error {
	if m.updateDeviceTokenFunc != nil {
		return m.updateDeviceTokenFunc(ctx, id, token)
	}
	return nil
}

func (m *mockUserRepoForController) GetDeviceToken(ctx context.Context, id string) (string, error) {
	return "", errors.New("not implemented")
}

func setupUserRouter(ctrl *UserController, userID string) *gin.Engine {
	r := gin.New()
	if userID != "" {
		r.Use(func(c *gin.Context) {
			c.Set("userID", userID)
			c.Next()
		})
	}
	r.POST("/api/v1/users/device-token", ctrl.SaveDeviceToken)
	r.DELETE("/api/v1/users/device-token", ctrl.DeleteDeviceToken)
	return r
}

func TestUserController_SaveDeviceToken(t *testing.T) {
	t.Run("success when authenticated", func(t *testing.T) {
		var capturedID string
		var capturedToken string

		repo := &mockUserRepoForController{
			updateDeviceTokenFunc: func(ctx context.Context, id string, token *string) error {
				capturedID = id
				if token != nil {
					capturedToken = *token
				}
				return nil
			},
		}

		ctrl := NewUserController(repo)
		router := setupUserRouter(ctrl, "usr_test_123")

		payload := []byte(`{"token":"fcm_device_token_xyz"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/device-token", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		if capturedID != "usr_test_123" {
			t.Errorf("expected userID usr_test_123, got %s", capturedID)
		}
		if capturedToken != "fcm_device_token_xyz" {
			t.Errorf("expected token fcm_device_token_xyz, got %s", capturedToken)
		}

		var resp map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["status"] != "success" {
			t.Errorf("expected status success, got %s", resp["status"])
		}
	})

	t.Run("unauthorized when user is not authenticated", func(t *testing.T) {
		repo := &mockUserRepoForController{}
		ctrl := NewUserController(repo)
		router := setupUserRouter(ctrl, "") // No user ID in context

		payload := []byte(`{"token":"fcm_device_token_xyz"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/device-token", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("bad request when token is empty or missing", func(t *testing.T) {
		repo := &mockUserRepoForController{}
		ctrl := NewUserController(repo)
		router := setupUserRouter(ctrl, "usr_test_123")

		payload := []byte(`{}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/device-token", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("internal server error when repository fails", func(t *testing.T) {
		repo := &mockUserRepoForController{
			updateDeviceTokenFunc: func(ctx context.Context, id string, token *string) error {
				return errors.New("db error")
			},
		}

		ctrl := NewUserController(repo)
		router := setupUserRouter(ctrl, "usr_test_123")

		payload := []byte(`{"token":"fcm_device_token_xyz"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/device-token", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 Internal Server Error, got %d", w.Code)
		}
	})
}

func TestUserController_DeleteDeviceToken(t *testing.T) {
	t.Run("success when authenticated", func(t *testing.T) {
		var capturedID string
		var tokenWasNil bool

		repo := &mockUserRepoForController{
			updateDeviceTokenFunc: func(ctx context.Context, id string, token *string) error {
				capturedID = id
				tokenWasNil = (token == nil)
				return nil
			},
		}

		ctrl := NewUserController(repo)
		router := setupUserRouter(ctrl, "usr_test_123")

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/users/device-token", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		if capturedID != "usr_test_123" {
			t.Errorf("expected userID usr_test_123, got %s", capturedID)
		}
		if !tokenWasNil {
			t.Errorf("expected token pointer to be nil")
		}
	})

	t.Run("unauthorized when not authenticated", func(t *testing.T) {
		repo := &mockUserRepoForController{}
		ctrl := NewUserController(repo)
		router := setupUserRouter(ctrl, "")

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/users/device-token", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("internal server error when repository fails", func(t *testing.T) {
		repo := &mockUserRepoForController{
			updateDeviceTokenFunc: func(ctx context.Context, id string, token *string) error {
				return errors.New("db error")
			},
		}

		ctrl := NewUserController(repo)
		router := setupUserRouter(ctrl, "usr_test_123")

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/users/device-token", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 Internal Server Error, got %d", w.Code)
		}
	})
}
