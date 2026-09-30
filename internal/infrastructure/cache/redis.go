package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
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
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("redis indisponível")
	}
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
	if s == nil || s.client == nil {
		return nil
	}
	data, err := json.Marshal(categories)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "home:categories", data, 24*time.Hour).Err()
}

// Near Services Cache by Geohash (Precision 6 = ~1.2km x 0.6km, TTL: 5 Minutes)
func (s *RedisService) GetNearServices(ctx context.Context, lat, lon float64) ([]domain.Service, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("redis indisponível")
	}
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
	if s == nil || s.client == nil {
		return nil
	}
	hash := geohash.EncodeWithPrecision(lat, lon, 6)
	key := fmt.Sprintf("home:near:%s", hash)

	data, err := json.Marshal(services)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, data, 5*time.Minute).Err()
}

// Near Bikkers Cache by Geohash (Precision 6 = ~1.2km x 0.6km, TTL: 5 Minutes)
func (s *RedisService) GetNearBikkers(ctx context.Context, lat, lon float64) ([]domain.NearBikkerItem, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("redis indisponível")
	}
	hash := geohash.EncodeWithPrecision(lat, lon, 6)
	key := fmt.Sprintf("home:near_bikkers:%s", hash)

	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err // Cache miss
	}

	var bikkers []domain.NearBikkerItem
	if err := json.Unmarshal([]byte(val), &bikkers); err != nil {
		return nil, err
	}
	return bikkers, nil
}

func (s *RedisService) SetNearBikkers(ctx context.Context, lat, lon float64, bikkers []domain.NearBikkerItem) error {
	if s == nil || s.client == nil {
		return nil
	}
	hash := geohash.EncodeWithPrecision(lat, lon, 6)
	key := fmt.Sprintf("home:near_bikkers:%s", hash)

	data, err := json.Marshal(bikkers)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, data, 5*time.Minute).Err()
}

// Featured Advice Services Cache (TTL: 1 Hour)
func (s *RedisService) GetAdviceServices(ctx context.Context) ([]domain.Service, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("redis indisponível")
	}
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
	if s == nil || s.client == nil {
		return nil
	}
	data, err := json.Marshal(services)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "home:advice", data, 1*time.Hour).Err()
}

// Features (Feature Toggles / Flags)
func (s *RedisService) GetFeatures(ctx context.Context) (map[string]bool, error) {
	if s == nil || s.client == nil {
		return map[string]bool{
			"enable_profile_reviews":     false,
			"enable_quote_counter_offer": true,
			"enable_instant_chat":        false,
			"enable_pix_direct_payment":  true,
			"enable_dark_mode_beta":      false,
		}, nil
	}

	res, err := s.client.HGetAll(ctx, "bikkofy:features").Result()
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		defaults := map[string]interface{}{
			"enable_profile_reviews":     "false",
			"enable_quote_counter_offer": "true",
			"enable_instant_chat":        "false",
			"enable_pix_direct_payment":  "true",
			"enable_dark_mode_beta":      "false",
		}
		if err := s.client.HSet(ctx, "bikkofy:features", defaults).Err(); err != nil {
			log.Printf("[RedisService] Error seeding feature toggles: %v", err)
		}
		res = map[string]string{
			"enable_profile_reviews":     "false",
			"enable_quote_counter_offer": "true",
			"enable_instant_chat":        "false",
			"enable_pix_direct_payment":  "true",
			"enable_dark_mode_beta":      "false",
		}
	}

	features := make(map[string]bool, len(res))
	for k, v := range res {
		b, err := strconv.ParseBool(v)
		if err != nil {
			b = (v == "true" || v == "1")
		}
		features[k] = b
	}

	return features, nil
}
