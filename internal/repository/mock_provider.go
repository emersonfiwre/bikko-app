package repository

import (
	"time"

	"bikko-app/internal/domain"
)

type MockProviderRepository struct{}

func NewMockProviderRepository() *MockProviderRepository {
	return &MockProviderRepository{}
}

func (r *MockProviderRepository) GetMockLeads() []domain.OpportunityLead {
	return []domain.OpportunityLead{
		{
			ID:             "lead-1",
			Title:          "Vazamento sob a pia",
			Description:    "Cano estourou sob a pia da cozinha, precisa de reparo urgente.",
			Category:       "Encanador",
			Urgency:        "EMERGENCY",
			Photos:         []string{"https://example.com/photo1.jpg"},
			PriceType:      "BUDGET_TO_NEGOTIATE",
			DistanceKm:     2.5,
			Neighborhood:   "Vila Mariana",
			ClientName:     "Maria",
			ClientVerified: true,
			CreatedAt:      time.Now().Add(-1 * time.Hour),
		},
		{
			ID:             "lead-2",
			Title:          "Pintura de sala 40m²",
			Description:    "Pintura de parede interna cor branca.",
			Category:       "Pintor",
			Urgency:        "SCHEDULED",
			Photos:         []string{"https://example.com/photo2.jpg"},
			PriceType:      "BUDGET_TO_NEGOTIATE",
			DistanceKm:     5.0,
			Neighborhood:   "Pinheiros",
			ClientName:     "João",
			ClientVerified: true,
			CreatedAt:      time.Now().Add(-5 * time.Hour),
		},
		{
			ID:             "lead-3",
			Title:          "Instalação de ar-condicionado",
			Description:    "Instalar split 9000 BTUs.",
			Category:       "Eletricista",
			Urgency:        "SCHEDULED",
			Photos:         []string{},
			PriceType:      "BUDGET_TO_NEGOTIATE",
			DistanceKm:     8.2,
			Neighborhood:   "Itaim Bibi",
			ClientName:     "Carlos",
			ClientVerified: false,
			CreatedAt:      time.Now().Add(-24 * time.Hour),
		},
	}
}

func (r *MockProviderRepository) GetMockPortfolio() []domain.PortfolioItem {
	return []domain.PortfolioItem{
		{
			ID:          "port-1",
			Title:       "Reparo de Encanamento",
			Category:    "Encanador",
			ImageUrl:    "https://example.com/port1.jpg",
			CompletedAt: time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			ID:          "port-2",
			Title:       "Pintura Externa",
			Category:    "Pintor",
			ImageUrl:    "https://example.com/port2.jpg",
			CompletedAt: time.Now().Add(-60 * 24 * time.Hour),
		},
	}
}
