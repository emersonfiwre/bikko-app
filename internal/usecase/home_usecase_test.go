package usecase

import (
	"context"
	"errors"
	"testing"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/cache"
)

type fakeCategoryRepo struct {
	categories       []domain.Category
	categoryByID     map[string]*domain.Category
	getAllActiveErr  error
	getByIDErr       error
	searchCategories []domain.Category
	searchErr        error
}

func (f *fakeCategoryRepo) GetAllActive(ctx context.Context) ([]domain.Category, error) {
	if f.getAllActiveErr != nil {
		return nil, f.getAllActiveErr
	}
	return f.categories, nil
}

func (f *fakeCategoryRepo) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	if cat, ok := f.categoryByID[id]; ok {
		return cat, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeCategoryRepo) SearchCategories(ctx context.Context, query string) ([]domain.Category, error) {
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	return f.searchCategories, nil
}

type fakeServiceRepo struct {
	serviceByID          map[string]*domain.Service
	servicesNear         []domain.Service
	adviceServices       []domain.Service
	servicesByBikker     map[string][]domain.Service
	servicesByCategory   map[string][]domain.Service
	searchServices       []domain.Service
	getByIDError         error
	getFeaturedError     error
	getByCategoryError   error
	getByBikkerError     error
	searchServicesError  error
	lastAdviceLimit      int
}

func (f *fakeServiceRepo) GetByID(ctx context.Context, id string) (*domain.Service, error) {
	if f.getByIDError != nil {
		return nil, f.getByIDError
	}
	if s, ok := f.serviceByID[id]; ok {
		return s, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeServiceRepo) GetServiceByID(ctx context.Context, id string) (*domain.Service, error) {
	return f.GetByID(ctx, id)
}

func (f *fakeServiceRepo) GetServicesNear(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]domain.Service, error) {
	return f.servicesNear, nil
}

func (f *fakeServiceRepo) GetFeaturedAdviceServices(ctx context.Context, limit int) ([]domain.Service, error) {
	f.lastAdviceLimit = limit
	if f.getFeaturedError != nil {
		return nil, f.getFeaturedError
	}
	return f.adviceServices, nil
}

func (f *fakeServiceRepo) GetActiveServicesByBikkerID(ctx context.Context, bikkerID string) ([]domain.Service, error) {
	if f.getByBikkerError != nil {
		return nil, f.getByBikkerError
	}
	return f.servicesByBikker[bikkerID], nil
}

func (f *fakeServiceRepo) GetServicesByCategoryID(ctx context.Context, categoryID string) ([]domain.Service, error) {
	if f.getByCategoryError != nil {
		return nil, f.getByCategoryError
	}
	return f.servicesByCategory[categoryID], nil
}

func (f *fakeServiceRepo) SearchServices(ctx context.Context, query string) ([]domain.Service, error) {
	if f.searchServicesError != nil {
		return nil, f.searchServicesError
	}
	return f.searchServices, nil
}

type fakeBikkerRepo struct {
	nearBikkers    []domain.NearBikkerItem
	nearBikkersErr error
	bikkerByID     map[string]*domain.BikkerProfileResponse
	allBikkers     []domain.BikkerProfileResponse
	getIDErr       error
	getAllErr      error
}

func (f *fakeBikkerRepo) GetByID(ctx context.Context, id string) (*domain.BikkerProfileResponse, error) {
	if f.getIDErr != nil {
		return nil, f.getIDErr
	}
	if b, ok := f.bikkerByID[id]; ok {
		return b, nil
	}
	return nil, errors.New("bikker not found")
}

func (f *fakeBikkerRepo) GetNearBikkers(ctx context.Context, lat, lon float64, limit int) ([]domain.NearBikkerItem, error) {
	if f.nearBikkersErr != nil {
		return nil, f.nearBikkersErr
	}
	return f.nearBikkers, nil
}

func (f *fakeBikkerRepo) GetAll(ctx context.Context, limit int) ([]domain.BikkerProfileResponse, error) {
	if f.getAllErr != nil {
		return nil, f.getAllErr
	}
	return f.allBikkers, nil
}

