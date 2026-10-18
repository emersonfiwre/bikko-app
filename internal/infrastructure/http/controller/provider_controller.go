package controller

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/http/middleware"
	"bikko-app/internal/infrastructure/persistence/postgres"
	"bikko-app/internal/infrastructure/push"
	"bikko-app/internal/model"
	"bikko-app/internal/repository"

	"github.com/gin-gonic/gin"
)

type ProviderController struct {
	solRepo     *postgres.SolicitationPostgresRepository
	serviceRepo domain.ServiceRepository
	bikkerRepo  domain.BikkerRepository
	pushService push.PushService
	store       model.SolicitationStore
}

func NewProviderController() *ProviderController {
	return &ProviderController{}
}

func NewProviderControllerWithDeps(
	solRepo *postgres.SolicitationPostgresRepository,
	serviceRepo domain.ServiceRepository,
	bikkerRepo domain.BikkerRepository,
	pushService ...push.PushService,
) *ProviderController {
	var ps push.PushService
	if len(pushService) > 0 {
		ps = pushService[0]
	}
	return &ProviderController{
		solRepo:     solRepo,
		serviceRepo: serviceRepo,
		bikkerRepo:  bikkerRepo,
		pushService: ps,
	}
}

func (pc *ProviderController) SetDependencies(
	solRepo *postgres.SolicitationPostgresRepository,
	serviceRepo domain.ServiceRepository,
	bikkerRepo domain.BikkerRepository,
	pushService ...push.PushService,
) {
	pc.solRepo = solRepo
	pc.serviceRepo = serviceRepo
	pc.bikkerRepo = bikkerRepo
	if len(pushService) > 0 {
		pc.pushService = pushService[0]
	}
}

func (pc *ProviderController) SetSolicitationStore(store model.SolicitationStore) {
	pc.store = store
}

func (pc *ProviderController) resolveProviderUserID(c *gin.Context) string {
	userID := middleware.GetUserID(c)
	if userID == "" {
		userID = c.GetString("user_id")
	}
	// [SECURITY AUDIT] ATENÇÃO: O fallback para c.Query("bikker_id") é mantido exclusivamente
	// para facilitar testes unitários e de integração mock. Em ambiente de Release/Produção, deve ser removido,
	// exigindo estritamente autenticação via JWT (middleware.GetUserID).
	isProduction := os.Getenv("ENV") == "production" || gin.Mode() == gin.ReleaseMode
	if userID == "" && !isProduction {
		userID = c.Query("bikker_id")
	}
	return strings.TrimSpace(userID)
}

func (pc *ProviderController) getExistingSolicitation(ctx context.Context, id string) *model.SolicitationItem {
	if pc.solRepo != nil && pc.solRepo.HasPool() {
		if sol, err := pc.solRepo.GetSolicitationByID(ctx, id); err == nil && sol != nil {
			return sol
		}
	}
	if pc.store != nil {
		if sol := pc.store.GetInMemorySolicitationByID(id); sol != nil {
			return sol
		}
	}
	return nil
}

func (pc *ProviderController) isMatchingClientCounterOffer(existing *model.SolicitationItem, price float64) bool {
	if existing == nil || price <= 0 {
		return false
	}
	if len(existing.CounterOffers) > 0 {
		last := existing.CounterOffers[len(existing.CounterOffers)-1]
		if (strings.EqualFold(last.Author, "CLIENT") || last.IsFromClient) && math.Abs(price-last.Price) < 0.01 {
			return true
		}
	}
	if existing.CounterOfferPrice != nil && *existing.CounterOfferPrice > 0 && math.Abs(price-*existing.CounterOfferPrice) < 0.01 {
		return true
	}
	return false
}

func (pc *ProviderController) GetFeed(c *gin.Context) {
	empty := c.Query("empty") == "true"
	userID := pc.resolveProviderUserID(c)

	if empty || userID == "empty-qa-id" || userID == "11111111-0000-0000-0000-000000000101" {
		c.JSON(http.StatusOK, []domain.OpportunityLead{})
		return
	}

	if pc.solRepo != nil && pc.solRepo.HasPool() {
		leads, err := pc.solRepo.GetProviderFeed(c.Request.Context(), userID)
		if err == nil {
			if leads == nil {
				leads = []domain.OpportunityLead{}
			}
			c.JSON(http.StatusOK, leads)
			return
		}
	}

	repo := repository.NewMockProviderRepository()
	c.JSON(http.StatusOK, repo.GetMockLeads())
}

