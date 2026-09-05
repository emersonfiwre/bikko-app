package usecase

import (
	"context"

	"bikko-app/internal/domain"
)

type RatingUseCase struct {
	reviewRepo domain.ReviewRepository
}

func NewRatingUseCase(reviewRepo domain.ReviewRepository) *RatingUseCase {
	return &RatingUseCase{reviewRepo: reviewRepo}
}

func (uc *RatingUseCase) CreateReview(ctx context.Context, input domain.CreateReviewInput) (*domain.Review, error) {
	if input.Rating < 1 || input.Rating > 5 {
		return nil, domain.ErrInvalidRating
	}

	review := &domain.Review{
		OrderID:    input.OrderID,
		ReviewerID: input.ReviewerID,
		Rating:     input.Rating,
		Comment:    input.Comment,
	}

	if err := uc.reviewRepo.CreateReviewInTx(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}
