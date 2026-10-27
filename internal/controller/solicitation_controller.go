package controller

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/http/middleware"
	"bikko-app/internal/model"
	"bikko-app/internal/usecase"

	"github.com/gin-gonic/gin"
)

type SolicitationController struct {
	uc          *usecase.SolicitationUseCase
	repo        model.SolicitationRepository
	serviceRepo domain.ServiceRepository
}

func NewSolicitationController(services domain.ServiceRepository) *SolicitationController {
	return &SolicitationController{serviceRepo: services}
}

func NewSolicitationControllerWithUseCase(uc *usecase.SolicitationUseCase) *SolicitationController {
	return &SolicitationController{uc: uc}
}

func (c *SolicitationController) SetPostgresRepository(repo model.SolicitationRepository) {
	c.repo = repo
	c.uc = usecase.NewSolicitationUseCase(repo, c.serviceRepo)
}

func solicitationError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	switch {
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
		code = "FORBIDDEN"
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		code = "NOT_FOUND"
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrOrderNotCompleted):
		status = http.StatusConflict
		code = "INVALID_STATE"
	case errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
		code = "INVALID_INPUT"
	case errors.Is(err, domain.ErrUnavailable):
		status = http.StatusServiceUnavailable
		code = "DEPENDENCY_UNAVAILABLE"
	}
	if status >= 500 {
		slog.Error("solicitation request failed", slog.Any("error", err))
	}
	c.JSON(status, gin.H{"error": code, "code": code})
}

func (c *SolicitationController) actor(ctx *gin.Context) (string, bool) {
	id := middleware.GetUserID(ctx)
	if id == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "UNAUTHORIZED"})
		return "", false
	}
	if c.uc == nil && c.repo == nil {
		solicitationError(ctx, domain.ErrUnavailable)
		return "", false
	}
	return id, true
}

func (c *SolicitationController) GetSolicitations(ctx *gin.Context) {
	actor, ok := c.actor(ctx)
	if !ok {
		return
	}
	limit := 50
	offset := 0
	var err error
	if raw := ctx.Query("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			solicitationError(ctx, domain.ErrInvalidInput)
			return
		}
	}
	if raw := ctx.Query("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil {
			solicitationError(ctx, domain.ErrInvalidInput)
			return
		}
	}

	var items []model.SolicitationItem
	if c.uc != nil {
		items, err = c.uc.GetSolicitations(ctx.Request.Context(), actor, limit, offset)
	} else if pager, ok := c.repo.(interface {
		GetSolicitationsPage(context.Context, string, int, int) ([]model.SolicitationItem, error)
	}); ok {
		items, err = pager.GetSolicitationsPage(ctx.Request.Context(), actor, limit, offset)
	} else {
		items, err = c.repo.GetSolicitationsByUser(ctx.Request.Context(), actor)
	}
	if err != nil {
		solicitationError(ctx, err)
		return
	}
	if items == nil {
		items = []model.SolicitationItem{}
	}
	ctx.JSON(http.StatusOK, items)
}

func (c *SolicitationController) GetSolicitationByID(ctx *gin.Context) {
	actor, ok := c.actor(ctx)
	if !ok {
		return
	}
	var item *model.SolicitationItem
	var err error
	if c.uc != nil {
		item, err = c.uc.GetSolicitationByID(ctx.Request.Context(), ctx.Param("id"), actor)
	} else {
		item, err = c.repo.GetSolicitationForUser(ctx.Request.Context(), ctx.Param("id"), actor)
	}
	if err != nil {
		solicitationError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *SolicitationController) CreateSolicitation(ctx *gin.Context) {
	actor, ok := c.actor(ctx)
	if !ok {
		return
	}
	var req model.CreateSolicitationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		solicitationError(ctx, domain.ErrInvalidInput)
		return
	}

	var item *model.SolicitationItem
	var err error
	if c.uc != nil {
		item, err = c.uc.CreateSolicitation(ctx.Request.Context(), actor, ctx.GetHeader("Idempotency-Key"), req)
	} else {
		// Fallback for direct repo usage in tests
		item, err = c.repo.CreateSolicitation(ctx.Request.Context(), model.SolicitationItem{
			RequestKey:      ctx.GetHeader("Idempotency-Key"),
			ClientID:        actor,
			ProviderID:      req.ProviderID,
			ServiceID:       req.ServiceID,
			ServiceName:     req.ServiceName,
			Description:     req.Description,
			PreferredDate:   req.PreferredDate,
			LocationAddress: req.LocationAddress,
		})
	}
	if err != nil {
		solicitationError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, item)
}

func (c *SolicitationController) UpdateStatus(ctx *gin.Context) {
	actor, ok := c.actor(ctx)
	if !ok {
		return
	}
	var req struct {
		Status              string   `json:"status"`
		TechnicalReport     *string  `json:"technical_report"`
		CancellationReason  *string  `json:"cancellation_reason"`
		CounterOfferPrice   *float64 `json:"counter_offer_price"`
		CounterOfferMessage *string  `json:"counter_offer_message"`
		Photos              []string `json:"photos"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.Status == "" {
		solicitationError(ctx, domain.ErrInvalidInput)
		return
	}

	in := model.SolicitationStatusUpdateInput{
		ActorID:             actor,
		RequestKey:          ctx.GetHeader("Idempotency-Key"),
		Status:              req.Status,
		TechnicalReport:     req.TechnicalReport,
		CancellationReason:  req.CancellationReason,
		CounterOfferPrice:   req.CounterOfferPrice,
		CounterOfferMessage: req.CounterOfferMessage,
		Photos:              req.Photos,
	}

	var item *model.SolicitationItem
	var err error
	if c.uc != nil {
		item, err = c.uc.UpdateStatus(ctx.Request.Context(), in, ctx.Param("id"))
	} else {
		item, err = c.repo.UpdateSolicitationStatus(ctx.Request.Context(), ctx.Param("id"), in)
	}
	if err != nil {
		solicitationError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *SolicitationController) GetSolicitationsByUser(ctx context.Context, actor string) ([]model.SolicitationItem, error) {
	if c.uc != nil {
		return c.uc.GetSolicitations(ctx, actor, 50, 0)
	}
	if c.repo == nil {
		return nil, domain.ErrUnavailable
	}
	return c.repo.GetSolicitationsByUser(ctx, actor)
}