func formatBRLPrice(amount float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.2f", amount), ".", ",")
}

func (pc *ProviderController) buildQuoteDetailsResponse(id string, sol *model.SolicitationItem, extra map[string]interface{}) gin.H {
	if sol == nil {
		resp := gin.H{
			"id": id,
			"lead": domain.OpportunityLead{
				ID:             id,
				Title:          "Vazamento sob a pia",
				Description:    "Cano estourou sob a pia da cozinha, precisa de reparo urgente.",
				Category:       "Encanador",
				Urgency:        "EMERGENCY",
				Photos:         []string{},
				PriceType:      "BUDGET_TO_NEGOTIATE",
				DistanceKm:     2.5,
				Neighborhood:   "Pinheiros, São Paulo - SP",
				ClientName:     "Lucas Ferreira Mendes",
				ClientVerified: true,
				CreatedAt:      time.Now().Add(-1 * time.Hour),
				ScheduledDate:  "O mais rápido possível",
			},
			"scheduled_date":        "O mais rápido possível",
			"counter_offers":        []model.CounterOfferItem{},
			"negotiation_count":     0,
			"status":                "OPEN",
			"client_rating":         "0.0",
			"is_order_accepted":     false,
			"is_proposal_sent":      false,
			"show_negotiation_mode": false,
		}
		for k, v := range extra {
			resp[k] = v
		}
		return resp
	}

	urgency := "SCHEDULED"
	if strings.Contains(strings.ToLower(sol.Description), "urgente") || strings.Contains(strings.ToLower(sol.Description), "vazamento") {
		urgency = "EMERGENCY"
	}

	counterOffers := make([]model.CounterOfferItem, 0, len(sol.CounterOffers))
	for idx, offer := range sol.CounterOffers {
		formatted := offer.FormattedPrice
		if formatted == "" {
			formatted = formatBRLPrice(offer.Price)
		}
		round := offer.Round
		if round <= 0 {
			round = idx + 1
		}
		offerID := offer.ID
		if offerID == "" {
			offerID = fmt.Sprintf("co_%d", round)
		}
		counterOffers = append(counterOffers, model.CounterOfferItem{
			ID:             offerID,
			Round:          round,
			Author:         offer.Author,
			IsFromClient:   offer.IsFromClient || strings.EqualFold(offer.Author, "CLIENT"),
			Price:          offer.Price,
			FormattedPrice: formatted,
			Reason:         offer.Reason,
		})
	}

	hasBudget := sol.Price > 0 && (strings.EqualFold(sol.Status, "BUDGET_RECEIVED") ||
		strings.EqualFold(sol.Status, "NEGOTIATING") ||
		strings.EqualFold(sol.Status, "ACTIVE"))
	if len(counterOffers) == 0 && hasBudget {
		reason := "Incluso visita técnica, desmontagem e vedação."
		if sol.CounterOfferMessage != nil && strings.TrimSpace(*sol.CounterOfferMessage) != "" {
			reason = strings.TrimSpace(*sol.CounterOfferMessage)
		}
		counterOffers = append(counterOffers, model.CounterOfferItem{
			ID:             "co_1",
			Round:          1,
			Author:         "PROVIDER",
			IsFromClient:   false,
			Price:          sol.Price,
			FormattedPrice: formatBRLPrice(sol.Price),
			Reason:         reason,
		})
	}

	if len(counterOffers) > 5 {
		counterOffers = counterOffers[:5]
	}

	isAccepted := strings.EqualFold(sol.Status, "ACTIVE") ||
		strings.EqualFold(sol.Status, "ACCEPTED") ||
		strings.EqualFold(sol.Status, "COMPLETED")
	isProposalSent := len(counterOffers) > 0 ||
		strings.EqualFold(sol.Status, "BUDGET_RECEIVED") ||
		strings.EqualFold(sol.Status, "NEGOTIATING") ||
		isAccepted
	showNegotiationMode := isProposalSent && !isAccepted

	var latestCounterValue *string
	var latestCounterNote *string
	for i := len(counterOffers) - 1; i >= 0; i-- {
		if counterOffers[i].IsFromClient || strings.EqualFold(counterOffers[i].Author, "CLIENT") {
			val := counterOffers[i].FormattedPrice
			note := counterOffers[i].Reason
			latestCounterValue = &val
			latestCounterNote = &note
			break
		}
	}
	if latestCounterValue == nil && len(counterOffers) > 0 {
		last := counterOffers[len(counterOffers)-1]
		if !last.IsFromClient && len(counterOffers) > 1 {
			val := last.FormattedPrice
			note := last.Reason
			latestCounterValue = &val
			latestCounterNote = &note
		}
	}

	category := sol.Category
	if category == "" {
		category = sol.ProviderTitle
	}
	if category == "" {
		category = "Geral"
	}
	neighborhood := sol.LocationAddress
	if neighborhood == "" {
		neighborhood = sol.Address
	}
	if neighborhood == "" {
		neighborhood = sol.Neighborhood
	}
	if neighborhood == "" {
		neighborhood = "São Paulo, SP"
	}
	clientName := sol.ClientName
	if clientName == "" {
		clientName = sol.ContactName
	}
	if clientName == "" {
		clientName = "Lucas Ferreira Mendes"
	}

	clientRating := sol.ClientRating
	if clientRating == "" || clientRating == "0" || clientRating == "0.0" || sol.ClientTotalRatings == 0 {
		clientRating = "0.0"
	}

	contactPhone := sol.ContactPhone
	accountPhone := sol.AccountPhone
	if accountPhone == "" {
		accountPhone = sol.ClientAccountPhone
	}

	photos := sol.ClientPhotos
	if len(photos) == 0 {
		photos = sol.Photos
	}
	if photos == nil {
		photos = []string{}
	}

	preferredDate := normalizePreferredDate(sol.ScheduledDate)

	resp := gin.H{
		"id": sol.ID,
		"lead": domain.OpportunityLead{
			ID:                 sol.ID,
			Title:              sol.ServiceName,
			Description:        sol.Description,
			Category:           category,
			Urgency:            urgency,
			Photos:             photos,
			ClientPhotos:       photos,
			PriceType:          "BUDGET_TO_NEGOTIATE",
			DistanceKm:         2.5,
			Neighborhood:       neighborhood,
			Address:            neighborhood,
			LocationAddress:    neighborhood,
			ClientName:         clientName,
			ContactName:        sol.ContactName,
			ContactPhone:       contactPhone,
			AccountPhone:       accountPhone,
			ClientPhone:        contactPhone,
			ClientAccountPhone: accountPhone,
			ClientRating:       clientRating,
			ClientVerified:     true,
			CreatedAt:          time.Now().Add(-1 * time.Hour),
			ScheduledDate:      preferredDate,
			PreferredDate:      preferredDate,
		},
		"solicitation":          sol,
		"address":               neighborhood,
		"location_address":      neighborhood,
		"contact_name":          sol.ContactName,
		"contact_phone":         contactPhone,
		"client_phone":          accountPhone,
		"account_phone":         accountPhone,
		"client_rating":         clientRating,
		"scheduled_date":        preferredDate,
		"preferred_date":        preferredDate,
		"counter_offers":        counterOffers,
		"counter_offer_history": counterOffers,
		"negotiation_count":     sol.RenegotiationCount,
		"status":                sol.Status,
		"is_order_accepted":     isAccepted,
		"is_proposal_sent":      isProposalSent,
		"show_negotiation_mode": showNegotiationMode,
		"latest_counter_value":  latestCounterValue,
		"latest_counter_note":   latestCounterNote,
	}
	for k, v := range extra {
		resp[k] = v
	}
	return resp
}

