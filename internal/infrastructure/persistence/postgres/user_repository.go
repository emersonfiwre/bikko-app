package postgres

import (
	"context"
	"fmt"

	"bikko-app/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection pool is not initialized")
	}

	query := `
		INSERT INTO users (full_name, email, phone, cpf, profile_photo_url, is_bikker)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), $6)
		RETURNING id, full_name, email, COALESCE(phone, ''), COALESCE(cpf, ''), COALESCE(profile_photo_url, ''), rating, total_ratings, is_bikker, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		user.FullName,
		user.Email,
		user.Phone,
		user.CPF,
		user.ProfilePhotoURL,
		user.IsBikker,
	).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Phone,
		&user.CPF,
		&user.ProfilePhotoURL,
		&user.Rating,
		&user.TotalRatings,
		&user.IsBikker,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if user.IsBikker {
		profession := "Prestador de Serviços"
		expYears := "1 ano"
		desc := ""
		locName := "São Paulo, SP"
		lat := -23.55052
		lon := -46.63330
		if user.Bikker != nil {
			if user.Bikker.Profession != "" {
				profession = user.Bikker.Profession
			}
			if user.Bikker.ExperienceYears != "" {
				expYears = user.Bikker.ExperienceYears
			}
			desc = user.Bikker.Description
			if user.Bikker.LocationName != "" {
				locName = user.Bikker.LocationName
			}
			if user.Bikker.Latitude != 0 {
				lat = user.Bikker.Latitude
			}
			if user.Bikker.Longitude != 0 {
				lon = user.Bikker.Longitude
			}
		}

		bikkerQuery := `
			INSERT INTO bikkers (id, profession, experience_years, description, location_name, location_geom, is_active)
			VALUES ($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography, TRUE)
			ON CONFLICT (id) DO UPDATE SET
				profession = EXCLUDED.profession,
				experience_years = EXCLUDED.experience_years,
				description = EXCLUDED.description,
				location_name = EXCLUDED.location_name,
				location_geom = EXCLUDED.location_geom,
				is_active = TRUE,
				updated_at = NOW()
		`
		if _, err := r.db.Exec(ctx, bikkerQuery, user.ID, profession, expYears, desc, locName, lon, lat); err != nil {
			return nil, fmt.Errorf("failed to create bikker profile: %w", err)
		}
	}

	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection pool is not initialized")
	}
	query := `
		SELECT id, full_name, email, COALESCE(phone, ''), COALESCE(cpf, ''), COALESCE(profile_photo_url, ''), rating, total_ratings, is_bikker, COALESCE(device_token, ''), created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Phone,
		&user.CPF,
		&user.ProfilePhotoURL,
		&user.Rating,
		&user.TotalRatings,
		&user.IsBikker,
		&user.DeviceToken,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found or error: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection pool is not initialized")
	}
	query := `
		SELECT id, full_name, email, COALESCE(phone, ''), COALESCE(cpf, ''), COALESCE(profile_photo_url, ''), rating, total_ratings, is_bikker, COALESCE(device_token, ''), created_at, updated_at
		FROM users
		WHERE (phone = $1 OR phone = RIGHT($1, 11) OR RIGHT(phone, 11) = RIGHT($1, 11) OR phone = RIGHT($1, 9))
		  AND deleted_at IS NULL
		LIMIT 1
	`

	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, phone).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Phone,
		&user.CPF,
		&user.ProfilePhotoURL,
		&user.Rating,
		&user.TotalRatings,
		&user.IsBikker,
		&user.DeviceToken,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found by phone: %w", err)
	}

	r.populateBikkerIfNeeded(ctx, user)
	return user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection pool is not initialized")
	}
	query := `
		SELECT id, full_name, email, COALESCE(phone, ''), COALESCE(cpf, ''), COALESCE(profile_photo_url, ''), rating, total_ratings, is_bikker, COALESCE(device_token, ''), created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Phone,
		&user.CPF,
		&user.ProfilePhotoURL,
		&user.Rating,
		&user.TotalRatings,
		&user.IsBikker,
		&user.DeviceToken,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found or error: %w", err)
	}

	r.populateBikkerIfNeeded(ctx, user)
	return user, nil
}

func (r *userRepository) populateBikkerIfNeeded(ctx context.Context, user *domain.User) {
	if r.db == nil || user == nil || !user.IsBikker {
		return
	}
	bikkerQuery := `
		SELECT COALESCE(profession, ''), COALESCE(experience_years, ''), COALESCE(description, ''), COALESCE(location_name, 'São Paulo, SP'),
		       COALESCE(ST_Y(location_geom::geometry), -23.5614), COALESCE(ST_X(location_geom::geometry), -46.6914), COALESCE(is_active, TRUE)
		FROM bikkers
		WHERE id = $1
	`
	b := &domain.Bikker{ID: user.ID}
	if err := r.db.QueryRow(ctx, bikkerQuery, user.ID).Scan(
		&b.Profession,
		&b.ExperienceYears,
		&b.Description,
		&b.LocationName,
		&b.Latitude,
		&b.Longitude,
		&b.IsActive,
	); err == nil {
		user.Bikker = b
	}
}

func (r *userRepository) UpdateUser(ctx context.Context, id string, name, email, phone string) error {
	if r.db == nil {
		return fmt.Errorf("database connection pool is not initialized")
	}
	query := `
		UPDATE users 
		SET full_name = $1, email = $2, phone = NULLIF($3, ''), updated_at = NOW() 
		WHERE id = $4 AND deleted_at IS NULL
	`
	tag, err := r.db.Exec(ctx, query, name, email, phone, id)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found or already deleted")
	}
	return nil
}

func (r *userRepository) UpdateProfilePhoto(ctx context.Context, id string, photoURL string) error {
	if r.db == nil {
		return fmt.Errorf("database connection pool is not initialized")
	}
	query := `
		UPDATE users
		SET profile_photo_url = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`
	tag, err := r.db.Exec(ctx, query, photoURL, id)
	if err != nil {
		return fmt.Errorf("failed to update profile photo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found or already deleted")
	}
	return nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
	if r.db == nil {
		return fmt.Errorf("database connection pool is not initialized")
	}
	query := `
		UPDATE users 
		SET deleted_at = NOW() 
		WHERE id = $1 AND deleted_at IS NULL
	`
	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found or already deleted")
	}
	return nil
}

func (r *userRepository) UpdateDeviceToken(ctx context.Context, id string, token *string) error {
	if r.db == nil {
		return fmt.Errorf("database connection pool is not initialized")
	}
	var query string
	var args []interface{}

	if token != nil && *token != "" {
		query = `
			UPDATE users 
			SET device_token = $1, updated_at = NOW() 
			WHERE id = $2 AND deleted_at IS NULL
		`
		args = []interface{}{*token, id}
	} else {
		query = `
			UPDATE users 
			SET device_token = NULL, updated_at = NOW() 
			WHERE id = $1 AND deleted_at IS NULL
		`
		args = []interface{}{id}
	}

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update device token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found or already deleted")
	}
	return nil
}

func (r *userRepository) GetDeviceToken(ctx context.Context, id string) (string, error) {
	if r.db == nil {
		return "", fmt.Errorf("database connection pool is not initialized")
	}
	query := `
		SELECT COALESCE(device_token, '')
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	var token string
	err := r.db.QueryRow(ctx, query, id).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("failed to get device token: %w", err)
	}
	return token, nil
}

