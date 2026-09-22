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

func (r *userRepository) CreateUser(ctx context.Context, user *domain.User, password string) (*domain.User, error) {
	// 'password' argument is the hashed password, we store it in PasswordHash.
	// Wait, interface domain.UserRepository says CreateUser(ctx, user, password)
	// I'll expect the password passed here to ALREADY BE HASHED by the service.
	// Or I can just hash it in the service and pass it as user.PasswordHash.
	// Let's assume the service hashes it and passes it as `password` param.

	query := `
		INSERT INTO users (full_name, email, phone, cpf, password_hash)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5)
		RETURNING id, full_name, email, COALESCE(phone, ''), COALESCE(cpf, ''), rating, total_ratings, is_bikker, created_at, updated_at
	`
	
	err := r.db.QueryRow(ctx, query,
		user.FullName,
		user.Email,
		user.Phone,
		user.CPF,
		password,
	).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Phone,
		&user.CPF,
		&user.Rating,
		&user.TotalRatings,
		&user.IsBikker,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, full_name, email, COALESCE(phone, ''), COALESCE(cpf, ''), password_hash, rating, total_ratings, is_bikker, created_at, updated_at
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
		&user.PasswordHash,
		&user.Rating,
		&user.TotalRatings,
		&user.IsBikker,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found or error: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, full_name, email, COALESCE(phone, ''), COALESCE(cpf, ''), password_hash, rating, total_ratings, is_bikker, created_at, updated_at
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
		&user.PasswordHash,
		&user.Rating,
		&user.TotalRatings,
		&user.IsBikker,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found or error: %w", err)
	}

	return user, nil
}

// Ensure the Delete functionality is added if not present in domain.UserRepository
// I will implement DeleteAccount for the soft delete support later when we see the profile_service.
func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
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
