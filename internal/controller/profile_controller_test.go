package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bikko-app/internal/model"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeProfileService struct {
	getProfileFunc    func(ctx context.Context, userID string) (*model.ProfileResponse, error)
	deleteAccountFunc func(ctx context.Context, userID string) error
	updateProfileFunc func(ctx context.Context, userID, name, email, phone string) error
}

func (f *fakeProfileService) GetProfile(ctx context.Context, userID string) (*model.ProfileResponse, error) {
	if f.getProfileFunc != nil {
		return f.getProfileFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (f *fakeProfileService) DeleteAccount(ctx context.Context, userID string) error {
	if f.deleteAccountFunc != nil {
		return f.deleteAccountFunc(ctx, userID)
	}
	return errors.New("not implemented")
}

func (f *fakeProfileService) UpdateProfile(ctx context.Context, userID, name, email, phone string) error {
	if f.updateProfileFunc != nil {
		return f.updateProfileFunc(ctx, userID, name, email, phone)
	}
	return errors.New("not implemented")
}

func TestProfileController_GetProfile(t *testing.T) {
	t.Run("success with authenticated user", func(t *testing.T) {
		svc := &fakeProfileService{
			getProfileFunc: func(ctx context.Context, userID string) (*model.ProfileResponse, error) {
				if userID != "auth_user_123" {
					t.Errorf("expected userID 'auth_user_123', got '%s'", userID)
				}
				return &model.ProfileResponse{
					ID:    userID,
					Name:  "Emerson Torres",
					Email: "emerson@bikko.com.br",
				}, nil
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.GET("/profile", func(c *gin.Context) {
			c.Set("userID", "auth_user_123")
			ctrl.GetProfile(c)
		})

		req, _ := http.NewRequest(http.MethodGet, "/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp model.ProfileResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse json response: %v", err)
		}
		if resp.Name != "Emerson Torres" {
			t.Errorf("expected Name 'Emerson Torres', got '%s'", resp.Name)
		}
	})

	t.Run("fallback to query param userId if unauthenticated", func(t *testing.T) {
		svc := &fakeProfileService{
			getProfileFunc: func(ctx context.Context, userID string) (*model.ProfileResponse, error) {
				if userID != "query_user_456" {
					t.Errorf("expected userID 'query_user_456', got '%s'", userID)
				}
				return &model.ProfileResponse{ID: userID, Name: "Query User"}, nil
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.GET("/profile", ctrl.GetProfile)

		req, _ := http.NewRequest(http.MethodGet, "/profile?userId=query_user_456", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("returns 404 when service returns error", func(t *testing.T) {
		svc := &fakeProfileService{
			getProfileFunc: func(ctx context.Context, userID string) (*model.ProfileResponse, error) {
				return nil, errors.New("user not found")
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.GET("/profile", ctrl.GetProfile)

		req, _ := http.NewRequest(http.MethodGet, "/profile?userId=unknown", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})
}

func TestProfileController_DeleteAccount(t *testing.T) {
	t.Run("success deletion", func(t *testing.T) {
		svc := &fakeProfileService{
			deleteAccountFunc: func(ctx context.Context, userID string) error {
				if userID != "user_del_1" {
					t.Errorf("expected userID 'user_del_1', got '%s'", userID)
				}
				return nil
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.DELETE("/profile", func(c *gin.Context) {
			c.Set("userID", "user_del_1")
			ctrl.DeleteAccount(c)
		})

		req, _ := http.NewRequest(http.MethodDelete, "/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["status"] != "success" {
			t.Errorf("expected status 'success', got '%s'", resp["status"])
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		svc := &fakeProfileService{
			deleteAccountFunc: func(ctx context.Context, userID string) error {
				return errors.New("database locked")
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.DELETE("/profile", ctrl.DeleteAccount)

		req, _ := http.NewRequest(http.MethodDelete, "/profile?userId=user_error", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})
}

func TestProfileController_UpdateProfile(t *testing.T) {
	t.Run("returns 401 Unauthorized if no userID", func(t *testing.T) {
		ctrl := NewProfileController(&fakeProfileService{})
		router := gin.New()
		router.PUT("/profile", ctrl.UpdateProfile)

		body, _ := json.Marshal(UpdateProfileRequest{Name: "Test", Email: "test@example.com"})
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("returns 400 Bad Request if validation fails", func(t *testing.T) {
		ctrl := NewProfileController(&fakeProfileService{})
		router := gin.New()
		router.PUT("/profile", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfile(c)
		})

		// Missing name and invalid email
		invalidBody := []byte(`{"email":"not-an-email"}`)
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(invalidBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("success updates profile", func(t *testing.T) {
		var receivedName, receivedEmail, receivedPhone string
		svc := &fakeProfileService{
			updateProfileFunc: func(ctx context.Context, userID, name, email, phone string) error {
				if userID != "user_123" {
					t.Errorf("expected userID 'user_123', got '%s'", userID)
				}
				receivedName = name
				receivedEmail = email
				receivedPhone = phone
				return nil
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.PUT("/profile", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfile(c)
		})

		payload := UpdateProfileRequest{
			Name:  "Novo Nome",
			Email: "novo@bikko.com.br",
			Phone: "(11) 98765-4321",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		if receivedName != "Novo Nome" || receivedEmail != "novo@bikko.com.br" || receivedPhone != "(11) 98765-4321" {
			t.Errorf("values mismatch: got %s, %s, %s", receivedName, receivedEmail, receivedPhone)
		}
	})

	t.Run("returns 500 when update fails in service", func(t *testing.T) {
		svc := &fakeProfileService{
			updateProfileFunc: func(ctx context.Context, userID, name, email, phone string) error {
				return errors.New("internal database error")
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.PUT("/profile", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfile(c)
		})

		payload := UpdateProfileRequest{
			Name:  "Novo Nome",
			Email: "novo@bikko.com.br",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})
}
