package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"bikko-app/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepository struct{ db *pgxpool.Pool }

func NewUserRepository(db *pgxpool.Pool) domain.UserRepository { return &userRepository{db: db} }

func identityDBError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var dbError *pgconn.PgError
	if errors.As(err, &dbError) && dbError.Code == "23505" {
		return fmt.Errorf("%s: %w", operation, domain.ErrConflict)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func rollbackIdentity(ctx context.Context, tx pgx.Tx) {
	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := tx.Rollback(rollbackCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		slog.Error("identity transaction rollback failed", slog.Any("error", err))
	}
}

const userColumns = `id, full_name, email, COALESCE(phone, ''), COALESCE(cpf, ''), COALESCE(profile_photo_url, ''), rating, total_ratings, is_bikker, COALESCE(device_token, ''), created_at, updated_at, COALESCE(location_name, ''), ST_Y(location_geom::geometry), ST_X(location_geom::geometry)`

func scanIdentity(row pgx.Row) (*domain.User, error) {
	user := &domain.User{}
	err := row.Scan(&user.ID, &user.FullName, &user.Email, &user.Phone, &user.CPF, &user.ProfilePhotoURL, &user.Rating, &user.TotalRatings, &user.IsBikker, &user.DeviceToken, &user.CreatedAt, &user.UpdatedAt, &user.LocationName, &user.Latitude, &user.Longitude)
	if err != nil {
		return nil, identityDBError("identity: read user", err)
	}
	return user, nil
}

func (r *userRepository) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	if r.db == nil {
		return nil, domain.ErrUnavailable
	}
	if user == nil {
		return nil, domain.ErrInvalidInput
	}
	phone, err := domain.NormalizePhone(user.Phone)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, identityDBError("identity: begin registration", err)
	}
	defer rollbackIdentity(ctx, tx)
	created, err := scanIdentity(tx.QueryRow(ctx, `INSERT INTO users (full_name,email,phone,cpf,profile_photo_url,is_bikker) VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),$6) RETURNING `+userColumns, user.FullName, user.Email, phone, user.CPF, user.ProfilePhotoURL, user.IsBikker))
	if err != nil {
		return nil, err
	}
	if user.IsBikker {
		if user.Bikker == nil {
			return nil, domain.ErrInvalidInput
		}
		if err := writeProviderIdentity(ctx, tx, created.ID, user.Bikker); err != nil {
			return nil, err
		}
		created.Bikker = user.Bikker
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, identityDBError("identity: commit registration", err)
	}
	return created, nil
}

func writeProviderIdentity(ctx context.Context, tx pgx.Tx, userID string, b *domain.Bikker) error {
	if b == nil {
		return domain.ErrInvalidInput
	}
	_, err := tx.Exec(ctx, `INSERT INTO bikkers(id,profession,experience_years,description,location_name,location_geom,is_active)
 VALUES($1,$2,$3,$4,$5,ST_SetSRID(ST_MakePoint($6,$7),4326)::geography,$8)
 ON CONFLICT(id) DO UPDATE SET profession=EXCLUDED.profession,experience_years=EXCLUDED.experience_years,description=EXCLUDED.description,location_name=EXCLUDED.location_name,location_geom=EXCLUDED.location_geom,is_active=EXCLUDED.is_active,updated_at=NOW()`, userID, b.Profession, b.ExperienceYears, b.Description, b.LocationName, b.Longitude, b.Latitude, b.IsActive)
	if err != nil {
		return identityDBError("identity: write provider", err)
	}
	return nil
}

