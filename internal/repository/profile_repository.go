package repository

import (
	"context"
	"fmt"
	"strings"

	"bikko-app/internal/model"
)

type SolicitationProvider interface {
	GetActiveSolicitations() []model.SolicitationItem
}

type UserSolicitationsProvider interface {
	GetSolicitationsByUser(ctx context.Context, userID string) ([]model.SolicitationItem, error)
}

type ProfileRepository interface {
	GetProfileByID(ctx context.Context, id string) (*model.ProfileResponse, error)
	DeleteAccount(ctx context.Context, id string) error
}

type mockProfileRepository struct {
	solicitationProvider SolicitationProvider
}

func NewMockProfileRepository(sp SolicitationProvider) ProfileRepository {
	return &mockProfileRepository{
		solicitationProvider: sp,
	}
}

func (r *mockProfileRepository) GetProfileByID(ctx context.Context, id string) (*model.ProfileResponse, error) {
	orders := make([]model.ProfileOrderItem, 0)
	var solicitations []model.SolicitationItem

	if up, ok := r.solicitationProvider.(UserSolicitationsProvider); ok && id != "" {
		if items, err := up.GetSolicitationsByUser(ctx, id); err == nil && len(items) > 0 {
			solicitations = items
		}
	}
	if len(solicitations) == 0 && r.solicitationProvider != nil {
		solicitations = r.solicitationProvider.GetActiveSolicitations()
	}

	for _, s := range solicitations {
		var statusText string
		switch s.Status {
		case "PENDING_BUDGET":
			statusText = "Aguardando Orçamento"
		case "BUDGET_RECEIVED":
			if s.Price > 0 {
				statusText = fmt.Sprintf("Orçamento Recebido · R$ %.2f", s.Price)
			} else {
				statusText = "Orçamento Recebido"
			}
		case "NEGOTIATING":
			if s.Price > 0 {
				statusText = fmt.Sprintf("Em Negociação · R$ %.2f", s.Price)
			} else {
				statusText = "Em Negociação"
			}
		case "ACTIVE", "IN_PROGRESS", "SCHEDULED":
			statusText = "Em Andamento"
		case "COMPLETED":
			statusText = "Concluído"
		case "CANCELLED":
			statusText = "Cancelado"
		default:
			statusText = s.ScheduledDate
		}

		thumb := s.ProviderPhotoURL
		if len(s.Photos) > 0 && s.Photos[0] != "" {
			thumb = s.Photos[0]
		} else if len(s.ClientPhotos) > 0 && s.ClientPhotos[0] != "" {
			thumb = s.ClientPhotos[0]
		}

		bikkerID := s.ProviderID
		if bikkerID == "" {
			bikkerID = s.ID
		}

		catName := s.CategoryName
		if catName == "" || catName == "Geral" {
			if s.Category != "" && s.Category != "Geral" {
				catName = s.Category
			} else if s.ProviderTitle != "" {
				catName = s.ProviderTitle
			} else {
				catName = "Serviço"
			}
		}

		orderServiceName := strings.TrimSpace(s.ServiceName)
		if orderServiceName == "" || orderServiceName == "Solicitação de Serviço" || orderServiceName == s.ProviderName || strings.Contains(strings.ToLower(orderServiceName), "rafael") {
			if catName != "" && catName != "Serviço" && catName != "Geral" {
				orderServiceName = catName
			}
		}

		orders = append(orders, model.ProfileOrderItem{
			ID:             s.ID,
			Name:           orderServiceName,
			Photos:         s.Photos,
			Thumbnail:      thumb,
			Description:    s.Description,
			Distance:       statusText,
			Status:         s.Status,
			ReviewsAverage: s.ProviderRating,
			TotalReviews:   1,
			BikkerID:       bikkerID,
			Category: &model.ProfileOrderCategory{
				ID:      "cat_" + s.ID,
				Name:    catName,
				IconURL: s.ProviderPhotoURL,
			},
		})
	}

	isActive := true
	return &model.ProfileResponse{
		ID:              id,
		ImageProfile:    "https://images.pexels.com/photos/774909/pexels-photo-774909.jpeg",
		Name:            "Emerson Torres",
		Email:           "emerson@bikko.com.br",
		Phone:           "(11) 99999-9999",
		Rating:          "4.9",
		TotalRatings:    18,
		Profession:      "Encanador & Eletricista Especializado",
		ExperienceYears: "Mais de 5 anos",
		Description:     "Profissional verificado com mais de 8 anos de atuação em instalações e manutenções residenciais na Grande São Paulo.",
		LocationName:    "Pinheiros, São Paulo - SP",
		Latitude:        -23.5615,
		Longitude:       -46.6960,
		IsActive:        &isActive,
		Orders:          orders,
	}, nil
}

func (r *mockProfileRepository) DeleteAccount(ctx context.Context, id string) error {
	// Soft delete logically marks account as deactivated
	return nil
}
