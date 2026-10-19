package postgres

import (
	"context"
	"fmt"
	"time"

	"bikko-app/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type bikkerRepository struct {
	pool *pgxpool.Pool
}

func NewBikkerRepository(pool *pgxpool.Pool) domain.BikkerRepository {
	return &bikkerRepository{pool: pool}
}

func (r *bikkerRepository) GetByID(ctx context.Context, idOrSlug string) (*domain.BikkerProfileResponse, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("banco de dados não inicializado")
	}

	query := `
		SELECT 
			b.id, 
			u.full_name, 
			b.profession, 
			COALESCE(u.profile_photo_url, ''),
			COALESCE((SELECT ROUND(AVG(r.rating)::numeric, 1) FROM reviews r WHERE r.bikker_id = b.id), 0.0) AS rating,
			COALESCE((SELECT COUNT(r.id) FROM reviews r WHERE r.bikker_id = b.id), 0) AS total_reviews,
			COALESCE(b.experience_years, ''), 
			COALESCE(b.location_name, ''),
			COALESCE(b.description, '')
		FROM bikkers b
		JOIN users u ON b.id = u.id
		WHERE b.id::text = $1 OR LOWER(REPLACE(u.full_name, ' ', '_')) = LOWER($1) OR u.email LIKE $1 || '%'
		LIMIT 1
	`

	var resp domain.BikkerProfileResponse
	var ratingFloat float64

	err := r.pool.QueryRow(ctx, query, idOrSlug).Scan(
		&resp.ID,
		&resp.Name,
		&resp.Profession,
		&resp.ProfilePhotoURL,
		&ratingFloat,
		&resp.TotalReviews,
		&resp.ExperienceYears,
		&resp.Location,
		&resp.Description,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	resp.Rating = fmt.Sprintf("%.1f", ratingFloat)

	// 1. Fetch Offered Services
	servicesQuery := `
		SELECT 
			s.id, s.bikker_id, s.category_id, s.name, COALESCE(s.description, ''),
			COALESCE(s.thumbnail_url, ''), 
			COALESCE((SELECT ROUND(AVG(r.rating)::numeric, 1) FROM reviews r WHERE r.service_id = s.id), 0.0) AS reviews_average,
			COALESCE((SELECT COUNT(r.id) FROM reviews r WHERE r.service_id = s.id), 0) AS total_reviews,
			COALESCE(c.id::text, ''), COALESCE(c.name, ''), COALESCE(c.icon_url, '')
		FROM services s
		LEFT JOIN categories c ON s.category_id = c.id
		WHERE s.bikker_id = $1 AND s.is_active = TRUE
		ORDER BY s.created_at ASC
	`
	serviceRows, err := r.pool.Query(ctx, servicesQuery, resp.ID)
	if err == nil {
		defer serviceRows.Close()
		var services []domain.Service
		for serviceRows.Next() {
			var s domain.Service
			var catID, catName, catIcon string
			if err := serviceRows.Scan(
				&s.ID, &s.BikkerID, &s.CategoryID, &s.Name, &s.Description,
				&s.ThumbnailURL, &s.ReviewsAverage, &s.TotalReviews,
				&catID, &catName, &catIcon,
			); err == nil {
				if catID != "" {
					s.Category = &domain.Category{
						ID:      catID,
						Name:    catName,
						IconURL: catIcon,
					}
				}
				s.IsActive = true
				services = append(services, s)
			}
		}
		resp.OfferedServices = services
	}
	if resp.OfferedServices == nil {
		resp.OfferedServices = []domain.Service{}
	}

	// 2. Fetch Reviews
	reviewsQuery := `
		SELECT 
			COALESCE(u.full_name, 'Cliente'),
			COALESCE(u.profile_photo_url, ''),
			r.rating,
			COALESCE(r.comment, ''),
			COALESCE(r.like_count, 0),
			COALESCE(r.dislike_count, 0),
			r.created_at,
			COALESCE(s.name, '') as service_tag,
			COALESCE(b.profession, '') as provider_title
		FROM reviews r
		LEFT JOIN users u ON r.reviewer_id = u.id
		LEFT JOIN services s ON r.service_id = s.id
		LEFT JOIN bikkers b ON r.bikker_id = b.id
		WHERE r.bikker_id = $1
		ORDER BY r.created_at DESC
	`
	reviewRows, err := r.pool.Query(ctx, reviewsQuery, resp.ID)
	if err == nil {
		defer reviewRows.Close()
		var reviews []domain.ServiceReviewItem
		for reviewRows.Next() {
			var userName, userAvatar, comment, serviceTag, providerTitle string
			var rating, like, dislike int
			var createdAt time.Time
			if err := reviewRows.Scan(
				&userName, &userAvatar, &rating, &comment, &like, &dislike,
				&createdAt, &serviceTag, &providerTitle,
			); err == nil {
				reviews = append(reviews, domain.ServiceReviewItem{
					ReviewProfile: domain.ServiceReviewProfile{
						Name:         userName,
						ImageProfile: userAvatar,
					},
					Rating:        rating,
					Comment:       comment,
					Date:          createdAt.Format("02/01/2006"),
					Like:          like,
					Dislike:       dislike,
					ServiceTag:    serviceTag,
					ProviderTitle: providerTitle,
				})
			}
		}
		resp.Reviews = reviews
	}
	if resp.Reviews == nil {
		resp.Reviews = []domain.ServiceReviewItem{}
	}

	return &resp, nil
}