func (r *userRepository) readUser(ctx context.Context, condition string, value string) (*domain.User, error) {
	if r.db == nil {
		return nil, domain.ErrUnavailable
	}
	user, err := scanIdentity(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE `+condition+` AND deleted_at IS NULL`, value))
	if err != nil {
		return nil, err
	}
	if user.IsBikker {
		b := &domain.Bikker{ID: user.ID}
		err := r.db.QueryRow(ctx, `SELECT profession,COALESCE(experience_years,''),COALESCE(description,''),COALESCE(location_name,''),COALESCE(ST_Y(location_geom::geometry),0),COALESCE(ST_X(location_geom::geometry),0),is_active FROM bikkers WHERE id=$1`, user.ID).Scan(&b.Profession, &b.ExperienceYears, &b.Description, &b.LocationName, &b.Latitude, &b.Longitude, &b.IsActive)
		if err != nil {
			return nil, identityDBError("identity: load provider", err)
		}
		user.Bikker = b
	}
	return user, nil
}
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.readUser(ctx, "email=$1", email)
}
func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	canonical, err := domain.NormalizePhone(phone)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	return r.readUser(ctx, "phone=$1", canonical)
}
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.readUser(ctx, "id=$1", id)
}

func (r *userRepository) UpdateUser(ctx context.Context, id, name, email, phone string) error {
	if r.db == nil {
		return domain.ErrUnavailable
	}
	canonical, err := domain.NormalizePhone(phone)
	if err != nil {
		return domain.ErrInvalidInput
	}
	tag, err := r.db.Exec(ctx, `UPDATE users SET full_name=$1,email=$2,phone=$3,updated_at=NOW() WHERE id=$4 AND deleted_at IS NULL`, name, email, canonical, id)
	if err != nil {
		return identityDBError("identity: update user", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (r *userRepository) UpdateProfilePhoto(ctx context.Context, id, photoURL string) error {
	if r.db == nil {
		return domain.ErrUnavailable
	}
	tag, err := r.db.Exec(ctx, `UPDATE users SET profile_photo_url=$1,updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL`, photoURL, id)
	if err != nil {
		return identityDBError("identity: update photo", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
	if r.db == nil {
		return domain.ErrUnavailable
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return identityDBError("identity: begin deletion", err)
	}
	defer rollbackIdentity(ctx, tx)
	tag, err := tx.Exec(ctx, `UPDATE users SET deleted_at=NOW(),device_token=NULL,notifications_enabled=FALSE,updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return identityDBError("identity: delete account", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	for _, statement := range []string{`UPDATE bikkers SET is_active=FALSE,updated_at=NOW() WHERE id=$1`, `UPDATE services SET is_active=FALSE,updated_at=NOW() WHERE bikker_id=$1`, `DELETE FROM user_devices WHERE user_id=$1`} {
		if _, err := tx.Exec(ctx, statement, id); err != nil {
			return identityDBError("identity: deactivate account dependencies", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return identityDBError("identity: commit account deletion", err)
	}
	return nil
}
func (r *userRepository) UpgradeToBikker(ctx context.Context, id string, b *domain.Bikker) (*domain.User, error) {
	if r.db == nil {
		return nil, domain.ErrUnavailable
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, identityDBError("identity: begin provider upgrade", err)
	}
	defer rollbackIdentity(ctx, tx)
	user, err := scanIdentity(tx.QueryRow(ctx, `UPDATE users SET is_bikker=TRUE,updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL RETURNING `+userColumns, id))
	if err != nil {
		return nil, err
	}
	if err := writeProviderIdentity(ctx, tx, id, b); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, identityDBError("identity: commit provider upgrade", err)
	}
	user.Bikker = b
	return user, nil
}

// Compatibility entry points use the client application scope; runtime controllers
// use UserDeviceRepository to keep installations of both apps independent.
func (r *userRepository) UpdateDeviceToken(ctx context.Context, id string, token *string) error {
	if token == nil || *token == "" {
		return r.DeleteDevice(ctx, id, "client", "")
	}
	return r.RegisterDevice(ctx, id, "client", *token)
}
func (r *userRepository) GetDeviceToken(ctx context.Context, id string) (string, error) {
	tokens, err := r.GetDeviceTokens(ctx, id)
	if err != nil {
		return "", err
	}
	if len(tokens) == 0 {
		return "", nil
	}
	return tokens[0], nil
}
func (r *userRepository) RegisterDevice(ctx context.Context, id, appType, token string) error {
	if r.db == nil {
		return domain.ErrUnavailable
	}
	if (appType != "client" && appType != "provider") || token == "" || len(token) > 4096 {
		return domain.ErrInvalidInput
	}
	tag, err := r.db.Exec(ctx, `INSERT INTO user_devices(user_id,app_type,token) SELECT id,$2,$3 FROM users WHERE id=$1 AND deleted_at IS NULL ON CONFLICT(token) DO UPDATE SET user_id=EXCLUDED.user_id,app_type=EXCLUDED.app_type,updated_at=NOW()`, id, appType, token)
	if err != nil {
		return identityDBError("identity: register device", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (r *userRepository) DeleteDevice(ctx context.Context, id, appType, token string) error {
	if r.db == nil {
		return domain.ErrUnavailable
	}
	if appType != "client" && appType != "provider" {
		return domain.ErrInvalidInput
	}
	_, err := r.db.Exec(ctx, `DELETE FROM user_devices WHERE user_id=$1 AND app_type=$2 AND ($3='' OR token=$3)`, id, appType, token)
	if err != nil {
		return identityDBError("identity: unregister device", err)
	}
	return nil
}
func (r *userRepository) GetDeviceTokens(ctx context.Context, id string) ([]string, error) {
	if r.db == nil {
		return nil, domain.ErrUnavailable
	}
	rows, err := r.db.Query(ctx, `SELECT d.token FROM user_devices d JOIN users u ON u.id=d.user_id WHERE d.user_id=$1 AND u.deleted_at IS NULL ORDER BY d.updated_at DESC`, id)
	if err != nil {
		return nil, identityDBError("identity: list devices", err)
	}
	defer rows.Close()
	tokens := []string{}
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, identityDBError("identity: read device", err)
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		return nil, identityDBError("identity: list devices", err)
	}
	return tokens, nil
}