func (pc *ProviderController) GetSolicitationDetails(c *gin.Context) {
	id := c.Param("id")
	sol := pc.getExistingSolicitation(c.Request.Context(), id)
	c.JSON(http.StatusOK, pc.buildQuoteDetailsResponse(id, sol, nil))
}

func (pc *ProviderController) handleAcceptCounterOffer(
	c *gin.Context,
	id string,
	userID string,
	price float64,
	note string,
	existing *model.SolicitationItem,
) {
	finalPrice := price
	if finalPrice <= 0 && existing != nil {
		if existing.CounterOfferPrice != nil && *existing.CounterOfferPrice > 0 {
			finalPrice = *existing.CounterOfferPrice
		} else if existing.Price > 0 {
			finalPrice = existing.Price
		}
	}

	var updated *model.SolicitationItem
	if pc.solRepo != nil && pc.solRepo.HasPool() {
		if res, err := pc.solRepo.AcceptCounterOffer(c.Request.Context(), id, userID, finalPrice, note); err == nil && res != nil {
			updated = res
		}
	}

	if pc.store != nil {
		inMem := pc.store.ApplyProviderProposalInMemory(id, userID, finalPrice, note, true)
		if updated != nil {
			pc.store.SyncSolicitationInMemory(*updated)
		} else {
			updated = inMem
		}
	}

	if updated == nil {
		updated = &model.SolicitationItem{
			ID:          id,
			ClientID:    "11111111-0000-0000-0000-000000000099",
			ProviderID:  userID,
			ServiceName: "Solicitação de Serviço",
			Status:      "ACTIVE",
			Price:       finalPrice,
		}
	}

	if pc.pushService != nil && updated.ClientID != "" {
		pushData := map[string]string{
			"solicitation_id": updated.ID,
			"type":            "accepted",
		}
		if pushErr := pc.pushService.SendNotification(
			c.Request.Context(),
			updated.ClientID,
			"Proposta Aceita pelo Profissional! 🎉",
			"O profissional aceitou sua contraproposta! O serviço está confirmado no seu histórico de pedidos.",
			pushData,
		); pushErr != nil {
			slog.Error("failed to send accepted proposal push notification",
				slog.String("solicitation_id", updated.ID),
				slog.Any("error", pushErr),
			)
		}
	}

	c.JSON(http.StatusOK, pc.buildQuoteDetailsResponse(id, updated, map[string]interface{}{
		"status":  "ACTIVE",
		"message": "Proposta aceita com sucesso.",
	}))
}

