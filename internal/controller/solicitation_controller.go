package controller

import (
	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/http/middleware"
	"bikko-app/internal/model"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type SolicitationController struct {
	repo        model.SolicitationRepository
	serviceRepo domain.ServiceRepository
}

func NewSolicitationController(services domain.ServiceRepository) *SolicitationController {
	return &SolicitationController{serviceRepo: services}
}
func (c *SolicitationController) SetPostgresRepository(repo model.SolicitationRepository) {
	c.repo = repo
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
	if c.repo == nil {
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
	var items []model.SolicitationItem
	var err error
	if pager, ok := c.repo.(interface {
		GetSolicitationsPage(context.Context, string, int, int) ([]model.SolicitationItem, error)
	}); ok {
		limit := 50
		offset := 0
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
	item, err := c.repo.GetSolicitationForUser(ctx.Request.Context(), ctx.Param("id"), actor)
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
	provider := strings.TrimSpace(req.ProviderID)
	if provider == "" {
		provider = strings.TrimSpace(req.BikkerID)
	}
	name := strings.TrimSpace(req.ServiceTitle)
	if name == "" {
		name = strings.TrimSpace(req.Title)
	}
	if name == "" {
		name = strings.TrimSpace(req.ServiceName)
	}
	if req.ServiceID != "" && c.serviceRepo != nil {
		service, err := c.serviceRepo.GetServiceByID(ctx.Request.Context(), req.ServiceID)
		if err != nil {
			solicitationError(ctx, err)
			return
		}
		if provider != "" && provider != service.BikkerID {
			solicitationError(ctx, domain.ErrInvalidInput)
			return
		}
		provider = service.BikkerID
		name = service.Name
	}
	address := strings.TrimSpace(req.LocationAddress)
	if address == "" {
		address = strings.TrimSpace(req.Address)
	}
	if address == "" {
		address = strings.TrimSpace(req.Location)
		if req.HouseNumber != "" {
			address += ", " + strings.TrimSpace(req.HouseNumber)
		}
	}
	preferred := strings.TrimSpace(req.PreferredDate)
	if preferred == "" {
		preferred = strings.TrimSpace(req.ScheduledDate)
	}
	photos := req.ClientPhotos
	if len(photos) == 0 {
		photos = req.Photos
	}
	if photos == nil {
		photos = []string{}
	}
	item, err := c.repo.CreateSolicitation(ctx.Request.Context(), model.SolicitationItem{RequestKey: ctx.GetHeader("Idempotency-Key"), ClientID: actor, ProviderID: provider, ServiceID: strings.TrimSpace(req.ServiceID), ServiceName: name, ServiceTitle: name, CategoryName: strings.TrimSpace(req.CategoryName), Description: strings.TrimSpace(req.Description), PreferredDate: preferred, ScheduledDate: preferred, LocationAddress: address, Address: address, ContactName: strings.TrimSpace(req.ContactName), ContactPhone: strings.TrimSpace(req.ContactPhone), Latitude: req.Latitude, Longitude: req.Longitude, Photos: photos, ClientPhotos: photos})
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
	item, err := c.repo.UpdateSolicitationStatus(ctx.Request.Context(), ctx.Param("id"), model.SolicitationStatusUpdateInput{ActorID: actor, RequestKey: ctx.GetHeader("Idempotency-Key"), Status: req.Status, TechnicalReport: req.TechnicalReport, CancellationReason: req.CancellationReason, CounterOfferPrice: req.CounterOfferPrice, CounterOfferMessage: req.CounterOfferMessage, Photos: req.Photos})
	if err != nil {
		solicitationError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, item)
}
func (c *SolicitationController) GetSolicitationsByUser(ctx context.Context, actor string) ([]model.SolicitationItem, error) {
	if c.repo == nil {
		return nil, domain.ErrUnavailable
	}
	return c.repo.GetSolicitationsByUser(ctx, actor)
}
