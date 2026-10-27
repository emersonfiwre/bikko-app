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
	"strings"
)

type ProviderController struct {
	solRepo     model.SolicitationRepository
	serviceRepo domain.ServiceRepository
	bikkerRepo  domain.BikkerRepository
}

func NewProviderController() *ProviderController { return &ProviderController{} }
func NewProviderControllerWithDeps(repo model.SolicitationRepository, services domain.ServiceRepository, bikkers domain.BikkerRepository) *ProviderController {
	return &ProviderController{solRepo: repo, serviceRepo: services, bikkerRepo: bikkers}
}
func contractError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	switch {
	case errors.Is(err, domain.ErrForbidden):
		status = 403
		code = "FORBIDDEN"
	case errors.Is(err, domain.ErrNotFound):
		status = 404
		code = "NOT_FOUND"
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrOrderNotCompleted):
		status = 409
		code = "INVALID_STATE"
	case errors.Is(err, domain.ErrAlreadyReviewed):
		status = 409
		code = "ALREADY_REVIEWED"
	case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrInvalidRating):
		status = 400
		code = "INVALID_INPUT"
	case errors.Is(err, domain.ErrUnavailable):
		status = 503
		code = "DEPENDENCY_UNAVAILABLE"
	}
	if status >= 500 {
		slog.Error("provider request failed", slog.Any("error", err))
	}
	c.JSON(status, gin.H{"error": code, "code": code})
}
func (p *ProviderController) actor(c *gin.Context) (string, bool) {
	actor := middleware.GetUserID(c)
	if actor == "" {
		c.JSON(401, gin.H{"error": "UNAUTHORIZED"})
		return "", false
	}
	if !c.GetBool("isBikker") {
		contractError(c, domain.ErrForbidden)
		return "", false
	}
	return actor, true
}
func (p *ProviderController) repository(c *gin.Context) bool {
	if p.solRepo == nil {
		contractError(c, domain.ErrUnavailable)
		return false
	}
	return true
}
func (p *ProviderController) GetFeed(c *gin.Context) {
	actor, ok := p.actor(c)
	if !ok || !p.repository(c) {
		return
	}
	items, err := p.solRepo.GetProviderFeed(c.Request.Context(), actor)
	if err != nil {
		contractError(c, err)
		return
	}
	if items == nil {
		items = []domain.OpportunityLead{}
	}
	c.JSON(200, items)
}
func buildQuoteDetails(s *model.SolicitationItem) gin.H {
	accepted := s.Status == "ACTIVE" || s.Status == "COMPLETED"
	hasPrice := s.Price > 0
	lead := domain.OpportunityLead{ID: s.ID, Title: s.ServiceName, ServiceTitle: s.ServiceName, Description: s.Description, Category: s.CategoryName, Urgency: "SCHEDULED", Photos: s.Photos, ClientPhotos: s.ClientPhotos, PriceType: "BUDGET_TO_NEGOTIATE", DistanceKm: s.DistanceKm, Neighborhood: s.Neighborhood, Address: s.Address, LocationAddress: s.LocationAddress, ClientName: s.ClientName, ContactName: s.ContactName, ClientPhone: s.AccountPhone, ContactPhone: s.ContactPhone, AccountPhone: s.AccountPhone, ClientAccountPhone: s.AccountPhone, ClientRating: s.ClientRating, CreatedAt: s.CreatedAt, ScheduledDate: s.PreferredDate, PreferredDate: s.PreferredDate}
	var value, note *string
	for i := len(s.CounterOffers) - 1; i >= 0; i-- {
		co := s.CounterOffers[i]
		if co.Price > 0 {
			if !co.IsFromClient {
				break
			}
			v := co.FormattedPrice
			n := co.Reason
			value = &v
			note = &n
			break
		}
	}
	return gin.H{"id": s.ID, "lead": lead, "solicitation": s, "client_name": s.ClientName, "address": s.Address, "location_address": s.LocationAddress, "contact_name": s.ContactName, "contact_phone": s.ContactPhone, "client_phone": s.AccountPhone, "account_phone": s.AccountPhone, "client_rating": s.ClientRating, "scheduled_date": s.PreferredDate, "preferred_date": s.PreferredDate, "counter_offers": s.CounterOffers, "counter_offer_history": s.CounterOffers, "negotiation_count": s.RenegotiationCount, "status": s.Status, "is_order_accepted": accepted, "is_proposal_sent": hasPrice, "show_negotiation_mode": hasPrice && !accepted && s.Status != "CANCELLED", "latest_counter_value": value, "latest_counter_note": note}
}
func (p *ProviderController) GetSolicitationDetails(c *gin.Context) {
	actor, ok := p.actor(c)
	if !ok || !p.repository(c) {
		return
	}
	s, err := p.solRepo.GetSolicitationForUser(c.Request.Context(), c.Param("id"), actor)
	if err != nil {
		contractError(c, err)
		return
	}
	if s.ProviderID != actor {
		contractError(c, domain.ErrForbidden)
		return
	}
	c.JSON(200, buildQuoteDetails(s))
}
func (p *ProviderController) command(c *gin.Context, actor string, in model.SolicitationStatusUpdateInput) {
	in.ActorID = actor
	in.RequestKey = c.GetHeader("Idempotency-Key")
	s, err := p.solRepo.GetSolicitationForUser(c.Request.Context(), c.Param("id"), actor)
	if err != nil {
		contractError(c, err)
		return
	}
	if s.ProviderID != actor {
		contractError(c, domain.ErrForbidden)
		return
	}
	s, err = p.solRepo.UpdateSolicitationStatus(c.Request.Context(), c.Param("id"), in)
	if err != nil {
		contractError(c, err)
		return
	}
	c.JSON(200, buildQuoteDetails(s))
}
func (p *ProviderController) SubmitProposal(c *gin.Context) {
	actor, ok := p.actor(c)
	if !ok || !p.repository(c) {
		return
	}
	var req domain.ProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contractError(c, domain.ErrInvalidInput)
		return
	}
	in := model.SolicitationStatusUpdateInput{Status: "BUDGET_RECEIVED", CounterOfferPrice: &req.Price, CounterOfferMessage: &req.Note}
	if req.Accepted || req.AcceptCounterOffer {
		in = model.SolicitationStatusUpdateInput{Status: "ACTIVE"}
	}
	p.command(c, actor, in)
}
func (p *ProviderController) SubmitCounterOffer(c *gin.Context) {
	actor, ok := p.actor(c)
	if !ok || !p.repository(c) {
		return
	}
	var req domain.CounterOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contractError(c, domain.ErrInvalidInput)
		return
	}
	price := req.CounterPrice
	if price == 0 {
		price = req.Price
	}
	note := req.Reason
	if note == "" {
		note = req.Note
	}
	in := model.SolicitationStatusUpdateInput{Status: "NEGOTIATING", CounterOfferPrice: &price, CounterOfferMessage: &note}
	if req.Accepted || req.AcceptCounterOffer {
		in = model.SolicitationStatusUpdateInput{Status: "ACTIVE"}
	}
	p.command(c, actor, in)
}
func (p *ProviderController) AcceptCounterOffer(c *gin.Context) {
	actor, ok := p.actor(c)
	if !ok || !p.repository(c) {
		return
	}
	p.command(c, actor, model.SolicitationStatusUpdateInput{Status: "ACTIVE"})
}
func (p *ProviderController) RejectSolicitation(c *gin.Context) {
	actor, ok := p.actor(c)
	if !ok || !p.repository(c) {
		return
	}
	var req domain.RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contractError(c, domain.ErrInvalidInput)
		return
	}
	reason := req.Reason
	if reason == "" {
		reason = req.CancellationReason
	}
	if reason == "" {
		reason = req.ReasonChip
	}
	p.command(c, actor, model.SolicitationStatusUpdateInput{Status: "CANCELLED", CancellationReason: &reason})
}
func (p *ProviderController) GetPortfolio(c *gin.Context) {
	actor, ok := p.actor(c)
	if !ok {
		return
	}
	repo, ok := p.serviceRepo.(domain.BikkerAllServicesRepository)
	if !ok {
		contractError(c, domain.ErrUnavailable)
		return
	}
	services, err := repo.GetServicesByBikkerID(c.Request.Context(), actor)
	if err != nil {
		contractError(c, err)
		return
	}
	items := make([]domain.PortfolioItem, 0, len(services))
	for _, s := range services {
		items = append(items, portfolioItem(&s))
	}
	c.JSON(200, items)
}
func portfolioItem(s *domain.Service) domain.PortfolioItem {
	category := ""
	if s.Category != nil {
		category = s.Category.Name
	}
	active := s.IsActive
	return domain.PortfolioItem{ID: s.ID, Title: s.Name, Category: category, Description: s.Description, ImageUrl: s.ThumbnailURL, Photos: s.Photos, IsActive: &active, CompletedAt: s.CreatedAt, ReviewsAverage: s.ReviewsAverage, TotalReviews: s.TotalReviews}
}
func (p *ProviderController) portfolioMutation(c *gin.Context, update bool) {
	actor, ok := p.actor(c)
	if !ok {
		return
	}
	repo, ok := p.serviceRepo.(domain.ServiceMutationRepository)
	if !ok {
		contractError(c, domain.ErrUnavailable)
		return
	}
	var req domain.PortfolioCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contractError(c, domain.ErrInvalidInput)
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = strings.TrimSpace(req.Name)
	}
	category := req.CategoryID
	if category == "" {
		category = req.Category
	}
	thumbnail := req.ImageURL
	if thumbnail == "" {
		thumbnail = req.ThumbnailURL
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	s := &domain.Service{BikkerID: actor, Name: title, CategoryID: category, Description: strings.TrimSpace(req.Description), ThumbnailURL: thumbnail, Photos: req.Photos, IsActive: active}
	var err error
	if update {
		s.ID = c.Param("id")
		s, err = repo.UpdateService(c.Request.Context(), s)
	} else {
		if req.Latitude == nil || req.Longitude == nil {
			contractError(c, domain.ErrInvalidInput)
			return
		}
		s.Latitude = *req.Latitude
		s.Longitude = *req.Longitude
		s, err = repo.CreateService(c.Request.Context(), s)
	}
	if err != nil {
		contractError(c, err)
		return
	}
	status := http.StatusCreated
	action := "added"
	if update {
		status = 200
		action = "updated"
	}
	c.JSON(status, gin.H{"status": action, "id": s.ID, "item": portfolioItem(s)})
}
func (p *ProviderController) AddPortfolioItem(c *gin.Context)    { p.portfolioMutation(c, false) }
func (p *ProviderController) UpdatePortfolioItem(c *gin.Context) { p.portfolioMutation(c, true) }
func (p *ProviderController) DeletePortfolioItem(c *gin.Context) {
	actor, ok := p.actor(c)
	if !ok {
		return
	}
	repo, ok := p.serviceRepo.(domain.ServiceMutationRepository)
	if !ok {
		contractError(c, domain.ErrUnavailable)
		return
	}
	if err := repo.DeleteService(c.Request.Context(), c.Param("id"), actor); err != nil {
		contractError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "deleted", "id": c.Param("id")})
}
func (p *ProviderController) UpdateProviderProfile(c *gin.Context) {
	actor, ok := p.actor(c)
	if !ok {
		return
	}
	repo, ok := p.bikkerRepo.(interface {
		UpdateProviderProfile(context.Context, string, domain.ProviderProfileUpdateRequest) (*domain.BikkerProfileResponse, error)
	})
	if !ok {
		contractError(c, domain.ErrUnavailable)
		return
	}
	var req domain.ProviderProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contractError(c, domain.ErrInvalidInput)
		return
	}
	profile, err := repo.UpdateProviderProfile(c.Request.Context(), actor, req)
	if err != nil {
		contractError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "updated", "profile": profile})
}
func (p *ProviderController) SubmitClientEvaluation(c *gin.Context) {
	actor, ok := p.actor(c)
	if !ok || !p.repository(c) {
		return
	}
	var req domain.ClientEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contractError(c, domain.ErrInvalidInput)
		return
	}
	if err := p.solRepo.SubmitClientEvaluation(c.Request.Context(), req.SolicitationID, actor, req); err != nil {
		contractError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "evaluated", "technical_report": req.TechnicalReport})
}
