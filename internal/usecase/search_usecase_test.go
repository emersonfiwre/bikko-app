package usecase

import (
	"context"
	"errors"
	"testing"

	"bikko-app/internal/domain"
)

type searchTestCategoryRepo struct {
	categories []domain.Category
	err        error
}

func (s *searchTestCategoryRepo) GetAllActive(ctx context.Context) ([]domain.Category, error) {
	return nil, nil
}

func (s *searchTestCategoryRepo) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	return nil, nil
}

func (s *searchTestCategoryRepo) SearchCategories(ctx context.Context, query string) ([]domain.Category, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.categories, nil
}

type searchTestServiceRepo struct {
	services []domain.Service
	err      error
}

func (s *searchTestServiceRepo) GetByID(ctx context.Context, id string) (*domain.Service, error) {
	return nil, nil
}

func (s *searchTestServiceRepo) GetServiceByID(ctx context.Context, id string) (*domain.Service, error) {
	return nil, nil
}

func (s *searchTestServiceRepo) GetServicesNear(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]domain.Service, error) {
	return nil, nil
}

func (s *searchTestServiceRepo) GetFeaturedAdviceServices(ctx context.Context, limit int) ([]domain.Service, error) {
	return nil, nil
}

func (s *searchTestServiceRepo) GetActiveServicesByBikkerID(ctx context.Context, bikkerID string) ([]domain.Service, error) {
	return nil, nil
}

func (s *searchTestServiceRepo) GetServicesByCategoryID(ctx context.Context, categoryID string) ([]domain.Service, error) {
	return nil, nil
}

func (s *searchTestServiceRepo) SearchServices(ctx context.Context, query string) ([]domain.Service, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.services, nil
}

func TestSearchUseCase_Search(t *testing.T) {
	ctx := context.Background()

	t.Run("empty query returns empty list without calling repositories", func(t *testing.T) {
		catRepo := &searchTestCategoryRepo{
			err: errors.New("should not be called"),
		}
		srvRepo := &searchTestServiceRepo{
			err: errors.New("should not be called"),
		}

		uc := NewSearchUseCase(catRepo, srvRepo)
		resp, err := uc.Search(ctx, "")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
		if len(resp.Items) != 0 {
			t.Errorf("expected 0 items, got %d", len(resp.Items))
		}
	})

	t.Run("returns aggregated categories and services correctly mapped", func(t *testing.T) {
		catRepo := &searchTestCategoryRepo{
			categories: []domain.Category{
				{
					ID:      "cat_paint",
					Name:    "Pintura",
					IconURL: "https://example.com/paint.png",
				},
			},
		}
		srvRepo := &searchTestServiceRepo{
			services: []domain.Service{
				{
					ID:           "srv_paint_1",
					Name:         "Pintor de Parede",
					ThumbnailURL: "https://example.com/painter.jpg",
					Description:  "Pintura interna e externa",
				},
			},
		}

		uc := NewSearchUseCase(catRepo, srvRepo)
		resp, err := uc.Search(ctx, "pint")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(resp.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(resp.Items))
		}

		// First category item
		cItem := resp.Items[0]
		if cItem.ID != "cat_paint" || cItem.Name != "Pintura" || cItem.Type != "category" {
			t.Errorf("unexpected category item: %+v", cItem)
		}
		if cItem.ImageURL != "https://example.com/paint.png" || cItem.Description != "Categoria de Serviço" {
			t.Errorf("unexpected category metadata: %+v", cItem)
		}

		// Second service item
		sItem := resp.Items[1]
		if sItem.ID != "srv_paint_1" || sItem.Name != "Pintor de Parede" || sItem.Type != "service" {
			t.Errorf("unexpected service item: %+v", sItem)
		}
		if sItem.ImageURL != "https://example.com/painter.jpg" || sItem.Description != "Pintura interna e externa" {
			t.Errorf("unexpected service metadata: %+v", sItem)
		}
	})

	t.Run("gracefully degrades when service repo errors", func(t *testing.T) {
		catRepo := &searchTestCategoryRepo{
			categories: []domain.Category{
				{ID: "cat_1", Name: "Hidráulica", IconURL: "https://example.com/h.png"},
			},
		}
		srvRepo := &searchTestServiceRepo{
			err: errors.New("service search db timeout"),
		}

		uc := NewSearchUseCase(catRepo, srvRepo)
		resp, err := uc.Search(ctx, "hidro")

		if err != nil {
			t.Fatalf("expected graceful success, got error: %v", err)
		}
		if len(resp.Items) != 1 || resp.Items[0].ID != "cat_1" {
			t.Errorf("expected 1 category item, got %+v", resp.Items)
		}
	})

	t.Run("gracefully degrades when category repo errors", func(t *testing.T) {
		catRepo := &searchTestCategoryRepo{
			err: errors.New("category search db timeout"),
		}
		srvRepo := &searchTestServiceRepo{
			services: []domain.Service{
				{ID: "srv_1", Name: "Eletricista Predial", Description: "Troca de rede"},
			},
		}

		uc := NewSearchUseCase(catRepo, srvRepo)
		resp, err := uc.Search(ctx, "eletro")

		if err != nil {
			t.Fatalf("expected graceful success, got error: %v", err)
		}
		if len(resp.Items) != 1 || resp.Items[0].ID != "srv_1" {
			t.Errorf("expected 1 service item, got %+v", resp.Items)
		}
	})

	t.Run("gracefully handles both repositories erroring", func(t *testing.T) {
		catRepo := &searchTestCategoryRepo{err: errors.New("cat error")}
		srvRepo := &searchTestServiceRepo{err: errors.New("srv error")}

		uc := NewSearchUseCase(catRepo, srvRepo)
		resp, err := uc.Search(ctx, "anything")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(resp.Items) != 0 {
			t.Errorf("expected 0 items, got %d", len(resp.Items))
		}
	})
}