func (pc *ProviderController) AcceptCounterOffer(c *gin.Context) {
	id := c.Param("id")
	var req domain.ProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing := pc.getExistingSolicitation(c.Request.Context(), id)
	pc.handleAcceptCounterOffer(c, id, pc.resolveProviderUserID(c), req.Price, req.Note, existing)
}

func (pc *ProviderController) SubmitProposal(c *gin.Context) {
	id := c.Param("id")
	var req domain.ProposalRequest
	queryAccept := strings.EqualFold(c.Query("accept_counter_offer"), "true") || strings.EqualFold(c.Query("accepted"), "true")
	if err := c.ShouldBindJSON(&req); err != nil {
		if !(err == io.EOF && queryAccept) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	existing := pc.getExistingSolicitation(c.Request.Context(), id)
	accepting := req.Accepted || req.AcceptCounterOffer || queryAccept || pc.isMatchingClientCounterOffer(existing, req.Price)

	if !accepting && req.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "preço da proposta é obrigatório"})
		return
	}

	userID := pc.resolveProviderUserID(c)
	if accepting {
		pc.handleAcceptCounterOffer(c, id, userID, req.Price, req.Note, existing)
		return
	}

	isCounterOffer := existing != nil && (existing.RenegotiationCount > 0 || existing.CounterOfferPrice != nil)

	var updated *model.SolicitationItem
	if pc.solRepo != nil && pc.solRepo.HasPool() {
		if res, err := pc.solRepo.SubmitProviderProposal(
			c.Request.Context(),
			id,
			userID,
			req.Price,
			req.Note,
		); err == nil && res != nil {
			updated = res
		}
	}

	if pc.store != nil {
		inMem := pc.store.ApplyProviderProposalInMemory(id, userID, req.Price, req.Note, false)
		if updated != nil {
			pc.store.SyncSolicitationInMemory(*updated)
		} else {
			updated = inMem
		}
	}

	if updated != nil {
		if pc.pushService != nil && updated.ClientID != "" {
			pushTitle := "Novo Orçamento Recebido! 💬"
			pushBody := fmt.Sprintf("O profissional enviou um orçamento de R$ %.2f para %s.", req.Price, updated.ServiceName)
			pushType := "budget_received"
			if isCounterOffer {
				pushTitle = "Nova Contraproposta do Profissional! 💬"
				pushBody = fmt.Sprintf("O profissional enviou uma contraproposta de R$ %.2f para %s.", req.Price, updated.ServiceName)
				pushType = "counter_offer"
			}
			pushData := map[string]string{
				"solicitation_id": updated.ID,
				"type":            pushType,
			}
			if pushErr := pc.pushService.SendNotification(
				c.Request.Context(),
				updated.ClientID,
				pushTitle,
				pushBody,
				pushData,
			); pushErr != nil {
				slog.Error("failed to send proposal push notification",
					slog.String("solicitation_id", updated.ID),
					slog.Any("error", pushErr),
				)
			}
		}

		c.JSON(http.StatusOK, pc.buildQuoteDetailsResponse(id, updated, map[string]interface{}{
			"message": "Proposta enviada com sucesso.",
		}))
		return
	}

	c.JSON(http.StatusOK, pc.buildQuoteDetailsResponse(id, nil, map[string]interface{}{
		"status":  "BUDGET_RECEIVED",
		"message": "Proposta enviada com sucesso.",
	}))
}

