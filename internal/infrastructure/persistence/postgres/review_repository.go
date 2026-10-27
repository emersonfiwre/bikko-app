package postgres

import (
	"bikko-app/internal/domain"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type reviewRepository struct{ pool *pgxpool.Pool }

func NewReviewRepository(pool *pgxpool.Pool) domain.ReviewRepository {
	return &reviewRepository{pool: pool}
}
func (r *reviewRepository) CreateReviewInTx(ctx context.Context, review *domain.Review) error {
	if r.pool == nil {
		return domain.ErrUnavailable
	}
	if review == nil || !isValidUUID(review.OrderID) || !isValidUUID(review.ReviewerID) || review.Rating < 1 || review.Rating > 5 || len([]rune(review.Comment)) > 1000 {
		return domain.ErrInvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q, err := lockContract(ctx, tx, review.OrderID, review.ReviewerID)
	if err != nil {
		return err
	}
	if q.client != review.ReviewerID || q.provider == review.ReviewerID {
		return domain.ErrForbidden
	}
	if q.orderStatus == nil || *q.orderStatus != "COMPLETED" {
		return domain.ErrOrderNotCompleted
	}
	var orderID string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM orders WHERE quote_request_id=$1::uuid`, q.id).Scan(&orderID); err != nil {
		return err
	}
	// Serialize reputation changes across orders for the same professional.
	var userID string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM users WHERE id=$1::uuid FOR UPDATE`, q.provider).Scan(&userID); err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `INSERT INTO reviews(order_id,service_id,reviewer_id,bikker_id,rating,comment) VALUES($1::uuid,$2,$3::uuid,$4::uuid,$5,$6) ON CONFLICT(order_id,reviewer_id) DO NOTHING RETURNING id::text,created_at`, orderID, q.service, q.client, q.provider, review.Rating, review.Comment).Scan(&review.ID, &review.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrAlreadyReviewed
	}
	if err != nil {
		return fmt.Errorf("insert review: %w", err)
	}
	review.OrderID = orderID
	review.BikkerID = q.provider
	if q.service != nil {
		review.ServiceID = *q.service
		_, err = tx.Exec(ctx, `UPDATE services SET reviews_average=(SELECT ROUND(AVG(rating)::numeric,2) FROM reviews WHERE service_id=$1::uuid),total_reviews=(SELECT COUNT(*) FROM reviews WHERE service_id=$1::uuid),updated_at=NOW() WHERE id=$1::uuid`, *q.service)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE bikkers SET rating=(SELECT ROUND(AVG(rating)::numeric,2) FROM reviews WHERE bikker_id=$1::uuid),total_reviews=(SELECT COUNT(*) FROM reviews WHERE bikker_id=$1::uuid),updated_at=NOW() WHERE id=$1::uuid`, q.provider)
	if err != nil {
		return err
	}
	if err = recomputeUserRating(ctx, tx, q.provider); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE quote_requests SET client_evaluated=TRUE,client_rating_given=$2,updated_at=NOW() WHERE id=$1::uuid`, q.id, review.Rating)
	if err != nil {
		return err
	}
	if err = enqueueContractNotification(ctx, tx, q.provider, q.id, "Nova avaliação", "O cliente avaliou o serviço.", "review_received"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func recomputeUserRating(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `UPDATE users SET rating=COALESCE((SELECT ROUND(AVG(rating)::numeric,2) FROM (SELECT rating FROM reviews WHERE bikker_id=$1::uuid UNION ALL SELECT rating FROM client_reviews WHERE client_id=$1::uuid) all_reviews),0),total_ratings=(SELECT COUNT(*) FROM (SELECT id FROM reviews WHERE bikker_id=$1::uuid UNION ALL SELECT id FROM client_reviews WHERE client_id=$1::uuid) all_reviews),updated_at=NOW() WHERE id=$1::uuid`, userID)
	return err
}
