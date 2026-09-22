package postgres

import (
	"context"
	"fmt"
	"time"

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
	if r.pool == nil {
		return nil, domain.ErrNotFound
	}

	query := `
		SELECT 
			s.id, s.bikker_id, s.category_id, s.name, COALESCE(s.description, ''), COALESCE(s.thumbnail_url, ''),
			ST_Y(s.location_geom::geometry) AS lat, ST_X(s.location_geom::geometry) AS lon,
			COALESCE((SELECT ROUND(AVG(r.rating)::numeric, 1) FROM reviews r WHERE r.service_id = s.id), 0.0) AS reviews_average,
			COALESCE((SELECT COUNT(r.id) FROM reviews r WHERE r.service_id = s.id), 0) AS total_reviews,
			s.is_active, s.created_at, s.updated_at,
			c.id, c.name, COALESCE(c.icon_url, ''), COALESCE(c.description, ''), c.is_active,
			COALESCE(u.full_name, ''), COALESCE(b.profession, ''), COALESCE(u.profile_photo_url, '')
		FROM services s
		INNER JOIN categories c ON c.id = s.category_id
		LEFT JOIN users u ON u.id = s.bikker_id
		LEFT JOIN bikkers b ON b.id = s.bikker_id
		WHERE s.id::text = $1 OR s.id::text LIKE $1 || '%'
		LIMIT 1
	`
	var s domain.Service
	var c domain.Category
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.BikkerID, &s.CategoryID, &s.Name, &s.Description, &s.ThumbnailURL,
		&s.Latitude, &s.Longitude, &s.ReviewsAverage, &s.TotalReviews, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		&c.ID, &c.Name, &c.IconURL, &c.Description, &c.IsActive,
		&s.ProviderName, &s.ProviderTitle, &s.ProviderPhotoURL,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	s.Category = &c

	// Load photos
	photosQuery := `SELECT photo_url FROM service_photos WHERE service_id = $1 ORDER BY display_order ASC`
	photoRows, err := r.pool.Query(ctx, photosQuery, s.ID)
	if err == nil {
		defer photoRows.Close()
		var photos []string
		for photoRows.Next() {
			var photoURL string
			if err := photoRows.Scan(&photoURL); err == nil {
				photos = append(photos, photoURL)
			}
		}
		s.Photos = photos
	}
	if s.Photos == nil {
		s.Photos = []string{}
	}

	// Load reviews
	reviewsQuery := `
		SELECT 
			COALESCE(u.full_name, 'Cliente'),
			COALESCE(u.profile_photo_url, ''),
			r.rating,
			COALESCE(r.comment, ''),
			COALESCE(r.like_count, 0),
			COALESCE(r.dislike_count, 0),
			r.created_at
		FROM reviews r
		LEFT JOIN users u ON r.reviewer_id = u.id
		WHERE r.service_id = $1
		ORDER BY r.created_at DESC
	`
	reviewRows, err := r.pool.Query(ctx, reviewsQuery, s.ID)
	if err == nil {
		defer reviewRows.Close()
		var reviews []domain.ServiceReviewItem
		for reviewRows.Next() {
			var userName, userAvatar, comment string
			var rating, like, dislike int
			var createdAt time.Time
			if err := reviewRows.Scan(&userName, &userAvatar, &rating, &comment, &like, &dislike, &createdAt); err == nil {
				dateStr := createdAt.Format("02/01/2006")
				reviews = append(reviews, domain.ServiceReviewItem{
					ReviewProfile: domain.ServiceReviewProfile{
						Name:         userName,
						ImageProfile: userAvatar,
					},
					Rating:  rating,
					Comment: comment,
					Date:    dateStr,
					Like:    like,
					Dislike: dislike,
				})
			}
		}
		s.Reviews = reviews
	}
	if s.Reviews == nil {
		s.Reviews = []domain.ServiceReviewItem{}
	}

	return &s, nil
}

func (r *serviceRepository) GetServiceByID(ctx context.Context, id string) (*domain.Service, error) {
	return r.GetByID(ctx, id)
}