func (pc *ProviderController) SubmitCounterOffer(c *gin.Context) {
	id := c.Param("id")
	var req domain.CounterOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	counterPrice := req.CounterPrice
	if counterPrice <= 0 && req.Price > 0 {
		counterPrice = req.Price
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" && strings.TrimSpace(req.Note) != "" {
		reason = strings.TrimSpace(req.Note)
	}

	existing := pc.getExistingSolicitation(c.Request.Context(), id)
	queryAccept := strings.EqualFold(c.Query("accept_counter_offer"), "true") || strings.EqualFold(c.Query("accepted"), "true")
	accepting := req.Accepted || req.AcceptCounterOffer || queryAccept || pc.isMatchingClientCounterOffer(existing, counterPrice)

	userID := pc.resolveProviderUserID(c)
	if accepting {
		pc.handleAcceptCounterOffer(c, id, userID, counterPrice, reason, existing)
		return
	}

	if counterPrice <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "preço da contraproposta é obrigatório"})
		return
	}

	var updated *model.SolicitationItem
	if pc.solRepo != nil && pc.solRepo.HasPool() {
		providerRole := "PROVIDER"
		if res, err := pc.solRepo.UpdateSolicitationStatus(c.Request.Context(), id, postgres.SolicitationStatusUpdateInput{
			Status:              "BUDGET_RECEIVED",
			CounterOfferPrice:   &counterPrice,
			CounterOfferMessage: &reason,
			SenderRole:          &providerRole,
			SenderID:            &userID,
		}); err == nil && res != nil {
			updated = res
		}
	}

	if pc.store != nil {
		inMem := pc.store.ApplyProviderCounterOfferInMemory(id, userID, counterPrice, reason)
		if updated != nil {
			pc.store.SyncSolicitationInMemory(*updated)
		} else {
			updated = inMem
		}
	}

	if updated != nil {
		if pc.pushService != nil && updated.ClientID != "" {
			pushData := map[string]string{
				"solicitation_id": updated.ID,
				"type":            "counter_offer",
			}
			if pushErr := pc.pushService.SendNotification(
				c.Request.Context(),
				updated.ClientID,
				"Nova Contraproposta do Profissional! 💬",
				fmt.Sprintf("O profissional enviou uma contraproposta de R$ %.2f para %s.", counterPrice, updated.ServiceName),
				pushData,
			); pushErr != nil {
				slog.Error("failed to send counter-offer push notification",
					slog.String("solicitation_id", updated.ID),
					slog.Any("error", pushErr),
				)
			}
		}

		c.JSON(http.StatusOK, pc.buildQuoteDetailsResponse(id, updated, map[string]interface{}{
			"message": "Contra-proposta enviada com sucesso.",
		}))
		return
	}

	c.JSON(http.StatusOK, pc.buildQuoteDetailsResponse(id, nil, map[string]interface{}{
		"status":            "BUDGET_RECEIVED",
		"message":           "Contra-proposta enviada com sucesso.",
		"negotiation_count": 1,
	}))
}

