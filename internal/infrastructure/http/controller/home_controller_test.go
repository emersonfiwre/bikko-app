package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/cache"
	"bikko-app/internal/usecase"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeCategoryRepository struct {
	categories   []domain.Category
	categoryByID map[string]*domain.Category
	err          error
}

func (f *fakeCategoryRepository) GetAllActive(ctx context.Context) ([]domain.Category, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.categories, nil
}

func (f *fakeCategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	if f.err != nil {
		return nil, f.err
	}
	if cat, ok := f.categoryByID[id]; ok {
		return cat, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeCategoryRepository) SearchCategories(ctx context.Context, query string) ([]domain.Category, error) {
	return nil, nil
}

type fakeServiceRepository struct {
	servicesByID       map[string]*domain.Service
	servicesByCategory map[string][]domain.Service
	servicesByBikker   map[string][]domain.Service
	adviceServices     []domain.Service
	err                error
}

func (f *fakeServiceRepository) GetByID(ctx context.Context, id string) (*domain.Service, error) {
	if f.err != nil {
		return nil, f.err
	}
	if s, ok := f.servicesByID[id]; ok {
		return s, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeServiceRepository) GetServiceByID(ctx context.Context, id string) (*domain.Service, error) {
	return f.GetByID(ctx, id)
}

func (f *fakeServiceRepository) GetServicesNear(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]domain.Service, error) {
	return nil, nil
}

func (f *fakeServiceRepository) GetFeaturedAdviceServices(ctx context.Context, limit int) ([]domain.Service, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.adviceServices, nil
}

func (f *fakeServiceRepository) GetActiveServicesByBikkerID(ctx context.Context, bikkerID string) ([]domain.Service, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.servicesByBikker[bikkerID], nil
}

func (f *fakeServiceRepository) GetServicesByCategoryID(ctx context.Context, categoryID string) ([]domain.Service, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.servicesByCategory[categoryID], nil
}

func (f *fakeServiceRepository) SearchServices(ctx context.Context, query string) ([]domain.Service, error) {
	return nil, nil
}

type fakeBikkerRepository struct {
	nearBikkers []domain.NearBikkerItem
	err         error
}

func (f *fakeBikkerRepository) GetByID(ctx context.Context, id string) (*domain.BikkerProfileResponse, error) {
	return nil, nil
}

func (f *fakeBikkerRepository) GetNearBikkers(ctx context.Context, lat, lon float64, limit int) ([]domain.NearBikkerItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.nearBikkers, nil
}

func (f *fakeBikkerRepository) GetAll(ctx context.Context, limit int) ([]domain.BikkerProfileResponse, error) {
	return nil, nil
}

func setupHomeTestRouter(catRepo domain.CategoryRepository, srvRepo domain.ServiceRepository, bkrRepo domain.BikkerRepository) *gin.Engine {
	uc := usecase.NewHomeUseCase(catRepo, srvRepo, bkrRepo, &cache.RedisService{})
	ctrl := NewHomeController(uc)

	r := gin.New()
	r.GET("/home", ctrl.GetHome)
	r.GET("/categories", ctrl.GetCategories)
	r.GET("/categories/:id", ctrl.GetCategoryByID)
	r.GET("/services", ctrl.GetServices)
	r.GET("/services/:id", ctrl.GetServiceByID)
	return r
}

func TestHomeController_GetHome(t *testing.T) {
	t.Run("success without location coordinates", func(t *testing.T) {
		catRepo := &fakeCategoryRepository{
			categories: []domain.Category{{ID: "c1", Name: "Pintura"}},
		}
		srvRepo := &fakeServiceRepository{
			adviceServices: []domain.Service{{ID: "s1", Name: "Pintor Profissional"}},
		}
		bkrRepo := &fakeBikkerRepository{}

		router := setupHomeTestRouter(catRepo, srvRepo, bkrRepo)

		req, _ := http.NewRequest(http.MethodGet, "/home", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp domain.HomeResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if len(resp.Collections) != 2 {
			t.Errorf("expected 2 collections, got %d", len(resp.Collections))
		}
	})

	t.Run("success with valid lat and lon coordinates", func(t *testing.T) {
		catRepo := &fakeCategoryRepository{
			categories: []domain.Category{{ID: "c1", Name: "Hidráulica"}},
		}
		srvRepo := &fakeServiceRepository{
			adviceServices: []domain.Service{{ID: "s1", Name: "Desentupimento"}},
		}
		bkrRepo := &fakeBikkerRepository{
			nearBikkers: []domain.NearBikkerItem{{ID: "b1", Title: "Encanador João"}},
		}

		router := setupHomeTestRouter(catRepo, srvRepo, bkrRepo)

		req, _ := http.NewRequest(http.MethodGet, "/home?lat=-23.5505&lon=-46.6333", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp domain.HomeResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Collections) != 3 {
			t.Errorf("expected 3 collections with location, got %d", len(resp.Collections))
		}
	})

	t.Run("returns 500 when repository fails", func(t *testing.T) {
		catRepo := &fakeCategoryRepository{
			err: errors.New("db failure"),
		}
		srvRepo := &fakeServiceRepository{}
		bkrRepo := &fakeBikkerRepository{}

		router := setupHomeTestRouter(catRepo, srvRepo, bkrRepo)

		req, _ := http.NewRequest(http.MethodGet, "/home", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestHomeController_GetCategories(t *testing.T) {
	t.Run("success returns categories list", func(t *testing.T) {
		catRepo := &fakeCategoryRepository{
			categories: []domain.Category{
				{ID: "c1", Name: "Reformas"},
				{ID: "c2", Name: "Limpeza"},
			},
		}
		router := setupHomeTestRouter(catRepo, &fakeServiceRepository{}, &fakeBikkerRepository{})

		req, _ := http.NewRequest(http.MethodGet, "/categories", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var cats []domain.Category
		_ = json.Unmarshal(w.Body.Bytes(), &cats)
		if len(cats) != 2 {
			t.Errorf("expected 2 categories, got %d", len(cats))
		}
	})

	t.Run("returns 500 on failure", func(t *testing.T) {
		catRepo := &fakeCategoryRepository{
			err: errors.New("cannot fetch"),
		}
		router := setupHomeTestRouter(catRepo, &fakeServiceRepository{}, &fakeBikkerRepository{})

		req, _ := http.NewRequest(http.MethodGet, "/categories", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestHomeController_GetCategoryByID(t *testing.T) {
	t.Run("success returns category", func(t *testing.T) {
		catRepo := &fakeCategoryRepository{
			categoryByID: map[string]*domain.Category{
				"c1": {ID: "c1", Name: "Eletricidade"},
			},
		}
		router := setupHomeTestRouter(catRepo, &fakeServiceRepository{}, &fakeBikkerRepository{})

		req, _ := http.NewRequest(http.MethodGet, "/categories/c1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		catRepo := &fakeCategoryRepository{
			categoryByID: map[string]*domain.Category{},
		}
		router := setupHomeTestRouter(catRepo, &fakeServiceRepository{}, &fakeBikkerRepository{})

		req, _ := http.NewRequest(http.MethodGet, "/categories/unknown", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestHomeController_GetServiceByID(t *testing.T) {
	t.Run("success returns service", func(t *testing.T) {
		srvRepo := &fakeServiceRepository{
			servicesByID: map[string]*domain.Service{
				"s1": {ID: "s1", Name: "Troca de Chuveiro"},
			},
		}
		router := setupHomeTestRouter(&fakeCategoryRepository{}, srvRepo, &fakeBikkerRepository{})

		req, _ := http.NewRequest(http.MethodGet, "/services/s1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 404 when service not found", func(t *testing.T) {
		srvRepo := &fakeServiceRepository{
			servicesByID: map[string]*domain.Service{},
		}
		router := setupHomeTestRouter(&fakeCategoryRepository{}, srvRepo, &fakeBikkerRepository{})

		req, _ := http.NewRequest(http.MethodGet, "/services/unknown", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestHomeController_GetServices(t *testing.T) {
	t.Run("filter by categoryId", func(t *testing.T) {
		srvRepo := &fakeServiceRepository{
			servicesByCategory: map[string][]domain.Service{
				"cat_elec": {{ID: "s1", Name: "Serviço Elétrico"}},
			},
		}
		router := setupHomeTestRouter(&fakeCategoryRepository{}, srvRepo, &fakeBikkerRepository{})

		req, _ := http.NewRequest(http.MethodGet, "/services?categoryId=cat_elec", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var services []domain.Service
		_ = json.Unmarshal(w.Body.Bytes(), &services)
		if len(services) != 1 || services[0].ID != "s1" {
			t.Errorf("expected 1 service with ID s1, got %+v", services)
		}
	})

	t.Run("filter by bikkerId", func(t *testing.T) {
		srvRepo := &fakeServiceRepository{
			servicesByBikker: map[string][]domain.Service{
				"bikker_99": {{ID: "s2", Name: "Serviço do Bikker 99"}},
			},
		}
		router := setupHomeTestRouter(&fakeCategoryRepository{}, srvRepo, &fakeBikkerRepository{})

		req, _ := http.NewRequest(http.MethodGet, "/services?bikkerId=bikker_99", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var services []domain.Service
		_ = json.Unmarshal(w.Body.Bytes(), &services)
		if len(services) != 1 || services[0].ID != "s2" {
			t.Errorf("expected 1 service with ID s2, got %+v", services)
		}
	})

	t.Run("fallback to featured advice services without query params", func(t *testing.T) {
		srvRepo := &fakeServiceRepository{
			adviceServices: []domain.Service{
				{ID: "s3", Name: "Serviço em Destaque"},
			},
		}
		router := setupHomeTestRouter(&fakeCategoryRepository{}, srvRepo, &fakeBikkerRepository{})

		req, _ := http.NewRequest(http.MethodGet, "/services", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var services []domain.Service
		_ = json.Unmarshal(w.Body.Bytes(), &services)
		if len(services) != 1 || services[0].ID != "s3" {
			t.Errorf("expected featured advice services, got %+v", services)
		}
	})
}
