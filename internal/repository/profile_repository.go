package repository

import (
	"context"
	"fmt"

	"bikko-app/internal/model"
)

type SolicitationProvider interface {
	GetActiveSolicitations() []model.SolicitationItem
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
	var orders []model.ProfileOrderItem
	if r.solicitationProvider != nil {
		for _, s := range r.solicitationProvider.GetActiveSolicitations() {
			var statusText string
			switch s.Status {
			case "PENDING_BUDGET":
				statusText = "Aguardando Orçamento"
			case "BUDGET_RECEIVED":
				statusText = fmt.Sprintf("Orçamento Recebido · R$ %.2f", s.Price)
			case "ACTIVE":
				statusText = "Em Andamento"
			default:
				statusText = s.ScheduledDate
			}

			thumb := s.ProviderPhotoURL
			if len(s.Photos) > 0 && s.Photos[0] != "" {
				thumb = s.Photos[0]
			}

			orders = append(orders, model.ProfileOrderItem{
				ID:             s.ID,
				Name:           s.ServiceName,
				Photos:         s.Photos,
				Thumbnail:      thumb,
				Description:    s.Description,
				Distance:       statusText,
				ReviewsAverage: s.ProviderRating,
				TotalReviews:   1,
				BikkerID:       s.ID,
				Category: &model.ProfileOrderCategory{
					ID:      "cat_" + s.ID,
					Name:    s.ProviderName,
					IconURL: s.ProviderPhotoURL,
				},
			})
		}
	}

	return &model.ProfileResponse{
		ID:           id,
		ImageProfile: "https://images.pexels.com/photos/774909/pexels-photo-774909.jpeg",
		Name:         "Emerson Torres",
		Email:        "emerson@bikko.com.br",
		Phone:        "(11) 99999-9999",
		Rating:       "4.9",
		TotalRatings: 18,
		Orders:       orders,
	}, nil
}

func (r *mockProfileRepository) DeleteAccount(ctx context.Context, id string) error {
	// Soft delete logically marks account as deactivated
	return nil
}