func (r *serviceRepository) GetServicesNear(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]domain.Service, error) {
	query := `
		SELECT 
			s.id, s.bikker_id, s.category_id, s.name, COALESCE(s.description, ''), COALESCE(s.thumbnail_url, ''),
			ST_Y(s.location_geom::geometry) AS lat, ST_X(s.location_geom::geometry) AS lon,
			COALESCE((SELECT ROUND(AVG(r.rating)::numeric, 1) FROM reviews r WHERE r.service_id = s.id), 0.0) AS reviews_average,
			COALESCE((SELECT COUNT(r.id) FROM reviews r WHERE r.service_id = s.id), 0) AS total_reviews,
			s.is_active, s.created_at, s.updated_at,
			c.id, c.name, COALESCE(c.icon_url, ''), COALESCE(c.description, ''), c.is_active
		FROM services s
		INNER JOIN categories c ON c.id = s.category_id
		WHERE s.is_active = TRUE
		  AND ST_DWithin(
			s.location_geom,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			$3
		  )
		ORDER BY s.location_geom <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
		LIMIT $4
	`
	rows, err := r.pool.Query(ctx, query, lon, lat, radiusMeters, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar serviços por geolocalização: %w", err)
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
			COALESCE((SELECT ROUND(AVG(r.rating)::numeric, 1) FROM reviews r WHERE r.service_id = s.id), 0.0) AS reviews_average,
			COALESCE((SELECT COUNT(r.id) FROM reviews r WHERE r.service_id = s.id), 0) AS total_reviews,
			s.is_active, s.created_at, s.updated_at,
			c.id, c.name, COALESCE(c.icon_url, ''), COALESCE(c.description, ''), c.is_active
		FROM services s
		INNER JOIN categories c ON c.id = s.category_id
		WHERE s.is_active = TRUE
		ORDER BY (SELECT COUNT(r.id) FROM reviews r WHERE r.service_id = s.id) DESC, s.created_at DESC
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
			COALESCE((SELECT ROUND(AVG(r.rating)::numeric, 1) FROM reviews r WHERE r.service_id = s.id), 0.0) AS reviews_average,
			COALESCE((SELECT COUNT(r.id) FROM reviews r WHERE r.service_id = s.id), 0) AS total_reviews,
			s.is_active, s.created_at, s.updated_at,
			c.id, c.name, COALESCE(c.icon_url, ''), COALESCE(c.description, ''), c.is_active
		FROM services s
		INNER JOIN categories c ON c.id = s.category_id
		WHERE (s.bikker_id::text = $1 OR s.bikker_id::text LIKE $1 || '%') AND s.is_active = TRUE
		ORDER BY s.created_at ASC
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

func (r *serviceRepository) GetServicesByCategoryID(ctx context.Context, categoryID string) ([]domain.Service, error) {
	query := `
		SELECT 
			s.id, s.bikker_id, s.category_id, s.name, COALESCE(s.description, ''), COALESCE(s.thumbnail_url, ''),
			ST_Y(s.location_geom::geometry) AS lat, ST_X(s.location_geom::geometry) AS lon,
			COALESCE((SELECT ROUND(AVG(r.rating)::numeric, 1) FROM reviews r WHERE r.service_id = s.id), 0.0) AS reviews_average,
			COALESCE((SELECT COUNT(r.id) FROM reviews r WHERE r.service_id = s.id), 0) AS total_reviews,
			s.is_active, s.created_at, s.updated_at,
			c.id, c.name, COALESCE(c.icon_url, ''), COALESCE(c.description, ''), c.is_active
		FROM services s
		INNER JOIN categories c ON c.id = s.category_id
		WHERE (s.category_id::text = $1 OR s.category_id::text LIKE $1 || '%' OR c.id::text LIKE $1 || '%') AND s.is_active = TRUE
		ORDER BY s.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar serviços por categoria: %w", err)
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
			COALESCE((SELECT ROUND(AVG(r.rating)::numeric, 1) FROM reviews r WHERE r.service_id = s.id), 0.0) AS reviews_average,
			COALESCE((SELECT COUNT(r.id) FROM reviews r WHERE r.service_id = s.id), 0) AS total_reviews,
			s.is_active, s.created_at, s.updated_at,
			c.id, c.name, COALESCE(c.icon_url, ''), COALESCE(c.description, ''), c.is_active
		FROM services s
		INNER JOIN categories c ON c.id = s.category_id
		WHERE s.is_active = TRUE
		  AND (s.name ILIKE '%' || $1 || '%' OR s.description ILIKE '%' || $1 || '%')
		ORDER BY (SELECT COUNT(r.id) FROM reviews r WHERE r.service_id = s.id) DESC, s.created_at DESC
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
