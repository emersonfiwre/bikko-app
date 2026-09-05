package usecase

import (
	"context"

	"bikko-app/internal/domain"
)

type SearchUseCase struct {
	categoryRepo domain.CategoryRepository
	serviceRepo  domain.ServiceRepository
}

func NewSearchUseCase(
	categoryRepo domain.CategoryRepository,
	serviceRepo domain.ServiceRepository,
) *SearchUseCase {
	return &SearchUseCase{
		categoryRepo: categoryRepo,
		serviceRepo:  serviceRepo,
	}
}

func (uc *SearchUseCase) Search(ctx context.Context, query string) (*domain.SearchResponse, error) {
	if query == "" {
		return &domain.SearchResponse{
			Services:   []domain.Service{},
			Categories: []domain.Category{},
		}, nil
	}

	services, err := uc.serviceRepo.SearchServices(ctx, query)
	if err != nil {
		services = []domain.Service{}
	}

	categories, err := uc.categoryRepo.SearchCategories(ctx, query)
	if err != nil {
		categories = []domain.Category{}
	}

	return &domain.SearchResponse{
		Services:   services,
		Categories: categories,
	}, nil
}
