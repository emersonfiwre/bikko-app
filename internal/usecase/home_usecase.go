package usecase

import (
	"context"
	"log"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/cache"
)

type HomeUseCase struct {
	categoryRepo domain.CategoryRepository
	serviceRepo  domain.ServiceRepository
	bikkerRepo   domain.BikkerRepository
	redisService *cache.RedisService
}

func NewHomeUseCase(
	categoryRepo domain.CategoryRepository,
	serviceRepo domain.ServiceRepository,
	bikkerRepo domain.BikkerRepository,
	redisService *cache.RedisService,
) *HomeUseCase {
	return &HomeUseCase{
		categoryRepo: categoryRepo,
		serviceRepo:  serviceRepo,
		bikkerRepo:   bikkerRepo,
		redisService: redisService,
	}
}

func (uc *HomeUseCase) GetHomeFeed(ctx context.Context, lat, lon float64, hasLocation bool) (*domain.HomeResponse, error) {
	// 1. Categories (Redis Cache with 24h TTL)
	categories, err := uc.redisService.GetCategories(ctx)
	if err != nil {
		log.Println("[HomeUseCase] Cache Miss: home:categories -> buscando no PostgreSQL")
		categories, err = uc.categoryRepo.GetAllActive(ctx)
		if err != nil {
			return nil, err
		}
		_ = uc.redisService.SetCategories(ctx, categories)
	}

	// 2. Near Bikkers: ONLY when location permission was granted and coordinates provided!
	var nearBikkers []domain.NearBikkerItem
	if hasLocation {
		var err error
		nearBikkers, err = uc.redisService.GetNearBikkers(ctx, lat, lon)
		if err != nil {
			log.Println("[HomeUseCase] Cache Miss: home:near_bikkers:{geohash} -> executando query espacial PostGIS")
			nearBikkers, err = uc.bikkerRepo.GetNearBikkers(ctx, lat, lon, 10)
			if err != nil {
				log.Printf("[HomeUseCase] Erro ao buscar bikkers por perto: %v\n", err)
			} else if len(nearBikkers) > 0 {
				_ = uc.redisService.SetNearBikkers(ctx, lat, lon, nearBikkers)
			}
		}
	}

	// 3. Advice Services (Redis Cache with 1h TTL)
	adviceServices, err := uc.redisService.GetAdviceServices(ctx)
	if err != nil {
		log.Println("[HomeUseCase] Cache Miss: home:advice -> buscando serviços em destaque no PostgreSQL")
		adviceServices, err = uc.serviceRepo.GetFeaturedAdviceServices(ctx, 10)
		if err != nil {
			return nil, err
		}
		if len(adviceServices) > 0 {
			_ = uc.redisService.SetAdviceServices(ctx, adviceServices)
		}
	}

	collections := make([]domain.HomeCollection, 0)

	if len(categories) > 0 {
		var items []domain.HomeItem
		for _, c := range categories {
			items = append(items, domain.HomeItem{
				ID:       c.ID,
				Name:     c.Name,
				Title:    c.Name,
				IconURL:  c.IconURL,
				Subtitle: c.Description,
			})
		}
		collections = append(collections, domain.HomeCollection{
			ID:    "categories_collection",
			Name:  "Categorias",
			Type:  "Category",
			Items: items,
		})
	}

	if len(adviceServices) > 0 {
		var items []domain.HomeItem
		for _, s := range adviceServices {
			items = append(items, domain.HomeItem{
				ID:           s.ID,
				Title:        s.Name,
				Subtitle:     s.Description,
				ImageURL:     s.ThumbnailURL,
				Rating:       s.ReviewsAverage,
				ProviderName: s.ProviderName,
				CategoryID:   s.CategoryID,
				BikkerID:     s.BikkerID,
				Latitude:     s.Latitude,
				Longitude:    s.Longitude,
			})
		}
		collections = append(collections, domain.HomeCollection{
			ID:    "advice_collection",
			Name:  "Recomendações para Você",
			Type:  "Advice",
			Items: items,
		})
	}

	if len(nearBikkers) > 0 {
		var items []domain.HomeItem
		for _, nb := range nearBikkers {
			items = append(items, domain.HomeItem{
				ID:       nb.ID,
				Title:    nb.Title,
				Subtitle: nb.Subtitle,
				ImageURL: nb.ImageURL,
				Distance: nb.Distance,
			})
		}
		collections = append(collections, domain.HomeCollection{
			ID:    "near_collection",
			Name:  "Profissionais por Perto",
			Type:  "ProfileNear",
			Items: items,
		})
	}

	return &domain.HomeResponse{
		Collections: collections,
	}, nil
}

func (uc *HomeUseCase) GetCategories(ctx context.Context) ([]domain.Category, error) {
	categories, err := uc.redisService.GetCategories(ctx)
	if err == nil && len(categories) > 0 {
		return categories, nil
	}
	categories, err = uc.categoryRepo.GetAllActive(ctx)
	if err != nil {
		return nil, err
	}
	_ = uc.redisService.SetCategories(ctx, categories)
	return categories, nil
}

func (uc *HomeUseCase) GetCategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	cat, err := uc.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	services, err := uc.serviceRepo.GetServicesByCategoryID(ctx, id)
	if err == nil {
		cat.Services = services
	}
	return cat, nil
}

func (uc *HomeUseCase) GetServiceByID(ctx context.Context, id string) (*domain.Service, error) {
	return uc.serviceRepo.GetByID(ctx, id)
}

func (uc *HomeUseCase) GetServicesByBikkerID(ctx context.Context, bikkerID string) ([]domain.Service, error) {
	return uc.serviceRepo.GetActiveServicesByBikkerID(ctx, bikkerID)
}

func (uc *HomeUseCase) GetServicesByCategoryID(ctx context.Context, categoryID string) ([]domain.Service, error) {
	return uc.serviceRepo.GetServicesByCategoryID(ctx, categoryID)
}

func (uc *HomeUseCase) GetFeaturedAdviceServices(ctx context.Context, limit int) ([]domain.Service, error) {
	return uc.serviceRepo.GetFeaturedAdviceServices(ctx, limit)
}