func (r *userRepository) UpgradeToBikker(ctx context.Context, userID string, bikker *domain.Bikker) (*domain.User, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection pool is not initialized")
	}

	updateUserQuery := `
		UPDATE users 
		SET is_bikker = TRUE, updated_at = NOW() 
		WHERE id = $1 AND deleted_at IS NULL
	`
	tag, err := r.db.Exec(ctx, updateUserQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade user to bikker: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("user not found or already deleted")
	}

	profession := "Prestador de Serviços"
	expYears := "1 ano"
	desc := ""
	locName := "São Paulo, SP"
	lat := -23.55052
	lon := -46.63330
	if bikker != nil {
		if bikker.Profession != "" {
			profession = bikker.Profession
		}
		if bikker.ExperienceYears != "" {
			expYears = bikker.ExperienceYears
		}
		desc = bikker.Description
		if bikker.LocationName != "" {
			locName = bikker.LocationName
		}
		if bikker.Latitude != 0 {
			lat = bikker.Latitude
		}
		if bikker.Longitude != 0 {
			lon = bikker.Longitude
		}
	}

	bikkerQuery := `
		INSERT INTO bikkers (id, profession, experience_years, description, location_name, location_geom, is_active)
		VALUES ($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography, TRUE)
		ON CONFLICT (id) DO UPDATE SET
			profession = EXCLUDED.profession,
			experience_years = EXCLUDED.experience_years,
			description = EXCLUDED.description,
			location_name = EXCLUDED.location_name,
			location_geom = EXCLUDED.location_geom,
			is_active = TRUE,
			updated_at = NOW()
	`
	if _, err := r.db.Exec(ctx, bikkerQuery, userID, profession, expYears, desc, locName, lon, lat); err != nil {
		return nil, fmt.Errorf("failed to create or update bikker profile: %w", err)
	}

	upgradedUser, err := r.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upgraded user: %w", err)
	}
	upgradedUser.Bikker = bikker
	return upgradedUser, nil
}
