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
			Items: []domain.SearchItem{},
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

	items := make([]domain.SearchItem, 0, len(categories)+len(services))

	for _, c := range categories {
		items = append(items, domain.SearchItem{
			ID:          c.ID,
			Name:        c.Name,
			Type:        "category",
			ImageURL:    c.IconURL,
			Description: "Categoria de Serviço",
		})
	}

	for _, s := range services {
		items = append(items, domain.SearchItem{
			ID:          s.ID,
			Name:        s.Name,
			Type:        "service",
			ImageURL:    s.ThumbnailURL,
			Description: s.Description,
		})
	}

	return &domain.SearchResponse{
		Items: items,
	}, nil
}
