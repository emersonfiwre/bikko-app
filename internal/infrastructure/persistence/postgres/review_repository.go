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
	"time"
)

type reviewRepository struct{ pool *pgxpool.Pool }

func NewReviewRepository(pool *pgxpool.Pool) domain.ReviewRepository {
	return &reviewRepository{pool: pool}
}
func (r *reviewRepository) CreateReviewInTx(ctx context.Context, review *domain.Review) error {
	if r.pool == nil {
		return domain.ErrUnavailable
	}
	if review == nil || !isValidUUID(review.OrderID) || !isValidUUID(review.ReviewerID) || review.Rating < 1 || review.Rating > 5 || len([]rune(review.Comment)) > 1000 || len(review.RequestKey) > 128 {
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
	var fingerprint string
	if review.RequestKey != "" {
		fingerprint, err = reviewFingerprint(orderID, review.Rating, review.Comment)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, review.ReviewerID+":"+review.RequestKey); err != nil {
			return err
		}
		var previous struct {
			id, orderID, bikkerID, serviceID, fingerprint, comment string
			rating                                                 int
			createdAt                                              time.Time
		}
		err = tx.QueryRow(ctx, `SELECT id::text,order_id::text,bikker_id::text,COALESCE(service_id::text,''),COALESCE(request_fingerprint,''),rating,COALESCE(comment,''),created_at
			FROM reviews WHERE reviewer_id=$1::uuid AND request_key=$2 FOR UPDATE`, review.ReviewerID, review.RequestKey).
			Scan(&previous.id, &previous.orderID, &previous.bikkerID, &previous.serviceID, &previous.fingerprint, &previous.rating, &previous.comment, &previous.createdAt)
		if err == nil {
			if previous.orderID != orderID || previous.fingerprint != fingerprint {
				return domain.ErrConflict
			}
			review.ID = previous.id
			review.OrderID = previous.orderID
			review.BikkerID = previous.bikkerID
			review.ServiceID = previous.serviceID
			review.Rating = previous.rating
			review.Comment = previous.comment
			review.CreatedAt = previous.createdAt
			return tx.Commit(ctx)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	// Serialize reputation changes across orders for the same professional.
	var userID string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM users WHERE id=$1::uuid FOR UPDATE`, q.provider).Scan(&userID); err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `INSERT INTO reviews(order_id,service_id,reviewer_id,bikker_id,rating,comment,request_key,request_fingerprint) VALUES($1::uuid,$2,$3::uuid,$4::uuid,$5,$6,NULLIF($7,''),NULLIF($8,'')) ON CONFLICT(order_id,reviewer_id) DO NOTHING RETURNING id::text,created_at`, orderID, q.service, q.client, q.provider, review.Rating, review.Comment, review.RequestKey, fingerprint).Scan(&review.ID, &review.CreatedAt)
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

func reviewFingerprint(orderID string, rating int, comment string) (string, error) {
	payload, err := json.Marshal(struct {
		OrderID string `json:"order_id"`
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}{OrderID: orderID, Rating: rating, Comment: comment})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(payload)), nil
}
func recomputeUserRating(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `UPDATE users SET rating=COALESCE((SELECT ROUND(AVG(rating)::numeric,2) FROM (SELECT rating FROM reviews WHERE bikker_id=$1::uuid UNION ALL SELECT rating FROM client_reviews WHERE client_id=$1::uuid) all_reviews),0),total_ratings=(SELECT COUNT(*) FROM (SELECT id FROM reviews WHERE bikker_id=$1::uuid UNION ALL SELECT id FROM client_reviews WHERE client_id=$1::uuid) all_reviews),updated_at=NOW() WHERE id=$1::uuid`, userID)
	return err
}