func (r *bikkerRepository) GetNearBikkers(ctx context.Context, lat, lon float64, limit int) ([]domain.NearBikkerItem, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("banco de dados não inicializado")
	}

	query := `
		SELECT 
			b.id, 
			b.profession, 
			u.full_name, 
			COALESCE(u.profile_photo_url, ''),
			ROUND((ST_Distance(b.location_geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) / 1000)::numeric, 1) as dist_km
		FROM bikkers b
		JOIN users u ON b.id = u.id
		WHERE b.is_active = TRUE
		ORDER BY b.location_geom <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography ASC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, lon, lat, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar busca por perto de bikkers: %w", err)
	}
	defer rows.Close()

	var items []domain.NearBikkerItem
	for rows.Next() {
		var id, profession, fullName, photo string
		var distKm float64
		if err := rows.Scan(&id, &profession, &fullName, &photo, &distKm); err != nil {
			return nil, err
		}

		distStr := fmt.Sprintf("%.1f km de você", distKm)
		if distKm <= 0.05 {
			distStr = "0.9 km de você"
		}

		items = append(items, domain.NearBikkerItem{
			ID:       id,
			Title:    profession,
			Subtitle: fullName,
			ImageURL: photo,
			Distance: distStr,
		})
	}

	return items, nil
}

func (r *bikkerRepository) GetAll(ctx context.Context, limit int) ([]domain.BikkerProfileResponse, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("banco de dados não inicializado")
	}

	query := `
		SELECT 
			b.id, 
			u.full_name, 
			b.profession, 
			COALESCE(u.profile_photo_url, ''),
			COALESCE((SELECT ROUND(AVG(r.rating)::numeric, 1) FROM reviews r WHERE r.bikker_id = b.id), 0.0) AS rating,
			COALESCE((SELECT COUNT(r.id) FROM reviews r WHERE r.bikker_id = b.id), 0) AS total_reviews,
			COALESCE(b.experience_years, ''), 
			COALESCE(b.location_name, ''),
			COALESCE(b.description, '')
		FROM bikkers b
		JOIN users u ON b.id = u.id
		WHERE b.is_active = TRUE
		ORDER BY (SELECT COUNT(r.id) FROM reviews r WHERE r.bikker_id = b.id) DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.BikkerProfileResponse
	for rows.Next() {
		var resp domain.BikkerProfileResponse
		var ratingFloat float64
		if err := rows.Scan(
			&resp.ID, &resp.Name, &resp.Profession, &resp.ProfilePhotoURL,
			&ratingFloat, &resp.TotalReviews, &resp.ExperienceYears, &resp.Location, &resp.Description,
		); err == nil {
			resp.Rating = fmt.Sprintf("%.1f", ratingFloat)
			resp.OfferedServices = []domain.Service{}
			resp.Reviews = []domain.ServiceReviewItem{}
			list = append(list, resp)
		}
	}

	return list, nil
}

func (r *bikkerRepository) UpsertBikkerProfile(ctx context.Context, bikker *domain.Bikker) (*domain.BikkerProfileResponse, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("banco de dados não inicializado")
	}
	if bikker == nil || bikker.ID == "" {
		return nil, fmt.Errorf("bikker id is required")
	}

	profession := bikker.Profession
	if profession == "" {
		profession = "Prestador de Serviços"
	}
	locationName := bikker.LocationName
	if locationName == "" {
		locationName = "São Paulo, SP"
	}
	lat := bikker.Latitude
	lon := bikker.Longitude
	if lat == 0 {
		lat = -23.55052
	}
	if lon == 0 {
		lon = -46.63330
	}

	_, _ = r.pool.Exec(ctx, `UPDATE users SET is_bikker = TRUE, updated_at = NOW() WHERE id::text = $1`, bikker.ID)

	query := `
		INSERT INTO bikkers (id, profession, experience_years, description, location_name, location_geom, is_active, updated_at)
		VALUES ($1::uuid, $2, NULLIF($3, ''), NULLIF($4, ''), $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography, $8, NOW())
		ON CONFLICT (id) DO UPDATE SET
			profession = COALESCE(NULLIF(EXCLUDED.profession, ''), bikkers.profession),
			experience_years = COALESCE(EXCLUDED.experience_years, bikkers.experience_years),
			description = COALESCE(EXCLUDED.description, bikkers.description),
			location_name = COALESCE(NULLIF(EXCLUDED.location_name, ''), bikkers.location_name),
			location_geom = EXCLUDED.location_geom,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
	`
	if _, err := r.pool.Exec(ctx, query,
		bikker.ID,
		profession,
		bikker.ExperienceYears,
		bikker.Description,
		locationName,
		lon,
		lat,
		bikker.IsActive,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert bikker profile: %w", err)
	}

	if !bikker.IsActive {
		_, _ = r.pool.Exec(ctx, `
			UPDATE services
			SET is_active = FALSE, updated_at = NOW()
			WHERE bikker_id = $1::uuid
		`, bikker.ID)
	}

	return r.GetByID(ctx, bikker.ID)
}
