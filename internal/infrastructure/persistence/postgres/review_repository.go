package postgres

import (
	"context"
	"fmt"

	"bikko-app/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type reviewRepository struct {
	pool *pgxpool.Pool
}

func NewReviewRepository(pool *pgxpool.Pool) domain.ReviewRepository {
	return &reviewRepository{pool: pool}
}

func (r *reviewRepository) CreateReviewInTx(ctx context.Context, review *domain.Review) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("falha ao iniciar transação: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Verify Order status is COMPLETED and fetch service_id and bikker_id
	var orderStatus string
	var serviceID string
	var bikkerID string
	orderQuery := `SELECT status, service_id, bikker_id FROM orders WHERE id = $1`
	err = tx.QueryRow(ctx, orderQuery, review.OrderID).Scan(&orderStatus, &serviceID, &bikkerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ErrNotFound
		}
		return fmt.Errorf("erro ao verificar ordem: %w", err)
	}

	if orderStatus != string(domain.OrderStatusCompleted) {
		return domain.ErrOrderNotCompleted
	}

	review.ServiceID = serviceID
	review.BikkerID = bikkerID

	// Check if already reviewed (UNIQUE constraint on order_id)
	var exists bool
	checkReviewQuery := `SELECT EXISTS(SELECT 1 FROM reviews WHERE order_id = $1)`
	if err := tx.QueryRow(ctx, checkReviewQuery, review.OrderID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return domain.ErrAlreadyReviewed
	}

	// 2. Insert Review
	insertReviewQuery := `
		INSERT INTO reviews (order_id, service_id, reviewer_id, bikker_id, rating, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err = tx.QueryRow(ctx, insertReviewQuery,
		review.OrderID, review.ServiceID, review.ReviewerID, review.BikkerID, review.Rating, review.Comment,
	).Scan(&review.ID, &review.CreatedAt)
	if err != nil {
		return fmt.Errorf("erro ao inserir avaliação: %w", err)
	}

	// 3. Update SERVICES rating & count incrementally
	var currentAvg float64
	var currentTotal int
	getServiceQuery := `SELECT reviews_average, total_reviews FROM services WHERE id = $1 FOR UPDATE`
	if err := tx.QueryRow(ctx, getServiceQuery, serviceID).Scan(&currentAvg, &currentTotal); err != nil {
		return fmt.Errorf("erro ao buscar métricas do serviço: %w", err)
	}

	newTotalService := currentTotal + 1
	newAvgService := ((currentAvg * float64(currentTotal)) + float64(review.Rating)) / float64(newTotalService)

	updateServiceQuery := `
		UPDATE services 
		SET reviews_average = $1, total_reviews = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	if _, err := tx.Exec(ctx, updateServiceQuery, newAvgService, newTotalService, serviceID); err != nil {
		return fmt.Errorf("erro ao atualizar serviço: %w", err)
	}

	// 4. Recalculate BIKKERS rating as arithmetic mean of reviews_average of all active services
	recalcBikkerQuery := `
		SELECT COALESCE(AVG(reviews_average), 0.0), COALESCE(SUM(total_reviews), 0)
		FROM services
		WHERE bikker_id = $1 AND is_active = TRUE
	`
	var newBikkerRating float64
	var newBikkerTotalReviews int
	if err := tx.QueryRow(ctx, recalcBikkerQuery, bikkerID).Scan(&newBikkerRating, &newBikkerTotalReviews); err != nil {
		return fmt.Errorf("erro ao recalcular métricas do bikker: %w", err)
	}

	updateBikkerQuery := `
		UPDATE bikkers
		SET rating = $1, total_reviews = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	if _, err := tx.Exec(ctx, updateBikkerQuery, newBikkerRating, newBikkerTotalReviews, bikkerID); err != nil {
		return fmt.Errorf("erro ao atualizar bikker: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("erro ao comitar transação de avaliação: %w", err)
	}

	return nil
}
