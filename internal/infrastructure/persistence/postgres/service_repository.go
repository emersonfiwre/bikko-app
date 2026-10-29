package postgres

import (
	"bikko-app/internal/domain"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

type serviceRepository struct{ pool *pgxpool.Pool }

func NewServiceRepository(pool *pgxpool.Pool) domain.ServiceRepository {
	return &serviceRepository{pool: pool}
}

const serviceSelect = `SELECT s.id::text,s.bikker_id::text,s.category_id::text,s.name,COALESCE(s.description,''),COALESCE(s.thumbnail_url,''),COALESCE(ST_Y(s.location_geom::geometry),0),COALESCE(ST_X(s.location_geom::geometry),0),COALESCE(s.reviews_average,0),COALESCE(s.total_reviews,0),s.is_active,s.created_at,s.updated_at,c.name,COALESCE(c.icon_url,''),COALESCE(c.description,''),c.is_active,u.full_name,COALESCE(b.profession,''),COALESCE(u.profile_photo_url,''),COALESCE((SELECT jsonb_agg(sp.photo_url ORDER BY sp.display_order,sp.id) FROM service_photos sp WHERE sp.service_id=s.id),'[]'::jsonb) FROM services s JOIN categories c ON c.id=s.category_id JOIN users u ON u.id=s.bikker_id JOIN bikkers b ON b.id=s.bikker_id `
const serviceVisible = ` s.deleted_at IS NULL AND u.deleted_at IS NULL `

func scanService(row pgx.Row) (*domain.Service, error) {
	var s domain.Service
	var c domain.Category
	var photos []byte
	err := row.Scan(&s.ID, &s.BikkerID, &s.CategoryID, &s.Name, &s.Description, &s.ThumbnailURL, &s.Latitude, &s.Longitude, &s.ReviewsAverage, &s.TotalReviews, &s.IsActive, &s.CreatedAt, &s.UpdatedAt, &c.Name, &c.IconURL, &c.Description, &c.IsActive, &s.ProviderName, &s.ProviderTitle, &s.ProviderPhotoURL, &photos)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	c.ID = s.CategoryID
	s.Category = &c
	if err = json.Unmarshal(photos, &s.Photos); err != nil {
		return nil, err
	}
	if s.Photos == nil {
		s.Photos = []string{}
	}
	s.Reviews = []domain.ServiceReviewItem{}
	return &s, nil
}
func (r *serviceRepository) getByID(ctx context.Context, id string, private bool) (*domain.Service, error) {
	if r.pool == nil {
		return nil, domain.ErrUnavailable
	}
	if !isValidUUID(id) {
		return nil, domain.ErrNotFound
	}
	where := ` WHERE s.id=$1::uuid AND ` + serviceVisible
	if !private {
		where += ` AND s.is_active AND b.is_active`
	}
	s, err := scanService(r.pool.QueryRow(ctx, serviceSelect+where, id))
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT COALESCE(u.full_name,''),COALESCE(u.profile_photo_url,''),r.rating,COALESCE(r.comment,''),r.created_at FROM reviews r JOIN users u ON u.id=r.reviewer_id WHERE r.service_id=$1::uuid ORDER BY r.created_at DESC,r.id LIMIT 100`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.ServiceReviewItem
		var created time.Time
		if err = rows.Scan(&item.ReviewProfile.Name, &item.ReviewProfile.ImageProfile, &item.Rating, &item.Comment, &created); err != nil {
			return nil, err
		}
		item.Date = created.Format("02/01/2006")
		s.Reviews = append(s.Reviews, item)
	}
	return s, rows.Err()
}
func (r *serviceRepository) GetByID(ctx context.Context, id string) (*domain.Service, error) {
	return r.getByID(ctx, id, false)
}
func (r *serviceRepository) GetServiceByID(ctx context.Context, id string) (*domain.Service, error) {
	return r.GetByID(ctx, id)
}
func (r *serviceRepository) list(ctx context.Context, where string, args ...any) ([]domain.Service, error) {
	if r.pool == nil {
		return nil, domain.ErrUnavailable
	}
	rows, err := r.pool.Query(ctx, serviceSelect+` WHERE `+serviceVisible+` AND `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Service{}
	for rows.Next() {
		s, e := scanService(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}
func (r *serviceRepository) GetServicesNear(ctx context.Context, lat, lon, radius float64, limit int) ([]domain.Service, error) {
	if !validCoordinates(&lat, &lon) || radius <= 0 || radius > 100000 || limit < 1 || limit > 100 {
		return nil, domain.ErrInvalidInput
	}
	return r.list(ctx, `s.is_active AND b.is_active AND ST_DWithin(s.location_geom,ST_SetSRID(ST_MakePoint($1,$2),4326)::geography,$3) ORDER BY s.location_geom <-> ST_SetSRID(ST_MakePoint($1,$2),4326)::geography,s.id LIMIT $4`, lon, lat, radius, limit)
}
func (r *serviceRepository) GetFeaturedAdviceServices(ctx context.Context, limit int) ([]domain.Service, error) {
	if limit < 1 || limit > 100 {
		return nil, domain.ErrInvalidInput
	}
	return r.list(ctx, `s.is_active AND b.is_active ORDER BY s.total_reviews DESC,s.created_at DESC LIMIT $1`, limit)
}
func (r *serviceRepository) GetActiveServicesByBikkerID(ctx context.Context, id string) ([]domain.Service, error) {
	if !isValidUUID(id) {
		return nil, domain.ErrInvalidInput
	}
	return r.list(ctx, `s.bikker_id=$1::uuid AND s.is_active AND b.is_active ORDER BY s.created_at DESC LIMIT 100`, id)
}
func (r *serviceRepository) GetServicesByBikkerID(ctx context.Context, id string) ([]domain.Service, error) {
	if !isValidUUID(id) {
		return nil, domain.ErrInvalidInput
	}
	return r.list(ctx, `s.bikker_id=$1::uuid ORDER BY s.created_at DESC LIMIT 100`, id)
}
func (r *serviceRepository) GetServicesByCategoryID(ctx context.Context, id string) ([]domain.Service, error) {
	if !isValidUUID(id) {
		return nil, domain.ErrInvalidInput
	}
	return r.list(ctx, `s.category_id=$1::uuid AND s.is_active AND b.is_active ORDER BY s.created_at DESC LIMIT 100`, id)
}
func (r *serviceRepository) SearchServices(ctx context.Context, q string) ([]domain.Service, error) {
	q = strings.TrimSpace(q)
	if q == "" || len([]rune(q)) > 100 {
		return []domain.Service{}, nil
	}
	return r.list(ctx, `s.is_active AND b.is_active AND (s.name ILIKE '%'||$1||'%' OR s.description ILIKE '%'||$1||'%') ORDER BY s.total_reviews DESC,s.created_at DESC LIMIT 50`, q)
}
func categoryID(ctx context.Context, q contractQuerier, input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", domain.ErrInvalidInput
	}
	var id string
	err := q.QueryRow(ctx, `SELECT id::text FROM categories WHERE (id::text=$1 OR lower(name)=lower($1)) AND is_active`, strings.TrimSpace(input)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	return id, err
}
func validateService(s *domain.Service) error {
	if s == nil || !isValidUUID(s.BikkerID) || strings.TrimSpace(s.Name) == "" || len([]rune(s.Name)) > 255 || len([]rune(s.Description)) > 4000 || len(s.Photos) > 10 {
		return domain.ErrInvalidInput
	}
	return nil
}
func storePhotos(ctx context.Context, tx pgx.Tx, id string, photos []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM service_photos WHERE service_id=$1::uuid`, id); err != nil {
		return err
	}
	for i, url := range photos {
		if strings.TrimSpace(url) == "" {
			return domain.ErrInvalidInput
		}
		if _, err := tx.Exec(ctx, `INSERT INTO service_photos(service_id,photo_url,display_order) VALUES($1::uuid,$2,$3)`, id, url, i); err != nil {
			return err
		}
	}
	return nil
}
func serviceFingerprint(bikkerID, categoryID, name, description, thumbnail string, lat, lon float64, isActive bool, photos []string) (string, error) {
	body, err := json.Marshal(struct {
		Bikker, Category, Name, Description, Thumbnail string
		Lat, Lon                                       float64
		Active                                         bool
		Photos                                         []string
	}{bikkerID, categoryID, name, description, thumbnail, lat, lon, isActive, photos})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(body)), nil
}

