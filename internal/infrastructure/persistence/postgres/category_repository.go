package postgres

import (
	"context"
	"fmt"

	"bikko-app/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type categoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) domain.CategoryRepository {
	return &categoryRepository{pool: pool}
}

func (r *categoryRepository) GetAllActive(ctx context.Context) ([]domain.Category, error) {
	query := `
		SELECT id, name, COALESCE(icon_url, ''), COALESCE(description, ''), is_active, created_at
		FROM categories
		WHERE is_active = TRUE
		ORDER BY name ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar categorias: %w", err)
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.IconURL, &c.Description, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (r *categoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	query := `
		SELECT id, name, COALESCE(icon_url, ''), COALESCE(description, ''), is_active, created_at
		FROM categories
		WHERE id = $1
	`
	var c domain.Category
	err := r.pool.QueryRow(ctx, query, id).Scan(&c.ID, &c.Name, &c.IconURL, &c.Description, &c.IsActive, &c.CreatedAt)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return &c, nil
}

func (r *categoryRepository) SearchCategories(ctx context.Context, q string) ([]domain.Category, error) {
	query := `
		SELECT id, name, COALESCE(icon_url, ''), COALESCE(description, ''), is_active, created_at
		FROM categories
		WHERE is_active = TRUE
		  AND (name ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')
		ORDER BY name ASC
		LIMIT 10
	`
	rows, err := r.pool.Query(ctx, query, q)
	if err != nil {
		return nil, fmt.Errorf("erro ao pesquisar categorias: %w", err)
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.IconURL, &c.Description, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}
