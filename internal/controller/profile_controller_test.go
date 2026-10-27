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
	"bikko-app/internal/model"
	"bikko-app/internal/service"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeProfileService struct {
	getProfileFunc             func(ctx context.Context, userID string) (*model.ProfileResponse, error)
	deleteAccountFunc          func(ctx context.Context, userID string) error
	updateProfileFunc          func(ctx context.Context, userID, name, email, phone string) error
	updateProfileWithTokenFunc func(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error)
	updateProfilePhotoFunc     func(ctx context.Context, userID, photoURL string) error
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

func (f *fakeProfileService) UpdateProfileWithToken(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error) {
	if f.updateProfileWithTokenFunc != nil {
		return f.updateProfileWithTokenFunc(ctx, userID, name, email, phone, firebaseToken)
	}
	if f.updateProfileFunc != nil {
		return "", f.updateProfileFunc(ctx, userID, name, email, phone)
	}
	return "", errors.New("not implemented")
}

func (f *fakeProfileService) UpdateProfilePhoto(ctx context.Context, userID, photoURL string) error {
	if f.updateProfilePhotoFunc != nil {
		return f.updateProfilePhotoFunc(ctx, userID, photoURL)
	}
	return nil
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

	t.Run("query param cannot select unauthenticated identity", func(t *testing.T) {
		called := false
		svc := &fakeProfileService{getProfileFunc: func(context.Context, string) (*model.ProfileResponse, error) { called = true; return nil, nil }}
		ctrl := NewProfileController(svc)
		router := gin.New()
		router.GET("/profile", ctrl.GetProfile)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/profile?userId=victim", nil))
		if w.Code != http.StatusUnauthorized || called {
			t.Fatalf("unauthenticated profile lookup: %d", w.Code)
		}
	})

	t.Run("returns 404 when service returns error", func(t *testing.T) {
		svc := &fakeProfileService{
			getProfileFunc: func(ctx context.Context, userID string) (*model.ProfileResponse, error) {
				return nil, domain.ErrNotFound
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.GET("/profile", func(c *gin.Context) { c.Set("userID", "unknown"); ctrl.GetProfile(c) })

		req, _ := http.NewRequest(http.MethodGet, "/profile", nil)
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
		router.DELETE("/profile", func(c *gin.Context) { c.Set("userID", "user_error"); ctrl.DeleteAccount(c) })

		req, _ := http.NewRequest(http.MethodDelete, "/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("returns 401 Unauthorized when userID is empty", func(t *testing.T) {
		ctrl := NewProfileController(&fakeProfileService{})
		router := gin.New()
		router.DELETE("/profile", ctrl.DeleteAccount)

		req, _ := http.NewRequest(http.MethodDelete, "/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
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

	t.Run("success updates profile when phone does not change (no token in response)", func(t *testing.T) {
		svc := &fakeProfileService{
			updateProfileWithTokenFunc: func(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error) {
				if userID != "user_123" {
					t.Errorf("expected userID 'user_123', got '%s'", userID)
				}
				if firebaseToken != "" {
					t.Errorf("expected empty firebaseToken, got '%s'", firebaseToken)
				}
				return "", nil
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.PUT("/profile", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfile(c)
		})

		payload := UpdateProfileRequest{
			Name:  "Emerson Torres",
			Email: "emerson@bikko.com.br",
			Phone: "(11) 98888-7777",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["status"] != "success" {
			t.Errorf("expected status 'success', got %v", resp["status"])
		}
		if _, hasToken := resp["token"]; hasToken {
			t.Errorf("expected response not to contain token, got %v", resp["token"])
		}
	})

	t.Run("success updates profile when phone changes with valid firebase_token returns refreshed JWT", func(t *testing.T) {
		svc := &fakeProfileService{
			updateProfileWithTokenFunc: func(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error) {
				if firebaseToken != "mock_token_123" {
					t.Errorf("expected firebaseToken 'mock_token_123', got '%s'", firebaseToken)
				}
				return "jwt_refreshed_token_abc", nil
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.PUT("/profile", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfile(c)
		})

		payload := UpdateProfileRequest{
			Name:          "Emerson Torres",
			Email:         "emerson@bikko.com.br",
			Phone:         "(11) 99999-0000",
			FirebaseToken: "mock_token_123",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["status"] != "success" {
			t.Errorf("expected status 'success', got %v", resp["status"])
		}
		if resp["token"] != "jwt_refreshed_token_abc" {
			t.Errorf("expected token 'jwt_refreshed_token_abc', got %v", resp["token"])
		}
	})

	t.Run("returns 401 Unauthorized when phone change requires firebase token", func(t *testing.T) {
		svc := &fakeProfileService{
			updateProfileWithTokenFunc: func(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error) {
				return "", service.ErrPhoneChangeRequiresFirebaseToken
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.PUT("/profile", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfile(c)
		})

		payload := UpdateProfileRequest{
			Name:  "Emerson Torres",
			Email: "emerson@bikko.com.br",
			Phone: "(11) 99999-0000",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d. Body: %s", w.Code, w.Body.String())
		}
		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["error"] != "FIREBASE_TOKEN_INVALID" {
			t.Errorf("expected error 'FIREBASE_TOKEN_INVALID', got %q", resp["error"])
		}
		if resp["message"] != "Autenticação do novo número via Firebase é obrigatória" {
			t.Errorf("expected message 'Autenticação do novo número via Firebase é obrigatória', got %q", resp["message"])
		}
	})

	t.Run("returns 401 Unauthorized when firebase token is invalid", func(t *testing.T) {
		svc := &fakeProfileService{
			updateProfileWithTokenFunc: func(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error) {
				return "", service.ErrInvalidFirebaseToken
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.PUT("/profile", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfile(c)
		})

		payload := UpdateProfileRequest{
			Name:          "Emerson Torres",
			Email:         "emerson@bikko.com.br",
			Phone:         "(11) 99999-0000",
			FirebaseToken: "invalid_token",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d. Body: %s", w.Code, w.Body.String())
		}
		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["error"] != "FIREBASE_TOKEN_INVALID" {
			t.Errorf("expected error 'FIREBASE_TOKEN_INVALID', got %q", resp["error"])
		}
	})

	t.Run("returns 409 Conflict when phone already in use", func(t *testing.T) {
		svc := &fakeProfileService{
			updateProfileWithTokenFunc: func(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error) {
				return "", service.ErrPhoneAlreadyInUse
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.PUT("/profile", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfile(c)
		})

		payload := UpdateProfileRequest{
			Name:          "Emerson Torres",
			Email:         "emerson@bikko.com.br",
			Phone:         "(11) 99999-0000",
			FirebaseToken: "mock_token_123",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected status 409, got %d. Body: %s", w.Code, w.Body.String())
		}
		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["error"] != "PHONE_ALREADY_IN_USE" {
			t.Errorf("expected error 'PHONE_ALREADY_IN_USE', got %q", resp["error"])
		}
		if resp["message"] != "Este número de telefone já está vinculado a outra conta" {
			t.Errorf("expected message 'Este número de telefone já está vinculado a outra conta', got %q", resp["message"])
		}
	})

	t.Run("returns 409 Conflict if email is already in use", func(t *testing.T) {
		svc := &fakeProfileService{
			updateProfileWithTokenFunc: func(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error) {
				return "", service.ErrEmailAlreadyInUse
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.PUT("/profile", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfile(c)
		})

		payload := UpdateProfileRequest{
			Name:  "Emerson Torres",
			Email: "duplicated@bikko.com.br",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected status 409, got %d. Body: %s", w.Code, w.Body.String())
		}
		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["error"] != "EMAIL_ALREADY_IN_USE" {
			t.Errorf("expected error 'EMAIL_ALREADY_IN_USE', got %q", resp["error"])
		}
		if resp["message"] != "Este e-mail já está vinculado a outra conta" {
			t.Errorf("expected message 'Este e-mail já está vinculado a outra conta', got %q", resp["message"])
		}
	})

	t.Run("allows empty email for phone-only registered users", func(t *testing.T) {
		var receivedEmail string
		svc := &fakeProfileService{
			updateProfileWithTokenFunc: func(ctx context.Context, userID, name, email, phone, firebaseToken string) (string, error) {
				receivedEmail = email
				return "", nil
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.PUT("/profile", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfile(c)
		})

		payload := UpdateProfileRequest{
			Name:  "Nome Sem Email",
			Email: "",
			Phone: "11999990000",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
		if receivedEmail != "" {
			t.Errorf("expected empty email, got %q", receivedEmail)
		}
	})
}

func TestProfileController_UpdateProfilePhoto(t *testing.T) {
	t.Run("returns 401 Unauthorized if no userID", func(t *testing.T) {
		ctrl := NewProfileController(&fakeProfileService{})
		router := gin.New()
		router.PUT("/profile/photo", ctrl.UpdateProfilePhoto)

		body, _ := json.Marshal(UpdateProfilePhotoRequest{PhotoURL: "https://cdn.bikko.com.br/avatar.jpg"})
		req, _ := http.NewRequest(http.MethodPut, "/profile/photo", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("returns 400 Bad Request if photo_url is invalid", func(t *testing.T) {
		ctrl := NewProfileController(&fakeProfileService{})
		router := gin.New()
		router.PUT("/profile/photo", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfilePhoto(c)
		})

		invalidBody := []byte(`{"photo_url":"not-a-valid-url"}`)
		req, _ := http.NewRequest(http.MethodPut, "/profile/photo", bytes.NewBuffer(invalidBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("success updates profile photo", func(t *testing.T) {
		var savedUserID, savedURL string
		svc := &fakeProfileService{
			updateProfilePhotoFunc: func(ctx context.Context, userID, photoURL string) error {
				savedUserID = userID
				savedURL = photoURL
				return nil
			},
		}

		ctrl := NewProfileController(svc)
		router := gin.New()
		router.PUT("/profile/photo", func(c *gin.Context) {
			c.Set("userID", "user_123")
			ctrl.UpdateProfilePhoto(c)
		})

		payload := UpdateProfilePhotoRequest{
			PhotoURL: "https://cdn.bikko.com.br/profiles/user_123.jpg",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/profile/photo", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
		if savedUserID != "user_123" || savedURL != "https://cdn.bikko.com.br/profiles/user_123.jpg" {
			t.Errorf("unexpected saved values: userID=%s, photoURL=%s", savedUserID, savedURL)
		}
	})
}