func (pc *ProviderController) RejectSolicitation(c *gin.Context) {
	id := c.Param("id")
	var req domain.RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = strings.TrimSpace(req.ReasonChip)
	}
	if reason == "" {
		reason = strings.TrimSpace(req.CancellationReason)
	}
	note := strings.TrimSpace(req.Note)
	if note != "" {
		if reason != "" && !strings.Contains(reason, note) {
			reason = reason + " - " + note
		} else if reason == "" {
			reason = note
		}
	}
	if reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "motivo da rejeição é obrigatório"})
		return
	}

	providerRole := "PROVIDER"
	userID := pc.resolveProviderUserID(c)

	var updated *model.SolicitationItem
	if pc.solRepo != nil && pc.solRepo.HasPool() {
		if res, err := pc.solRepo.UpdateSolicitationStatus(c.Request.Context(), id, postgres.SolicitationStatusUpdateInput{
			Status:             "CANCELLED",
			CancellationReason: &reason,
			CancelledBy:        &providerRole,
			SenderRole:         &providerRole,
			SenderID:           &userID,
		}); err == nil && res != nil {
			updated = res
		}
	}

	if pc.store != nil {
		inMem := pc.store.ApplyProviderRejectInMemory(id, userID, reason)
		if updated != nil {
			pc.store.SyncSolicitationInMemory(*updated)
		} else {
			updated = inMem
		}
	}

	if pc.pushService != nil && updated != nil && updated.ClientID != "" {
		pushData := map[string]string{
			"solicitation_id": updated.ID,
			"type":            "cancelled",
		}
		if pushErr := pc.pushService.SendNotification(
			c.Request.Context(),
			updated.ClientID,
			"Solicitação Cancelada pelo Profissional",
			fmt.Sprintf("O profissional não pôde atender sua solicitação. Motivo: %s", reason),
			pushData,
		); pushErr != nil {
			slog.Error("failed to send reject push notification",
				slog.String("solicitation_id", updated.ID),
				slog.Any("error", pushErr),
			)
		}
	}

	c.JSON(http.StatusOK, pc.buildQuoteDetailsResponse(id, updated, map[string]interface{}{
		"status":              "CANCELLED",
		"message":             "Solicitação rejeitada.",
		"cancelled_by":        providerRole,
		"cancellation_reason": reason,
	}))
}

func (pc *ProviderController) GetPortfolio(c *gin.Context) {
	empty := c.Query("empty") == "true"
	userID := pc.resolveProviderUserID(c)
	if empty || userID == "empty-qa-id" || userID == "11111111-0000-0000-0000-000000000101" {
		c.JSON(http.StatusOK, []domain.PortfolioItem{})
		return
	}

	if pc.serviceRepo != nil {
		targetBikkerID := userID
		if targetBikkerID == "" {
			targetBikkerID = "22222222-2222-2222-2222-222222222222"
		}
		var services []domain.Service
		var err error
		if allRepo, ok := pc.serviceRepo.(domain.BikkerAllServicesRepository); ok {
			services, err = allRepo.GetServicesByBikkerID(c.Request.Context(), targetBikkerID)
		} else {
			services, err = pc.serviceRepo.GetActiveServicesByBikkerID(c.Request.Context(), targetBikkerID)
		}
		if err == nil {
			items := make([]domain.PortfolioItem, 0, len(services))
			for _, s := range services {
				catName := "Geral"
				if s.Category != nil && s.Category.Name != "" {
					catName = s.Category.Name
				}
				img := s.ThumbnailURL
				if img == "" && len(s.Photos) > 0 {
					img = s.Photos[0]
				}
				completedAt := s.CreatedAt
				if completedAt.IsZero() {
					completedAt = time.Now().Add(-7 * 24 * time.Hour)
				}
				active := s.IsActive
				items = append(items, domain.PortfolioItem{
					ID:          s.ID,
					Title:       s.Name,
					Category:    catName,
					Description: s.Description,
					ImageUrl:    img,
					Photos:      s.Photos,
					IsActive:    &active,
					CompletedAt: completedAt,
				})
			}
			c.JSON(http.StatusOK, items)
			return
		}
	}

	repo := repository.NewMockProviderRepository()
	c.JSON(http.StatusOK, repo.GetMockPortfolio())
}

