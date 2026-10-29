package middleware

import (
	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/security"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
)

type identityStatusRepo struct {
	domain.UserRepository
	user *domain.User
	err  error
}

func (r identityStatusRepo) GetByID(context.Context, string) (*domain.User, error) {
	return r.user, r.err
}
func TestAuthMiddlewareRevokesDeletedAccountAndRejectsMockBearers(t *testing.T) {
	jwt := security.NewJWTService("test-secret")
	token, err := jwt.GenerateToken("lucas")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, token string
		repo        identityStatusRepo
		status      int
	}{{"deleted", token, identityStatusRepo{err: domain.ErrNotFound}, 401}, {"outage", token, identityStatusRepo{err: errors.New("db down")}, 503}, {"mock token", "mock_token_123", identityStatusRepo{user: &domain.User{ID: "lucas"}}, 401}, {"active", token, identityStatusRepo{user: &domain.User{ID: "lucas"}}, 200}} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(AuthMiddleware(jwt, tc.repo))
			r.GET("/protected", func(c *gin.Context) {
				if GetUserID(c) != "lucas" {
					t.Fatal("wrong actor")
				}
				c.Status(200)
			})
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("got %d want %d", w.Code, tc.status)
			}
		})
	}
}
func TestProviderRoleIsSeparateFromAvailability(t *testing.T) {
	jwt := security.NewJWTService("test-secret")
	token, _ := jwt.GenerateToken("rafael")
	for _, tc := range []struct {
		name               string
		user               *domain.User
		status             int
		availableInHandler bool
	}{{"client", &domain.User{ID: "rafael"}, 403, false}, {"offline provider", &domain.User{ID: "rafael", IsBikker: true, Bikker: &domain.Bikker{IsActive: false}}, 200, false}, {"available provider", &domain.User{ID: "rafael", IsBikker: true, Bikker: &domain.Bikker{IsActive: true}}, 200, true}} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(AuthMiddleware(jwt, identityStatusRepo{user: tc.user}), RequireProvider())
			r.GET("/provider", func(c *gin.Context) {
				if c.GetBool("isProviderAvailable") != tc.availableInHandler {
					t.Errorf("availability=%v, want %v", c.GetBool("isProviderAvailable"), tc.availableInHandler)
				}
				c.Status(200)
			})
			req := httptest.NewRequest("GET", "/provider", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatal(w.Code)
			}
		})
	}
}
func TestGetUserIDDoesNotPanicOnMalformedContext(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("userID", 42)
	if GetUserID(c) != "" {
		t.Fatal("accepted malformed actor")
	}
}