func serviceUpdateFingerprint(id, bikkerID, categoryID, name, description, thumbnail string, isActive bool, photos []string) (string, error) {
	body, err := json.Marshal(struct {
		ID, Bikker, Category, Name, Description, Thumbnail string
		Active                                             bool
		Photos                                             []string
	}{id, bikkerID, categoryID, name, description, thumbnail, isActive, photos})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(body)), nil
}

func (r *serviceRepository) CreateService(ctx context.Context, s *domain.Service) (*domain.Service, error) {
	if r.pool == nil {
		return nil, domain.ErrUnavailable
	}
	if err := validateService(s); err != nil {
		return nil, err
	}
	if !validCoordinates(&s.Latitude, &s.Longitude) {
		return nil, domain.ErrInvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var active bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bikkers b JOIN users u ON u.id=b.id WHERE b.id=$1::uuid AND (NOT $2::boolean OR b.is_active) AND u.deleted_at IS NULL AND u.is_bikker)`, s.BikkerID, s.IsActive).Scan(&active); err != nil {
		return nil, err
	}
	if !active {
		return nil, domain.ErrForbidden
	}
	category, err := categoryID(ctx, tx, s.CategoryID)
	if err != nil {
		return nil, err
	}
	thumbnail := s.ThumbnailURL
	if thumbnail == "" && len(s.Photos) > 0 {
		thumbnail = s.Photos[0]
	}
	photos := s.Photos
	if len(photos) == 0 && thumbnail != "" {
		photos = []string{thumbnail}
	}
	var fingerprint string
	if s.RequestKey != "" {
		var e error
		fingerprint, e = serviceFingerprint(s.BikkerID, category, strings.TrimSpace(s.Name), s.Description, thumbnail, s.Latitude, s.Longitude, s.IsActive, photos)
		if e != nil {
			return nil, e
		}
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, s.BikkerID+":"+s.RequestKey); err != nil {
			return nil, err
		}
		var previousID, previousFingerprint string
		err = tx.QueryRow(ctx, `SELECT id::text,COALESCE(request_fingerprint,'') FROM services WHERE bikker_id=$1::uuid AND request_key=$2`, s.BikkerID, s.RequestKey).Scan(&previousID, &previousFingerprint)
		if err == nil {
			if previousFingerprint != fingerprint {
				return nil, domain.ErrConflict
			}
			if err = tx.Commit(ctx); err != nil {
				return nil, err
			}
			return r.getByID(ctx, previousID, true)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	err = tx.QueryRow(ctx, `INSERT INTO services(bikker_id,category_id,name,description,thumbnail_url,location_geom,is_active,request_key,request_fingerprint) VALUES($1::uuid,$2::uuid,$3,$4,$5,ST_SetSRID(ST_MakePoint($6,$7),4326)::geography,$8,NULLIF($9,''),NULLIF($10,'')) RETURNING id::text`, s.BikkerID, category, strings.TrimSpace(s.Name), s.Description, thumbnail, s.Longitude, s.Latitude, s.IsActive, s.RequestKey, fingerprint).Scan(&s.ID)
	if err != nil {
		return nil, fmt.Errorf("create service: %w", err)
	}
	if err = storePhotos(ctx, tx, s.ID, photos); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.getByID(ctx, s.ID, true)
}
func (r *serviceRepository) UpdateService(ctx context.Context, s *domain.Service) (*domain.Service, error) {
	if r.pool == nil {
		return nil, domain.ErrUnavailable
	}
	if s == nil || !isValidUUID(s.ID) || !isValidUUID(s.BikkerID) || len(s.Photos) > 10 {
		return nil, domain.ErrInvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var existing string
	if err = tx.QueryRow(ctx, `SELECT s.id::text FROM services s JOIN users u ON u.id=s.bikker_id WHERE s.id=$1::uuid AND s.bikker_id=$2::uuid AND s.deleted_at IS NULL AND u.deleted_at IS NULL AND u.is_bikker FOR UPDATE OF s`, s.ID, s.BikkerID).Scan(&existing); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	category := ""
	if s.CategoryID != "" {
		category, err = categoryID(ctx, tx, s.CategoryID)
		if err != nil {
			return nil, err
		}
	}
	thumbnail := s.ThumbnailURL
	if thumbnail == "" && len(s.Photos) > 0 {
		thumbnail = s.Photos[0]
	}
	if len([]rune(s.Name)) > 255 || len([]rune(s.Description)) > 4000 {
		return nil, domain.ErrInvalidInput
	}
	var fingerprint string
	if s.RequestKey != "" {
		var e error
		fingerprint, e = serviceUpdateFingerprint(s.ID, s.BikkerID, category, s.Name, s.Description, thumbnail, s.IsActive, s.Photos)
		if e != nil {
			return nil, e
		}
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, s.BikkerID+":"+s.RequestKey); err != nil {
			return nil, err
		}
		var previousID, previousFingerprint string
		err = tx.QueryRow(ctx, `SELECT id::text,COALESCE(request_fingerprint,'') FROM services WHERE bikker_id=$1::uuid AND request_key=$2`, s.BikkerID, s.RequestKey).Scan(&previousID, &previousFingerprint)
		if err == nil {
			if previousID != s.ID || previousFingerprint != fingerprint {
				return nil, domain.ErrConflict
			}
			if err = tx.Commit(ctx); err != nil {
				return nil, err
			}
			return r.getByID(ctx, previousID, true)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	// Portfolio updates are full representations: an empty thumbnail deliberately
	// clears a cover that the provider removed, instead of retaining stale media.
	_, err = tx.Exec(ctx, `UPDATE services SET name=COALESCE(NULLIF($2,''),name),description=$3,thumbnail_url=$4,category_id=COALESCE(NULLIF($5,'')::uuid,category_id),is_active=$6,request_key=COALESCE(NULLIF($7,''),request_key),request_fingerprint=COALESCE(NULLIF($8,''),request_fingerprint),updated_at=NOW() WHERE id=$1::uuid`, s.ID, s.Name, s.Description, thumbnail, category, s.IsActive, s.RequestKey, fingerprint)
	if err != nil {
		return nil, err
	}
	if s.Photos != nil {
		if err = storePhotos(ctx, tx, s.ID, s.Photos); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.getByID(ctx, s.ID, true)
}
func (r *serviceRepository) DeleteService(ctx context.Context, id, actor string) error {
	if r.pool == nil {
		return domain.ErrUnavailable
	}
	if !isValidUUID(id) || !isValidUUID(actor) {
		return domain.ErrNotFound
	}
	tag, err := r.pool.Exec(ctx, `UPDATE services s SET deleted_at=COALESCE(s.deleted_at,NOW()),is_active=FALSE,updated_at=NOW() WHERE s.id=$1::uuid AND s.bikker_id=$2::uuid AND EXISTS(SELECT 1 FROM users u WHERE u.id=s.bikker_id AND u.deleted_at IS NULL AND u.is_bikker)`, id, actor)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
