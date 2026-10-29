package repository

import (
	"context"
	"fmt"
	"strings"

	"bikko-app/internal/model"
)

type UserSolicitationsProvider interface {
	GetSolicitationsByUser(ctx context.Context, userID string) ([]model.SolicitationItem, error)
}

type ProfileRepository interface {
	GetProfileByID(ctx context.Context, id string) (*model.ProfileResponse, error)
}

type profileRepository struct {
	solicitationProvider UserSolicitationsProvider
}

func NewProfileRepository(sp UserSolicitationsProvider) ProfileRepository {
	return &profileRepository{solicitationProvider: sp}
}

func (r *profileRepository) GetProfileByID(ctx context.Context, id string) (*model.ProfileResponse, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("user id required")
	}
	if r.solicitationProvider == nil {
		return nil, fmt.Errorf("profile orders repository unavailable")
	}
	solicitations, err := r.solicitationProvider.GetSolicitationsByUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("profile: load user orders: %w", err)
	}
	orders := make([]model.ProfileOrderItem, 0, len(solicitations))

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
		if orderServiceName == "" || orderServiceName == "Solicitação de Serviço" || orderServiceName == s.ProviderName {
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
			TotalReviews:   0,
			BikkerID:       bikkerID,
			Category: &model.ProfileOrderCategory{
				ID:      "",
				Name:    catName,
				IconURL: s.ProviderPhotoURL,
			},
		})
	}

	return &model.ProfileResponse{ID: id, Orders: orders}, nil
}