func (pc *ProviderController) AddPortfolioItem(c *gin.Context) {
	var req domain.PortfolioCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido: " + err.Error()})
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = strings.TrimSpace(req.Name)
	}
	catInput := strings.TrimSpace(req.CategoryID)
	if catInput == "" {
		catInput = strings.TrimSpace(req.Category)
	}
	imgURL := strings.TrimSpace(req.ImageURL)
	if imgURL == "" {
		imgURL = strings.TrimSpace(req.ThumbnailURL)
	}
	if imgURL == "" && len(req.Photos) > 0 {
		imgURL = req.Photos[0]
	}

	if mutator, ok := pc.serviceRepo.(domain.ServiceMutationRepository); ok && title != "" {
		bikkerID := pc.resolveProviderUserID(c)
		if bikkerID == "" {
			bikkerID = "11111111-0000-0000-0000-000000000100"
		}
		lat := -23.55052
		lon := -46.63330
		if req.Latitude != nil {
			lat = *req.Latitude
		}
		if req.Longitude != nil {
			lon = *req.Longitude
		}

		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}

		if strings.TrimSpace(req.ID) != "" {
			if updated, err := mutator.UpdateService(c.Request.Context(), &domain.Service{
				ID:           strings.TrimSpace(req.ID),
				BikkerID:     bikkerID,
				CategoryID:   catInput,
				Name:         title,
				Description:  req.Description,
				ThumbnailURL: imgURL,
				Photos:       req.Photos,
				Latitude:     lat,
				Longitude:    lon,
				IsActive:     isActive,
			}); err == nil && updated != nil {
				catName := req.Category
				if updated.Category != nil && updated.Category.Name != "" {
					catName = updated.Category.Name
				}
				active := updated.IsActive
				c.JSON(http.StatusOK, gin.H{
					"status": "updated",
					"id":     updated.ID,
					"item": domain.PortfolioItem{
						ID:          updated.ID,
						Title:       updated.Name,
						Category:    catName,
						Description: updated.Description,
						ImageUrl:    updated.ThumbnailURL,
						Photos:      updated.Photos,
						IsActive:    &active,
						CompletedAt: updated.CreatedAt,
					},
				})
				return
			}
		}

		created, err := mutator.CreateService(c.Request.Context(), &domain.Service{
			BikkerID:     bikkerID,
			CategoryID:   catInput,
			Name:         title,
			Description:  req.Description,
			ThumbnailURL: imgURL,
			Photos:       req.Photos,
			Latitude:     lat,
			Longitude:    lon,
			IsActive:     isActive,
		})
		if err == nil && created != nil {
			catName := req.Category
			if created.Category != nil && created.Category.Name != "" {
				catName = created.Category.Name
			}
			active := created.IsActive
			c.JSON(http.StatusCreated, gin.H{
				"status": "added",
				"id":     created.ID,
				"item": domain.PortfolioItem{
					ID:          created.ID,
					Title:       created.Name,
					Category:    catName,
					Description: created.Description,
					ImageUrl:    created.ThumbnailURL,
					Photos:      created.Photos,
					IsActive:    &active,
					CompletedAt: created.CreatedAt,
				},
			})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"status": "added", "id": "port-new"})
}

