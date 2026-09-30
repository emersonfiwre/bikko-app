package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bikko-app/internal/domain"
	"bikko-app/internal/usecase"

	"github.com/gin-gonic/gin"
)

type searchCategoryRepoMock struct {
	categories []domain.Category
}

func (s *searchCategoryRepoMock) GetAllActive(ctx context.Context) ([]domain.Category, error) {
	return nil, nil
}

func (s *searchCategoryRepoMock) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	return nil, nil
}

func (s *searchCategoryRepoMock) SearchCategories(ctx context.Context, query string) ([]domain.Category, error) {
	return s.categories, nil
}

type searchServiceRepoMock struct {
	services []domain.Service
}

func (s *searchServiceRepoMock) GetByID(ctx context.Context, id string) (*domain.Service, error) {
	return nil, nil
}

func (s *searchServiceRepoMock) GetServiceByID(ctx context.Context, id string) (*domain.Service, error) {
	return nil, nil
}

func (s *searchServiceRepoMock) GetServicesNear(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]domain.Service, error) {
	return nil, nil
}

func (s *searchServiceRepoMock) GetFeaturedAdviceServices(ctx context.Context, limit int) ([]domain.Service, error) {
	return nil, nil
}

func (s *searchServiceRepoMock) GetActiveServicesByBikkerID(ctx context.Context, bikkerID string) ([]domain.Service, error) {
	return nil, nil
}

func (s *searchServiceRepoMock) GetServicesByCategoryID(ctx context.Context, categoryID string) ([]domain.Service, error) {
	return nil, nil
}

func (s *searchServiceRepoMock) SearchServices(ctx context.Context, query string) ([]domain.Service, error) {
	return s.services, nil
}

func TestSearchController_Search(t *testing.T) {
	catRepo := &searchCategoryRepoMock{
		categories: []domain.Category{
			{ID: "cat_1", Name: "Montagem", IconURL: "https://example.com/icon.png"},
		},
	}
	srvRepo := &searchServiceRepoMock{
		services: []domain.Service{
			{ID: "srv_1", Name: "Montador de Móveis", Description: "Montagem de armários e camas"},
		},
	}

	searchUC := usecase.NewSearchUseCase(catRepo, srvRepo)
	ctrl := NewSearchController(searchUC)

	router := gin.New()
	router.GET("/search", ctrl.Search)

	t.Run("empty query string returns empty result list", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/search?q=", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp domain.SearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp.Items) != 0 {
			t.Errorf("expected 0 items for empty query, got %d", len(resp.Items))
		}
	})

	t.Run("whitespace query string returns empty result list", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/search?q=%20%20%20", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp domain.SearchResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Items) != 0 {
			t.Errorf("expected 0 items for whitespace query, got %d", len(resp.Items))
		}
	})

	t.Run("valid query returns aggregated search items", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/search?q=montador", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp domain.SearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(resp.Items))
		}
		if resp.Items[0].Type != "category" || resp.Items[1].Type != "service" {
			t.Errorf("unexpected item types: %+v", resp.Items)
		}
	})
}
