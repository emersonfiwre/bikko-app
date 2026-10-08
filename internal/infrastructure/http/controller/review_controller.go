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

type CreateReviewRequest struct {
	OrderID    string `json:"order_id"`
	ReviewerID string `json:"reviewer_id"`
	Rating     int    `json:"rating" binding:"required,min=1,max=5"`
	Comment    string `json:"comment"`
}

func (ctrl *ReviewController) CreateReview(c *gin.Context) {
	var input domain.CreateReviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctrl.processReview(c, input)
}

func (ctrl *ReviewController) CreateSolicitationReview(c *gin.Context) {
	ctrl.handleTargetReview(c)
}

func (ctrl *ReviewController) CreateOrderReview(c *gin.Context) {
	ctrl.handleTargetReview(c)
}

func (ctrl *ReviewController) handleTargetReview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id é obrigatório"})
		return
	}

	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID := req.OrderID
	if orderID == "" {
		orderID = id
	}

	reviewerID := req.ReviewerID
	if reviewerID == "" {
		if uid, exists := c.Get("userID"); exists && uid != nil {
			reviewerID = uid.(string)
		}
	}
	if reviewerID == "" {
		reviewerID = "11111111-0000-0000-0000-000000000099"
	}

	input := domain.CreateReviewInput{
		OrderID:    orderID,
		ReviewerID: reviewerID,
		Rating:     req.Rating,
		Comment:    req.Comment,
	}

	ctrl.processReview(c, input)
}

func (ctrl *ReviewController) processReview(c *gin.Context, input domain.CreateReviewInput) {
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
