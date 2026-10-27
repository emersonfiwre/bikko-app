package postgres

import (
	"bikko-app/internal/domain"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type bikkerRepository struct{ pool *pgxpool.Pool }

func NewBikkerRepository(pool *pgxpool.Pool) domain.BikkerRepository {
	return &bikkerRepository{pool: pool}
}

const bikkerSelect = `SELECT b.id::text,u.full_name,b.profession,COALESCE(u.profile_photo_url,''),COALESCE(b.rating,0)::text,COALESCE(b.total_reviews,0),COALESCE(b.experience_years,''),COALESCE(b.location_name,''),COALESCE(b.description,'') FROM bikkers b JOIN users u ON u.id=b.id `

func scanBikker(row pgx.Row) (*domain.BikkerProfileResponse, error) {
	var p domain.BikkerProfileResponse
	err := row.Scan(&p.ID, &p.Name, &p.Profession, &p.ProfilePhotoURL, &p.Rating, &p.TotalReviews, &p.ExperienceYears, &p.Location, &p.Description)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	p.OfferedServices = []domain.Service{}
	p.Reviews = []domain.ServiceReviewItem{}
	return &p, nil
}
func (r *bikkerRepository) getByID(ctx context.Context, id string, private bool) (*domain.BikkerProfileResponse, error) {
	if r.pool == nil {
		return nil, domain.ErrUnavailable
	}
	if !isValidUUID(id) {
		return nil, domain.ErrNotFound
	}
	query := bikkerSelect + ` WHERE b.id=$1::uuid AND u.deleted_at IS NULL AND u.is_bikker`
	if !private {
		query += ` AND b.is_active`
	}
	profile, err := scanBikker(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, err
	}
	services := &serviceRepository{pool: r.pool}
	if private {
		profile.OfferedServices, err = services.GetServicesByBikkerID(ctx, id)
	} else {
		profile.OfferedServices, err = services.GetActiveServicesByBikkerID(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT COALESCE(u.full_name,''),COALESCE(u.profile_photo_url,''),r.rating,COALESCE(r.comment,''),to_char(r.created_at,'DD/MM/YYYY'),COALESCE(s.name,''),b.profession FROM reviews r JOIN users u ON u.id=r.reviewer_id JOIN bikkers b ON b.id=r.bikker_id LEFT JOIN services s ON s.id=r.service_id WHERE r.bikker_id=$1::uuid ORDER BY r.created_at DESC,r.id LIMIT 100`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var review domain.ServiceReviewItem
		if err = rows.Scan(&review.ReviewProfile.Name, &review.ReviewProfile.ImageProfile, &review.Rating, &review.Comment, &review.Date, &review.ServiceTag, &review.ProviderTitle); err != nil {
			return nil, err
		}
		profile.Reviews = append(profile.Reviews, review)
	}
	return profile, rows.Err()
}
func (r *bikkerRepository) GetByID(ctx context.Context, id string) (*domain.BikkerProfileResponse, error) {
	return r.getByID(ctx, id, false)
}
func (r *bikkerRepository) GetNearBikkers(ctx context.Context, lat, lon float64, limit int) ([]domain.NearBikkerItem, error) {
	if r.pool == nil {
		return nil, domain.ErrUnavailable
	}
	if !validCoordinates(&lat, &lon) || limit < 1 || limit > 100 {
		return nil, domain.ErrInvalidInput
	}
	rows, err := r.pool.Query(ctx, `SELECT b.id::text,b.profession,u.full_name,COALESCE(u.profile_photo_url,''),ST_Distance(b.location_geom,ST_SetSRID(ST_MakePoint($1,$2),4326)::geography)/1000 FROM bikkers b JOIN users u ON u.id=b.id WHERE b.is_active AND u.is_bikker AND u.deleted_at IS NULL AND ST_DWithin(b.location_geom,ST_SetSRID(ST_MakePoint($1,$2),4326)::geography,10000) ORDER BY b.location_geom <-> ST_SetSRID(ST_MakePoint($1,$2),4326)::geography LIMIT $3`, lon, lat, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.NearBikkerItem{}
	for rows.Next() {
		var s domain.NearBikkerItem
		var distance float64
		if err = rows.Scan(&s.ID, &s.Title, &s.Subtitle, &s.ImageURL, &distance); err != nil {
			return nil, err
		}
		s.Distance = fmt.Sprintf("%.1f km de você", distance)
		items = append(items, s)
	}
	return items, rows.Err()
}
func (r *bikkerRepository) GetAll(ctx context.Context, limit int) ([]domain.BikkerProfileResponse, error) {
	if r.pool == nil {
		return nil, domain.ErrUnavailable
	}
	if limit < 1 || limit > 100 {
		return nil, domain.ErrInvalidInput
	}
	rows, err := r.pool.Query(ctx, bikkerSelect+` WHERE b.is_active AND u.is_bikker AND u.deleted_at IS NULL ORDER BY b.total_reviews DESC,b.id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.BikkerProfileResponse{}
	for rows.Next() {
		p, e := scanBikker(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, *p)
	}
	return items, rows.Err()
}
func (r *bikkerRepository) UpdateProviderProfile(ctx context.Context, id string, in domain.ProviderProfileUpdateRequest) (*domain.BikkerProfileResponse, error) {
	if r.pool == nil {
		return nil, domain.ErrUnavailable
	}
	if !isValidUUID(id) {
		return nil, domain.ErrForbidden
	}
	if len([]rune(in.Profession)) > 100 || (in.Description != nil && len([]rune(*in.Description)) > 4000) {
		return nil, domain.ErrInvalidInput
	}
	if (in.Latitude == nil) != (in.Longitude == nil) {
		return nil, domain.ErrInvalidInput
	}
	if in.Latitude != nil && !validCoordinates(in.Latitude, in.Longitude) {
		return nil, domain.ErrInvalidInput
	}
	tag, err := r.pool.Exec(ctx, `UPDATE bikkers b SET profession=COALESCE(NULLIF($2,''),profession),experience_years=COALESCE(NULLIF($3,''),experience_years),description=COALESCE($4::text,description),location_name=COALESCE(NULLIF($5,''),location_name),location_geom=CASE WHEN $6::double precision IS NULL THEN location_geom ELSE ST_SetSRID(ST_MakePoint($7::double precision,$6::double precision),4326)::geography END,is_active=COALESCE($8::boolean,is_active),updated_at=NOW() WHERE b.id=$1::uuid AND EXISTS(SELECT 1 FROM users u WHERE u.id=b.id AND u.deleted_at IS NULL AND u.is_bikker)`, id, strings.TrimSpace(in.Profession), strings.TrimSpace(in.ExperienceYears), in.Description, strings.TrimSpace(in.LocationName), in.Latitude, in.Longitude, in.IsActive)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrNotFound
	}
	return r.getByID(ctx, id, true)
}
func (r *bikkerRepository) UpsertBikkerProfile(ctx context.Context, b *domain.Bikker) (*domain.BikkerProfileResponse, error) {
	if b == nil {
		return nil, domain.ErrInvalidInput
	}
	return r.UpdateProviderProfile(ctx, b.ID, domain.ProviderProfileUpdateRequest{Profession: b.Profession, ExperienceYears: b.ExperienceYears, Description: &b.Description, LocationName: b.LocationName, Latitude: &b.Latitude, Longitude: &b.Longitude, IsActive: &b.IsActive})
}