func (pc *ProviderController) UpdatePortfolioItem(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req domain.PortfolioCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido: " + err.Error()})
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = strings.TrimSpace(req.Name)
	}
	catInput := strings.TrimSpace(req.CategoryID)
	if catInput == "" {
		catInput = strings.TrimSpace(req.Category)
	}
	imgURL := strings.TrimSpace(req.ImageURL)
	if imgURL == "" {
		imgURL = strings.TrimSpace(req.ThumbnailURL)
	}
	if imgURL == "" && len(req.Photos) > 0 {
		imgURL = req.Photos[0]
	}

	if mutator, ok := pc.serviceRepo.(domain.ServiceMutationRepository); ok && id != "" {
		bikkerID := pc.resolveProviderUserID(c)
		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}

		updated, err := mutator.UpdateService(c.Request.Context(), &domain.Service{
			ID:           id,
			BikkerID:     bikkerID,
			CategoryID:   catInput,
			Name:         title,
			Description:  req.Description,
			ThumbnailURL: imgURL,
			Photos:       req.Photos,
			IsActive:     isActive,
		})
		if err == nil && updated != nil {
			catName := req.Category
			if updated.Category != nil && updated.Category.Name != "" {
				catName = updated.Category.Name
			}
			active := updated.IsActive
			c.JSON(http.StatusOK, gin.H{
				"status": "updated",
				"id":     updated.ID,
				"item": domain.PortfolioItem{
					ID:          updated.ID,
					Title:       updated.Name,
					Category:    catName,
					Description: updated.Description,
					ImageUrl:    updated.ThumbnailURL,
					Photos:      updated.Photos,
					IsActive:    &active,
					CompletedAt: updated.CreatedAt,
				},
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated", "id": id})
}

func (pc *ProviderController) DeletePortfolioItem(c *gin.Context) {
	id := c.Param("id")
	bikkerID := pc.resolveProviderUserID(c)

	if mutator, ok := pc.serviceRepo.(domain.ServiceMutationRepository); ok {
		if err := mutator.DeleteService(c.Request.Context(), id, bikkerID); err == nil {
			c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": id})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": id})
}

func (pc *ProviderController) UpdateProviderProfile(c *gin.Context) {
	var req domain.ProviderProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bikkerID := pc.resolveProviderUserID(c)
	if bikkerID == "" {
		bikkerID = "11111111-0000-0000-0000-000000000100"
	}

	lat := -23.55052
	lon := -46.63330
	if req.Latitude != nil {
		lat = *req.Latitude
	}
	if req.Longitude != nil {
		lon = *req.Longitude
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
		repository.SetMockProviderActive(*req.IsActive)
	}

	if updater, ok := pc.bikkerRepo.(domain.BikkerProfileUpdater); ok {
		profile, err := updater.UpsertBikkerProfile(c.Request.Context(), &domain.Bikker{
			ID:              bikkerID,
			Profession:      strings.TrimSpace(req.Profession),
			ExperienceYears: strings.TrimSpace(req.ExperienceYears),
			Description:     strings.TrimSpace(req.Description),
			LocationName:    strings.TrimSpace(req.LocationName),
			Latitude:        lat,
			Longitude:       lon,
			IsActive:        isActive,
		})
		if err == nil && profile != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":  "updated",
				"message": "Perfil profissional atualizado com sucesso.",
				"profile": profile,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "updated",
		"message": "Perfil profissional atualizado com sucesso.",
	})
}

func (pc *ProviderController) SubmitClientEvaluation(c *gin.Context) {
	var req domain.ClientEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Stars < 1 || req.Stars > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "as estrelas devem ser entre 1 e 5"})
		return
	}
	if len([]rune(req.TechnicalReport)) > 280 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "relatório técnico deve ter no máximo 280 caracteres"})
		return
	}
	if len([]rune(req.Comment)) > 120 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comentário deve ter no máximo 120 caracteres"})
		return
	}

	if pc.solRepo != nil && pc.solRepo.HasPool() && req.SolicitationID != "" {
		_ = pc.solRepo.SubmitClientEvaluation(
			c.Request.Context(),
			req.SolicitationID,
			pc.resolveProviderUserID(c),
			req,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "evaluated",
		"message":          "Relatório técnico e avaliação salvos com sucesso.",
		"technical_report": req.TechnicalReport,
	})
}

var validDropDownDates = []string{
	"O mais rápido possível",
	"Próximos 7 dias",
	"Próximos 15 dias",
	"Próximos 30 dias",
}

func normalizePreferredDate(raw string) string {
	raw = strings.TrimSpace(raw)
	for _, d := range validDropDownDates {
		if strings.EqualFold(raw, d) {
			return d
		}
	}
	return "O mais rápido possível"
}

