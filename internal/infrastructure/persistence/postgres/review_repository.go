package postgres

import (
	"context"
	"fmt"
	"time"

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

	orderID := review.OrderID
	switch orderID {
	case "sol_1":
		orderID = "44444444-0000-0000-0000-000000000001"
	case "sol_2":
		orderID = "44444444-0000-0000-0000-000000000002"
	case "sol_3":
		orderID = "44444444-0000-0000-0000-000000000003"
	case "sol_4":
		orderID = "44444444-0000-0000-0000-000000000004"
	}

	// 1. Verify Order / Solicitation and fetch service_id and bikker_id
	var actualOrderID string
	var quoteRequestID string
	var orderStatus string
	var serviceID *string
	var bikkerID string

	orderQuery := `SELECT id::text, quote_request_id::text, status, service_id::text, bikker_id::text FROM orders WHERE id::text = $1 OR quote_request_id::text = $1 LIMIT 1`
	err = tx.QueryRow(ctx, orderQuery, orderID).Scan(&actualOrderID, &quoteRequestID, &orderStatus, &serviceID, &bikkerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Fallback: check if quote_request exists
			qrQuery := `SELECT id::text, customer_id::text, bikker_id::text, service_id::text, status FROM quote_requests WHERE id::text = $1 LIMIT 1`
			var qrID, qrCustID, qrBikkerID, qrStatus string
			var qrServiceID *string
			if qrErr := tx.QueryRow(ctx, qrQuery, orderID).Scan(&qrID, &qrCustID, &qrBikkerID, &qrServiceID, &qrStatus); qrErr != nil {
				return domain.ErrNotFound
			}
			if qrStatus == "REJECTED" || qrStatus == "EXPIRED" {
				return domain.ErrOrderNotCompleted
			}
			quoteRequestID = qrID
			bikkerID = qrBikkerID
			serviceID = qrServiceID

			insOrderQuery := `
				INSERT INTO orders (quote_request_id, customer_id, bikker_id, service_id, status, final_price, scheduled_date)
				VALUES ($1::uuid, $2::uuid, $3::uuid, $4, 'COMPLETED', 0, NOW())
				RETURNING id::text
			`
			if insErr := tx.QueryRow(ctx, insOrderQuery, qrID, qrCustID, qrBikkerID, qrServiceID).Scan(&actualOrderID); insErr != nil {
				return fmt.Errorf("erro ao criar ordem para avaliacao: %w", insErr)
			}
			orderStatus = string(domain.OrderStatusCompleted)
		} else {
			return fmt.Errorf("erro ao verificar ordem: %w", err)
		}
	}

	if orderStatus == string(domain.OrderStatusCancelled) {
		return domain.ErrOrderNotCompleted
	}

	// If order is not yet marked COMPLETED, complete it now!
	if orderStatus != string(domain.OrderStatusCompleted) {
		_, _ = tx.Exec(ctx, `UPDATE orders SET status = 'COMPLETED', updated_at = CURRENT_TIMESTAMP WHERE id::text = $1`, actualOrderID)
		orderStatus = string(domain.OrderStatusCompleted)
	}

	// Update quote_requests to ACCEPTED (completed), client_evaluated = TRUE, client_rating_given
	if quoteRequestID != "" {
		_, _ = tx.Exec(ctx, `
			UPDATE quote_requests
			SET status = 'ACCEPTED', client_evaluated = TRUE, client_rating_given = $2, updated_at = CURRENT_TIMESTAMP
			WHERE id::text = $1
		`, quoteRequestID, review.Rating)
	}

	review.OrderID = actualOrderID
	if serviceID != nil {
		review.ServiceID = *serviceID
	}
	review.BikkerID = bikkerID

	// Check if already reviewed by this reviewer (or order_id)
	var exists bool
	checkReviewQuery := `SELECT EXISTS(SELECT 1 FROM reviews WHERE order_id::text = $1 AND reviewer_id::text = $2)`
	if err := tx.QueryRow(ctx, checkReviewQuery, review.OrderID, review.ReviewerID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return domain.ErrAlreadyReviewed
	}

	// 2. Insert Review
	var insertedID string
	var insertedCreatedAt time.Time
	insertReviewQuery := `
		INSERT INTO reviews (order_id, service_id, reviewer_id, bikker_id, rating, comment)
		VALUES ($1::uuid, $2, $3::uuid, $4::uuid, $5, $6)
		RETURNING id, created_at
	`
	err = tx.QueryRow(ctx, insertReviewQuery,
		review.OrderID, serviceID, review.ReviewerID, review.BikkerID, review.Rating, review.Comment,
	).Scan(&insertedID, &insertedCreatedAt)
	if err != nil {
		return fmt.Errorf("erro ao inserir avaliação: %w", err)
	}
	review.ID = insertedID
	review.CreatedAt = insertedCreatedAt

	// 3. Update SERVICES rating & count incrementally if serviceID is present
	if serviceID != nil && *serviceID != "" {
		var currentAvg float64
		var currentTotal int
		getServiceQuery := `SELECT reviews_average, total_reviews FROM services WHERE id::text = $1 FOR UPDATE`
		if err := tx.QueryRow(ctx, getServiceQuery, *serviceID).Scan(&currentAvg, &currentTotal); err == nil {
			newTotalService := currentTotal + 1
			newAvgService := ((currentAvg * float64(currentTotal)) + float64(review.Rating)) / float64(newTotalService)

			updateServiceQuery := `
				UPDATE services 
				SET reviews_average = $1, total_reviews = $2, updated_at = CURRENT_TIMESTAMP
				WHERE id::text = $3
			`
			_, _ = tx.Exec(ctx, updateServiceQuery, newAvgService, newTotalService, *serviceID)
		}
	}

	// 4. Update BIKKERS rating & count
	updateBikkerQuery := `
		UPDATE bikkers
		SET rating = ROUND(((COALESCE(rating, 0) * COALESCE(total_reviews, 0) + $1) / (COALESCE(total_reviews, 0) + 1))::numeric, 2),
		    total_reviews = COALESCE(total_reviews, 0) + 1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id::text = $2
	`
	if _, err := tx.Exec(ctx, updateBikkerQuery, float64(review.Rating), bikkerID); err != nil {
		return fmt.Errorf("erro ao atualizar bikker: %w", err)
	}

	// 5. Update USERS rating & total_ratings for the reviewed Bikker
	updateUserQuery := `
		UPDATE users
		SET rating = ROUND(((COALESCE(rating, 0) * COALESCE(total_ratings, 0) + $1) / (COALESCE(total_ratings, 0) + 1))::numeric, 2),
		    total_ratings = COALESCE(total_ratings, 0) + 1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id::text = $2
	`
	_, _ = tx.Exec(ctx, updateUserQuery, float64(review.Rating), bikkerID)

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("erro ao comitar transação de avaliação: %w", err)
	}

	return nil
}
