package postgres

import (
	"context"
	"fmt"

	"bikko-app/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type serviceRepository struct {
	pool *pgxpool.Pool
}

func NewServiceRepository(pool *pgxpool.Pool) domain.ServiceRepository {
	return &serviceRepository{pool: pool}
}

func (r *serviceRepository) GetByID(ctx context.Context, id string) (*domain.Service, error) {
	query := `
		SELECT 
			s.id, s.bikker_id, s.category_id, s.name, COALESCE(s.description, ''), COALESCE(s.thumbnail_url, ''),
			ST_Y(s.location_geom::geometry) AS lat, ST_X(s.location_geom::geometry) AS lon,
			s.reviews_average, s.total_reviews, s.is_active, s.created_at, s.updated_at,
			c.id, c.name, COALESCE(c.icon_url, ''), COALESCE(c.description, ''), c.is_active
		FROM services s
		INNER JOIN categories c ON c.id = s.category_id
		WHERE s.id = $1
	`
	var s domain.Service
	var c domain.Category
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.BikkerID, &s.CategoryID, &s.Name, &s.Description, &s.ThumbnailURL,
		&s.Latitude, &s.Longitude, &s.ReviewsAverage, &s.TotalReviews, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		&c.ID, &c.Name, &c.IconURL, &c.Description, &c.IsActive,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	s.Category = &c
	return &s, nil
}

func (r *serviceRepository) GetServicesNear(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]domain.Service, error) {
	query := `
		SELECT 
			s.id, s.bikker_id, s.category_id, s.name, COALESCE(s.description, ''), COALESCE(s.thumbnail_url, ''),
			ST_Y(s.location_geom::geometry) AS lat, ST_X(s.location_geom::geometry) AS lon,
			s.reviews_average, s.total_reviews, s.is_active, s.created_at, s.updated_at,
			c.id, c.name, COALESCE(c.icon_url, ''), COALESCE(c.description, ''), c.is_active
		FROM services s
		INNER JOIN categories c ON c.id = s.category_id
		WHERE s.is_active = TRUE
		  AND ST_DWithin(s.location_geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
		ORDER BY s.location_geom <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography ASC
		LIMIT $4
	`
	rows, err := r.pool.Query(ctx, query, lon, lat, radiusMeters, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar serviços próximos com PostGIS: %w", err)
	}
	defer rows.Close()

	var services []domain.Service
	for rows.Next() {
		var s domain.Service
		var c domain.Category
		err := rows.Scan(
			&s.ID, &s.BikkerID, &s.CategoryID, &s.Name, &s.Description, &s.ThumbnailURL,
			&s.Latitude, &s.Longitude, &s.ReviewsAverage, &s.TotalReviews, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
			&c.ID, &c.Name, &c.IconURL, &c.Description, &c.IsActive,
		)
		if err != nil {
			return nil, err
		}
		s.Category = &c
		services = append(services, s)
	}
	return services, nil
}

func (r *serviceRepository) GetFeaturedAdviceServices(ctx context.Context, limit int) ([]domain.Service, error) {
	query := `
		SELECT 
			s.id, s.bikker_id, s.category_id, s.name, COALESCE(s.description, ''), COALESCE(s.thumbnail_url, ''),
			ST_Y(s.location_geom::geometry) AS lat, ST_X(s.location_geom::geometry) AS lon,
			s.reviews_average, s.total_reviews, s.is_active, s.created_at, s.updated_at,
			c.id, c.name, COALESCE(c.icon_url, ''), COALESCE(c.description, ''), c.is_active
		FROM services s
		INNER JOIN categories c ON c.id = s.category_id
		WHERE s.is_active = TRUE
		ORDER BY s.reviews_average DESC, s.total_reviews DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar serviços recomendados: %w", err)
	}
	defer rows.Close()

	var services []domain.Service
	for rows.Next() {
		var s domain.Service
		var c domain.Category
		err := rows.Scan(
			&s.ID, &s.BikkerID, &s.CategoryID, &s.Name, &s.Description, &s.ThumbnailURL,
			&s.Latitude, &s.Longitude, &s.ReviewsAverage, &s.TotalReviews, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
			&c.ID, &c.Name, &c.IconURL, &c.Description, &c.IsActive,
		)
		if err != nil {
			return nil, err
		}
		s.Category = &c
		services = append(services, s)
	}
	return services, nil
}

func (r *serviceRepository) GetActiveServicesByBikkerID(ctx context.Context, bikkerID string) ([]domain.Service, error) {
	query := `
		SELECT 
			s.id, s.bikker_id, s.category_id, s.name, COALESCE(s.description, ''), COALESCE(s.thumbnail_url, ''),
			ST_Y(s.location_geom::geometry) AS lat, ST_X(s.location_geom::geometry) AS lon,
			s.reviews_average, s.total_reviews, s.is_active, s.created_at, s.updated_at,
			c.id, c.name, COALESCE(c.icon_url, ''), COALESCE(c.description, ''), c.is_active
		FROM services s
		INNER JOIN categories c ON c.id = s.category_id
		WHERE s.bikker_id = $1 AND s.is_active = TRUE
	`
	rows, err := r.pool.Query(ctx, query, bikkerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []domain.Service
	for rows.Next() {
		var s domain.Service
		var c domain.Category
		err := rows.Scan(
			&s.ID, &s.BikkerID, &s.CategoryID, &s.Name, &s.Description, &s.ThumbnailURL,
			&s.Latitude, &s.Longitude, &s.ReviewsAverage, &s.TotalReviews, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
			&c.ID, &c.Name, &c.IconURL, &c.Description, &c.IsActive,
		)
		if err != nil {
			return nil, err
		}
		s.Category = &c
		services = append(services, s)
	}
	return services, nil
}

func (r *serviceRepository) SearchServices(ctx context.Context, q string) ([]domain.Service, error) {
	query := `
		SELECT 
			s.id, s.bikker_id, s.category_id, s.name, COALESCE(s.description, ''), COALESCE(s.thumbnail_url, ''),
			ST_Y(s.location_geom::geometry) AS lat, ST_X(s.location_geom::geometry) AS lon,
			s.reviews_average, s.total_reviews, s.is_active, s.created_at, s.updated_at,
			c.id, c.name, COALESCE(c.icon_url, ''), COALESCE(c.description, ''), c.is_active
		FROM services s
		INNER JOIN categories c ON c.id = s.category_id
		WHERE s.is_active = TRUE
		  AND (s.name ILIKE '%' || $1 || '%' OR s.description ILIKE '%' || $1 || '%')
		ORDER BY s.reviews_average DESC, s.total_reviews DESC
		LIMIT 20
	`
	rows, err := r.pool.Query(ctx, query, q)
	if err != nil {
		return nil, fmt.Errorf("erro ao pesquisar serviços: %w", err)
	}
	defer rows.Close()

	var services []domain.Service
	for rows.Next() {
		var s domain.Service
		var c domain.Category
		err := rows.Scan(
			&s.ID, &s.BikkerID, &s.CategoryID, &s.Name, &s.Description, &s.ThumbnailURL,
			&s.Latitude, &s.Longitude, &s.ReviewsAverage, &s.TotalReviews, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
			&c.ID, &c.Name, &c.IconURL, &c.Description, &c.IsActive,
		)
		if err != nil {
			return nil, err
		}
		s.Category = &c
		services = append(services, s)
	}
	return services, nil
}