func TestHomeUseCase_GetHomeFeed(t *testing.T) {
	ctx := context.Background()

	t.Run("success without location returns categories and advice only", func(t *testing.T) {
		catRepo := &fakeCategoryRepo{
			categories: []domain.Category{
				{ID: "cat_1", Name: "Elétrica"},
			},
		}
		srvRepo := &fakeServiceRepo{
			adviceServices: []domain.Service{
				{ID: "srv_1", Name: "Instalação Elétrica"},
			},
		}
		bkrRepo := &fakeBikkerRepo{
			nearBikkers: []domain.NearBikkerItem{
				{ID: "bikker_1", Title: "João Elétrica"},
			},
		}

		redisSvc := &cache.RedisService{}
		uc := NewHomeUseCase(catRepo, srvRepo, bkrRepo, redisSvc)

		resp, err := uc.GetHomeFeed(ctx, 0, 0, false)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp == nil {
			t.Fatal("expected home response, got nil")
		}

		// Should have categories_collection and advice_collection, but NOT near_collection
		if len(resp.Collections) != 2 {
			t.Fatalf("expected 2 collections without location, got %d", len(resp.Collections))
		}
		if resp.Collections[0].ID != "categories_collection" || resp.Collections[0].Type != "Category" {
			t.Errorf("expected first collection to be categories_collection, got %+v", resp.Collections[0])
		}
		if resp.Collections[1].ID != "advice_collection" || resp.Collections[1].Type != "Advice" {
			t.Errorf("expected second collection to be advice_collection, got %+v", resp.Collections[1])
		}
	})

	t.Run("success with location includes near bikkers collection", func(t *testing.T) {
		catRepo := &fakeCategoryRepo{
			categories: []domain.Category{
				{ID: "cat_1", Name: "Hidráulica"},
			},
		}
		srvRepo := &fakeServiceRepo{
			adviceServices: []domain.Service{
				{ID: "srv_1", Name: "Desentupimento"},
			},
		}
		bkrRepo := &fakeBikkerRepo{
			nearBikkers: []domain.NearBikkerItem{
				{ID: "bikker_1", Title: "Carlos Encanador", Distance: "1.2 km"},
			},
		}

		redisSvc := &cache.RedisService{}
		uc := NewHomeUseCase(catRepo, srvRepo, bkrRepo, redisSvc)

		resp, err := uc.GetHomeFeed(ctx, -23.5505, -46.6333, true)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(resp.Collections) != 3 {
			t.Fatalf("expected 3 collections with location, got %d", len(resp.Collections))
		}
		if resp.Collections[2].ID != "near_collection" || resp.Collections[2].Type != "ProfileNear" {
			t.Errorf("expected third collection to be near_collection, got %+v", resp.Collections[2])
		}
	})

	t.Run("gracefully handles near bikkers error and still returns categories and advice", func(t *testing.T) {
		catRepo := &fakeCategoryRepo{
			categories: []domain.Category{{ID: "cat_1", Name: "Pintura"}},
		}
		srvRepo := &fakeServiceRepo{
			adviceServices: []domain.Service{{ID: "srv_1", Name: "Pintura Residencial"}},
		}
		bkrRepo := &fakeBikkerRepo{
			nearBikkersErr: errors.New("postgis query failed"),
		}

		redisSvc := &cache.RedisService{}
		uc := NewHomeUseCase(catRepo, srvRepo, bkrRepo, redisSvc)

		resp, err := uc.GetHomeFeed(ctx, -23.5505, -46.6333, true)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(resp.Collections) != 2 {
			t.Fatalf("expected 2 collections when near bikkers fails, got %d", len(resp.Collections))
		}
	})

	t.Run("returns error when category repository fails", func(t *testing.T) {
		catRepo := &fakeCategoryRepo{
			getAllActiveErr: errors.New("db categories error"),
		}
		srvRepo := &fakeServiceRepo{}
		bkrRepo := &fakeBikkerRepo{}

		redisSvc := &cache.RedisService{}
		uc := NewHomeUseCase(catRepo, srvRepo, bkrRepo, redisSvc)

		resp, err := uc.GetHomeFeed(ctx, 0, 0, false)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if resp != nil {
			t.Fatalf("expected nil response on error, got %+v", resp)
		}
	})

	t.Run("returns error when service repository advice fails", func(t *testing.T) {
		catRepo := &fakeCategoryRepo{
			categories: []domain.Category{{ID: "cat_1", Name: "Montagem"}},
		}
		srvRepo := &fakeServiceRepo{
			getFeaturedError: errors.New("db advice error"),
		}
		bkrRepo := &fakeBikkerRepo{}

		redisSvc := &cache.RedisService{}
		uc := NewHomeUseCase(catRepo, srvRepo, bkrRepo, redisSvc)

		resp, err := uc.GetHomeFeed(ctx, 0, 0, false)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if resp != nil {
			t.Fatalf("expected nil response on error, got %+v", resp)
		}
	})
}

