package controller

import (
	"errors"
	"net/http"

	"bikko-app/internal/domain"
	"bikko-app/internal/usecase"

	"github.com/gin-gonic/gin"
)

type ReviewController struct {
	ratingUC *usecase.RatingUseCase
}

func NewReviewController(ratingUC *usecase.RatingUseCase) *ReviewController {
	return &ReviewController{ratingUC: ratingUC}
}

func (ctrl *ReviewController) CreateReview(c *gin.Context) {
	var input domain.CreateReviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	review, err := ctrl.ratingUC.CreateReview(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotCompleted) || errors.Is(err, domain.ErrAlreadyReviewed) || errors.Is(err, domain.ErrInvalidRating) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ordem não encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Avaliação registrada com sucesso",
		"review":  review,
	})
}
