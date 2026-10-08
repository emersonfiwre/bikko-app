package controller

import (
	"net/http"
	"time"

	"bikko-app/internal/domain"
	"bikko-app/internal/repository"
	"github.com/gin-gonic/gin"
)

type ProviderController struct{}

func NewProviderController() *ProviderController {
	return &ProviderController{}
}

func (pc *ProviderController) GetFeed(c *gin.Context) {
	empty := c.Query("empty") == "true"
	userID := c.GetString("user_id")

	if empty || userID == "empty-qa-id" {
		c.JSON(http.StatusOK, []domain.OpportunityLead{})
		return
	}

	repo := repository.NewMockProviderRepository()
	c.JSON(http.StatusOK, repo.GetMockLeads())
}

func (pc *ProviderController) GetSolicitationDetails(c *gin.Context) {
	id := c.Param("id")
	// Return mock details
	c.JSON(http.StatusOK, gin.H{
		"id": id,
		"lead": domain.OpportunityLead{
			ID:             id,
			Title:          "Vazamento sob a pia",
			Description:    "Cano estourou sob a pia da cozinha, precisa de reparo urgente.",
			Category:       "Encanador",
			Urgency:        "EMERGENCY",
			Photos:         []string{"https://example.com/photo1.jpg", "https://example.com/photo1_2.jpg"},
			PriceType:      "BUDGET_TO_NEGOTIATE",
			DistanceKm:     2.5,
			Neighborhood:   "Vila Mariana",
			ClientName:     "Maria",
			ClientVerified: true,
			CreatedAt:      time.Now().Add(-1 * time.Hour),
		},
		"negotiation_count": 0,
		"status":            "OPEN",
	})
}

func (pc *ProviderController) SubmitProposal(c *gin.Context) {
	var req domain.ProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "proposal_submitted", "message": "Proposta enviada com sucesso."})
}

func (pc *ProviderController) SubmitCounterOffer(c *gin.Context) {
	var req domain.CounterOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "counter_offer_submitted", "message": "Contra-proposta enviada com sucesso.", "negotiation_count": 1})
}

func (pc *ProviderController) RejectSolicitation(c *gin.Context) {
	var req domain.RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "rejected", "message": "Solicitação rejeitada."})
}

func (pc *ProviderController) GetPortfolio(c *gin.Context) {
	empty := c.Query("empty") == "true"
	if empty {
		c.JSON(http.StatusOK, []domain.PortfolioItem{})
		return
	}

	repo := repository.NewMockProviderRepository()
	c.JSON(http.StatusOK, repo.GetMockPortfolio())
}

func (pc *ProviderController) AddPortfolioItem(c *gin.Context) {
	// Usually would handle form-data or JSON with photo URL
	c.JSON(http.StatusCreated, gin.H{"status": "added", "id": "port-new"})
}

func (pc *ProviderController) SubmitClientEvaluation(c *gin.Context) {
	var req domain.ClientEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "evaluated", "message": "Avaliação salva com sucesso."})
}
