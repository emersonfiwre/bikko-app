package usecase

import (
	"context"
	"fmt"
	"strings"

	"bikko-app/internal/domain"
	"bikko-app/internal/model"
)

// SolicitationUseCase centralizes all business logic and invariants for service solicitations,
// proposals, negotiations, counter-offers, and provider evaluations.
type SolicitationUseCase struct {
	solRepo     model.SolicitationRepository
	serviceRepo domain.ServiceRepository
}

func NewSolicitationUseCase(
	solRepo model.SolicitationRepository,
	serviceRepo domain.ServiceRepository,
) *SolicitationUseCase {
	return &SolicitationUseCase{
		solRepo:     solRepo,
		serviceRepo: serviceRepo,
	}
}

func (uc *SolicitationUseCase) GetSolicitations(ctx context.Context, actor string, limit, offset int) ([]model.SolicitationItem, error) {
	if uc.solRepo == nil {
		return nil, domain.ErrUnavailable
	}
	if pager, ok := uc.solRepo.(interface {
		GetSolicitationsPage(context.Context, string, int, int) ([]model.SolicitationItem, error)
	}); ok {
		return pager.GetSolicitationsPage(ctx, actor, limit, offset)
	}
	return uc.solRepo.GetSolicitationsByUser(ctx, actor)
}

func (uc *SolicitationUseCase) GetSolicitationByID(ctx context.Context, id, actor string) (*model.SolicitationItem, error) {
	if uc.solRepo == nil {
		return nil, domain.ErrUnavailable
	}
	return uc.solRepo.GetSolicitationForUser(ctx, id, actor)
}

func (uc *SolicitationUseCase) CreateSolicitation(ctx context.Context, actor, requestKey string, req model.CreateSolicitationRequest) (*model.SolicitationItem, error) {
	if uc.solRepo == nil {
		return nil, domain.ErrUnavailable
	}

	description := strings.TrimSpace(req.Description)
	if description == "" {
		return nil, fmt.Errorf("descrição do serviço é obrigatória: %w", domain.ErrInvalidInput)
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

	categoryName := strings.TrimSpace(req.CategoryName)

	if req.ServiceID != "" && uc.serviceRepo != nil {
		service, err := uc.serviceRepo.GetServiceByID(ctx, req.ServiceID)
		if err != nil {
			return nil, err
		}
		if provider != "" && provider != service.BikkerID {
			return nil, fmt.Errorf("prestador divergente do serviço: %w", domain.ErrInvalidInput)
		}
		provider = service.BikkerID
		name = service.Name
		if categoryName == "" && service.Category != nil {
			categoryName = service.Category.Name
		}
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
	if preferred == "" {
		preferred = "O mais rápido possível"
	}

	photos := req.ClientPhotos
	if len(photos) == 0 {
		photos = req.Photos
	}
	if photos == nil {
		photos = []string{}
	}

	item := model.SolicitationItem{
		RequestKey:      requestKey,
		ClientID:        actor,
		ProviderID:      provider,
		ServiceID:       strings.TrimSpace(req.ServiceID),
		ServiceName:     name,
		ServiceTitle:    name,
		CategoryName:    categoryName,
		Description:     description,
		PreferredDate:   preferred,
		ScheduledDate:   preferred,
		LocationAddress: address,
		Address:         address,
		ContactName:     strings.TrimSpace(req.ContactName),
		ContactPhone:    strings.TrimSpace(req.ContactPhone),
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		Photos:          photos,
		ClientPhotos:    photos,
	}

	return uc.solRepo.CreateSolicitation(ctx, item)
}

func (uc *SolicitationUseCase) UpdateStatus(ctx context.Context, in model.SolicitationStatusUpdateInput, id string) (*model.SolicitationItem, error) {
	if uc.solRepo == nil {
		return nil, domain.ErrUnavailable
	}
	return uc.solRepo.UpdateSolicitationStatus(ctx, id, in)
}

func (uc *SolicitationUseCase) SubmitProposal(ctx context.Context, actor, id, requestKey string, price float64, note string, accept bool) (*model.SolicitationItem, error) {
	if uc.solRepo == nil {
		return nil, domain.ErrUnavailable
	}

	status := "BUDGET_RECEIVED"
	if accept {
		status = "ACTIVE"
	}

	in := model.SolicitationStatusUpdateInput{
		ActorID:             actor,
		RequestKey:          requestKey,
		Status:              status,
		CounterOfferPrice:   &price,
		CounterOfferMessage: &note,
	}

	return uc.solRepo.UpdateSolicitationStatus(ctx, id, in)
}

func (uc *SolicitationUseCase) SubmitCounterOffer(ctx context.Context, actor, id, requestKey string, price float64, note string, accept bool) (*model.SolicitationItem, error) {
	if uc.solRepo == nil {
		return nil, domain.ErrUnavailable
	}

	status := "NEGOTIATING"
	if accept {
		status = "ACTIVE"
	}

	in := model.SolicitationStatusUpdateInput{
		ActorID:             actor,
		RequestKey:          requestKey,
		Status:              status,
		CounterOfferPrice:   &price,
		CounterOfferMessage: &note,
	}

	return uc.solRepo.UpdateSolicitationStatus(ctx, id, in)
}

func (uc *SolicitationUseCase) AcceptCounterOffer(ctx context.Context, actor, id, requestKey string) (*model.SolicitationItem, error) {
	if uc.solRepo == nil {
		return nil, domain.ErrUnavailable
	}

	in := model.SolicitationStatusUpdateInput{
		ActorID:    actor,
		RequestKey: requestKey,
		Status:     "ACTIVE",
	}

	return uc.solRepo.UpdateSolicitationStatus(ctx, id, in)
}

func (uc *SolicitationUseCase) RejectSolicitation(ctx context.Context, actor, id, requestKey, reason string) (*model.SolicitationItem, error) {
	if uc.solRepo == nil {
		return nil, domain.ErrUnavailable
	}

	in := model.SolicitationStatusUpdateInput{
		ActorID:            actor,
		RequestKey:         requestKey,
		Status:             "REJECTED",
		CancellationReason: &reason,
	}

	return uc.solRepo.UpdateSolicitationStatus(ctx, id, in)
}

func (uc *SolicitationUseCase) GetProviderFeed(ctx context.Context, actor string) ([]domain.OpportunityLead, error) {
	if uc.solRepo == nil {
		return nil, domain.ErrUnavailable
	}
	return uc.solRepo.GetProviderFeed(ctx, actor)
}

func (uc *SolicitationUseCase) SubmitClientEvaluation(ctx context.Context, actor, id string, req domain.ClientEvaluationRequest) error {
	if uc.solRepo == nil {
		return domain.ErrUnavailable
	}
	return uc.solRepo.SubmitClientEvaluation(ctx, id, actor, req)
}
