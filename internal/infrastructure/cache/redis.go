package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bikko-app/config"
	"bikko-app/internal/domain"

	"github.com/mmcloughlin/geohash"
	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	client *redis.Client
}

func NewRedisService(cfg *config.Config) (*RedisService, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.RedisPassword,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("falha ao conectar no Redis (%s): %w", cfg.GetRedisAddr(), err)
	}

	log.Println("Redis Client (go-redis/v9) conectado com sucesso!")
	return &RedisService{client: client}, nil
}

// Categories Cache (TTL: 24 Hours)
func (s *RedisService) GetCategories(ctx context.Context) ([]domain.Category, error) {
	val, err := s.client.Get(ctx, "home:categories").Result()
	if err != nil {
		return nil, err // Cache miss
	}
	var categories []domain.Category
	if err := json.Unmarshal([]byte(val), &categories); err != nil {
		return nil, err
	}
	return categories, nil
}

func (s *RedisService) SetCategories(ctx context.Context, categories []domain.Category) error {
	data, err := json.Marshal(categories)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "home:categories", data, 24*time.Hour).Err()
}

// Near Services Cache by Geohash (Precision 6 = ~1.2km x 0.6km, TTL: 5 Minutes)
func (s *RedisService) GetNearServices(ctx context.Context, lat, lon float64) ([]domain.Service, error) {
	hash := geohash.EncodeWithPrecision(lat, lon, 6)
	key := fmt.Sprintf("home:near:%s", hash)

	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err // Cache miss
	}

	var services []domain.Service
	if err := json.Unmarshal([]byte(val), &services); err != nil {
		return nil, err
	}
	return services, nil
}

func (s *RedisService) SetNearServices(ctx context.Context, lat, lon float64, services []domain.Service) error {
	hash := geohash.EncodeWithPrecision(lat, lon, 6)
	key := fmt.Sprintf("home:near:%s", hash)

	data, err := json.Marshal(services)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, data, 5*time.Minute).Err()
}

// Featured Advice Services Cache (TTL: 1 Hour)
func (s *RedisService) GetAdviceServices(ctx context.Context) ([]domain.Service, error) {
	val, err := s.client.Get(ctx, "home:advice").Result()
	if err != nil {
		return nil, err
	}
	var services []domain.Service
	if err := json.Unmarshal([]byte(val), &services); err != nil {
		return nil, err
	}
	return services, nil
}

func (s *RedisService) SetAdviceServices(ctx context.Context, services []domain.Service) error {
	data, err := json.Marshal(services)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "home:advice", data, 1*time.Hour).Err()
}
