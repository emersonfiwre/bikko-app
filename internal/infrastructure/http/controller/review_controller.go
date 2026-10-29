package controller

import (
	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/http/middleware"
	"bikko-app/internal/usecase"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type ReviewController struct{ ratingUC *usecase.RatingUseCase }

func NewReviewController(uc *usecase.RatingUseCase) *ReviewController {
	return &ReviewController{ratingUC: uc}
}

type CreateReviewRequest struct {
	OrderID    string   `json:"order_id"`
	ReviewerID string   `json:"reviewer_id"`
	Rating     int      `json:"rating" binding:"required,min=1,max=5"`
	Comment    string   `json:"comment"`
	Tags       []string `json:"tags"`
}

func (c *ReviewController) CreateReview(ctx *gin.Context)             { c.handleReview(ctx, false) }
func (c *ReviewController) CreateSolicitationReview(ctx *gin.Context) { c.handleReview(ctx, true) }
func (c *ReviewController) CreateOrderReview(ctx *gin.Context)        { c.handleReview(ctx, true) }
func (c *ReviewController) handleReview(ctx *gin.Context, pathID bool) {
	actor := middleware.GetUserID(ctx)
	if actor == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "UNAUTHORIZED"})
		return
	}
	requestKey, ok := requiredIdempotencyKey(ctx)
	if !ok {
		return
	}
	var req CreateReviewRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		contractError(ctx, domain.ErrInvalidInput)
		return
	}
	if req.ReviewerID != "" && req.ReviewerID != actor {
		contractError(ctx, domain.ErrForbidden)
		return
	}
	id := req.OrderID
	if pathID {
		id = ctx.Param("id")
		if req.OrderID != "" && req.OrderID != id {
			contractError(ctx, domain.ErrInvalidInput)
			return
		}
	}
	if id == "" || len(req.Tags) > 10 || len([]rune(req.Comment)) > 1000 {
		contractError(ctx, domain.ErrInvalidInput)
		return
	}
	comment := strings.TrimSpace(req.Comment)
	if len(req.Tags) > 0 {
		comment = strings.TrimSpace("[" + strings.Join(req.Tags, ", ") + "] " + comment)
	}
	review, err := c.ratingUC.CreateReview(ctx.Request.Context(), domain.CreateReviewInput{OrderID: id, ReviewerID: actor, Rating: req.Rating, Comment: comment, RequestKey: requestKey})
	if err != nil {
		contractError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"message": "Avaliação registrada com sucesso", "review": review})
}
