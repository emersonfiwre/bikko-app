package controller

import (
	"errors"
	"net/http"
	"strings"

	"bikko-app/internal/domain"
	"bikko-app/internal/model"
	"bikko-app/internal/usecase"

	"github.com/gin-gonic/gin"
)

type ReviewController struct {
	ratingUC *usecase.RatingUseCase
	store    model.SolicitationStore
}

func NewReviewController(ratingUC *usecase.RatingUseCase) *ReviewController {
	return &ReviewController{ratingUC: ratingUC}
}

func (ctrl *ReviewController) SetSolicitationStore(store model.SolicitationStore) {
	ctrl.store = store
}

type CreateReviewRequest struct {
	OrderID    string   `json:"order_id"`
	ReviewerID string   `json:"reviewer_id"`
	Rating     int      `json:"rating" binding:"required,min=1,max=5"`
	Comment    string   `json:"comment"`
	Tags       []string `json:"tags"`
}

func (ctrl *ReviewController) CreateReview(c *gin.Context) {
	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.OrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id é obrigatório"})
		return
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

	comment := strings.TrimSpace(req.Comment)
	if len(req.Tags) > 0 {
		tagStr := "[" + strings.Join(req.Tags, ", ") + "]"
		if comment != "" && !strings.HasPrefix(comment, "[") {
			comment = tagStr + " " + comment
		} else if comment == "" {
			comment = tagStr
		}
	}

	input := domain.CreateReviewInput{
		OrderID:    req.OrderID,
		ReviewerID: reviewerID,
		Rating:     req.Rating,
		Comment:    comment,
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

	comment := strings.TrimSpace(req.Comment)
	if len(req.Tags) > 0 {
		tagStr := "[" + strings.Join(req.Tags, ", ") + "]"
		if comment != "" && !strings.HasPrefix(comment, "[") {
			comment = tagStr + " " + comment
		} else if comment == "" {
			comment = tagStr
		}
	}

	input := domain.CreateReviewInput{
		OrderID:    orderID,
		ReviewerID: reviewerID,
		Rating:     req.Rating,
		Comment:    comment,
	}

	ctrl.processReview(c, input)
}

func (ctrl *ReviewController) processReview(c *gin.Context, input domain.CreateReviewInput) {
	review, err := ctrl.ratingUC.CreateReview(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyReviewed) {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "ALREADY_REVIEWED",
				"error":   "ALREADY_REVIEWED",
				"message": "Este serviço já foi avaliado anteriormente.",
			})
			return
		}
		if errors.Is(err, domain.ErrOrderNotCompleted) || errors.Is(err, domain.ErrInvalidRating) {
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

	if ctrl.store != nil {
		if inMem := ctrl.store.GetInMemorySolicitationByID(input.OrderID); inMem != nil {
			inMem.Status = "COMPLETED"
			inMem.IsClientEvaluated = true
			inMem.ClientEvaluated = true
			inMem.ClientRatingGiven = input.Rating
			ctrl.store.SyncSolicitationInMemory(*inMem)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Avaliação registrada com sucesso",
		"review":  review,
	})
}
