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
	redisService *cache.RedisService
}

func NewHomeUseCase(
	categoryRepo domain.CategoryRepository,
	serviceRepo domain.ServiceRepository,
	redisService *cache.RedisService,
) *HomeUseCase {
	return &HomeUseCase{
		categoryRepo: categoryRepo,
		serviceRepo:  serviceRepo,
		redisService: redisService,
	}
}

func (uc *HomeUseCase) GetHomeFeed(ctx context.Context, lat, lon float64) (*domain.HomeResponse, error) {
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

	// 2. Near Services by Geohash (Redis Cache with 5m TTL / PostGIS ST_DWithin fallback)
	nearServices, err := uc.redisService.GetNearServices(ctx, lat, lon)
	if err != nil {
		log.Println("[HomeUseCase] Cache Miss: home:near:{geohash} -> executando query espacial PostGIS ST_DWithin (raio 10km)")
		// Default radius: 10,000 meters (10km), limit: 10 items
		nearServices, err = uc.serviceRepo.GetServicesNear(ctx, lat, lon, 10000.0, 10)
		if err != nil {
			return nil, err
		}
		if len(nearServices) > 0 {
			_ = uc.redisService.SetNearServices(ctx, lat, lon, nearServices)
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

	return &domain.HomeResponse{
		Categories:   categories,
		NearServices: nearServices,
		Advice:       adviceServices,
	}, nil
}