func TestHomeUseCase_GetCategories(t *testing.T) {
	ctx := context.Background()

	t.Run("success returns categories", func(t *testing.T) {
		catRepo := &fakeCategoryRepo{
			categories: []domain.Category{
				{ID: "cat_1", Name: "Jardinagem"},
				{ID: "cat_2", Name: "Marcenaria"},
			},
		}
		uc := NewHomeUseCase(catRepo, &fakeServiceRepo{}, &fakeBikkerRepo{}, &cache.RedisService{})

		cats, err := uc.GetCategories(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(cats) != 2 {
			t.Errorf("expected 2 categories, got %d", len(cats))
		}
	})

	t.Run("error returns error", func(t *testing.T) {
		catRepo := &fakeCategoryRepo{
			getAllActiveErr: errors.New("cannot fetch"),
		}
		uc := NewHomeUseCase(catRepo, &fakeServiceRepo{}, &fakeBikkerRepo{}, &cache.RedisService{})

		_, err := uc.GetCategories(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestHomeUseCase_GetCategoryByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success attaches services to category", func(t *testing.T) {
		catRepo := &fakeCategoryRepo{
			categoryByID: map[string]*domain.Category{
				"cat_1": {ID: "cat_1", Name: "Climatização"},
			},
		}
		srvRepo := &fakeServiceRepo{
			servicesByCategory: map[string][]domain.Service{
				"cat_1": {
					{ID: "srv_1", Name: "Ar Condicionado Split"},
				},
			},
		}

		uc := NewHomeUseCase(catRepo, srvRepo, &fakeBikkerRepo{}, &cache.RedisService{})

		cat, err := uc.GetCategoryByID(ctx, "cat_1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cat == nil || cat.ID != "cat_1" {
			t.Fatalf("unexpected category: %+v", cat)
		}
		if len(cat.Services) != 1 || cat.Services[0].ID != "srv_1" {
			t.Errorf("expected services to be attached, got %+v", cat.Services)
		}
	})

	t.Run("category not found returns error", func(t *testing.T) {
		catRepo := &fakeCategoryRepo{
			getByIDErr: domain.ErrNotFound,
		}
		uc := NewHomeUseCase(catRepo, &fakeServiceRepo{}, &fakeBikkerRepo{}, &cache.RedisService{})

		_, err := uc.GetCategoryByID(ctx, "non_existent")
		if err != domain.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestHomeUseCase_GetServiceQueries(t *testing.T) {
	ctx := context.Background()

	srvRepo := &fakeServiceRepo{
		serviceByID: map[string]*domain.Service{
			"srv_100": {ID: "srv_100", Name: "Limpeza Pós Obra"},
		},
		servicesByBikker: map[string][]domain.Service{
			"bikker_50": {
				{ID: "srv_100", Name: "Limpeza Pós Obra"},
			},
		},
		servicesByCategory: map[string][]domain.Service{
			"cat_10": {
				{ID: "srv_100", Name: "Limpeza Pós Obra"},
			},
		},
		adviceServices: []domain.Service{
			{ID: "srv_100", Name: "Limpeza Pós Obra"},
		},
	}

	uc := NewHomeUseCase(&fakeCategoryRepo{}, srvRepo, &fakeBikkerRepo{}, &cache.RedisService{})

	t.Run("GetServiceByID", func(t *testing.T) {
		s, err := uc.GetServiceByID(ctx, "srv_100")
		if err != nil || s.ID != "srv_100" {
			t.Errorf("GetServiceByID failed: %v", err)
		}
	})

	t.Run("GetServicesByBikkerID", func(t *testing.T) {
		services, err := uc.GetServicesByBikkerID(ctx, "bikker_50")
		if err != nil || len(services) != 1 {
			t.Errorf("GetServicesByBikkerID failed: %v", err)
		}
	})

	t.Run("GetServicesByCategoryID", func(t *testing.T) {
		services, err := uc.GetServicesByCategoryID(ctx, "cat_10")
		if err != nil || len(services) != 1 {
			t.Errorf("GetServicesByCategoryID failed: %v", err)
		}
	})

	t.Run("GetFeaturedAdviceServices", func(t *testing.T) {
		services, err := uc.GetFeaturedAdviceServices(ctx, 15)
		if err != nil || len(services) != 1 {
			t.Errorf("GetFeaturedAdviceServices failed: %v", err)
		}
		if srvRepo.lastAdviceLimit != 15 {
			t.Errorf("expected limit 15, got %d", srvRepo.lastAdviceLimit)
		}
	})
}
