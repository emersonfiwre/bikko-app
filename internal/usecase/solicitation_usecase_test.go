package usecase

import (
	"context"
	"testing"

	"bikko-app/internal/domain"
	"bikko-app/internal/model"
)

type fakeSolicitationRepo struct {
	items                 []model.SolicitationItem
	singleItem            *model.SolicitationItem
	leads                 []domain.OpportunityLead
	err                   error
	lastStatusInput       model.SolicitationStatusUpdateInput
	lastCreated           model.SolicitationItem
	evaluationCalled      bool
	lastEvaluationRequest domain.ClientEvaluationRequest
}

func (f *fakeSolicitationRepo) GetSolicitationsByUser(ctx context.Context, actor string) ([]model.SolicitationItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func (f *fakeSolicitationRepo) GetSolicitationForUser(ctx context.Context, id, actor string) (*model.SolicitationItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.singleItem != nil {
		return f.singleItem, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeSolicitationRepo) CreateSolicitation(ctx context.Context, item model.SolicitationItem) (*model.SolicitationItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastCreated = item
	res := item
	res.ID = "new_solicitation_uuid"
	return &res, nil
}

func (f *fakeSolicitationRepo) UpdateSolicitationStatus(ctx context.Context, id string, in model.SolicitationStatusUpdateInput) (*model.SolicitationItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastStatusInput = in
	res := model.SolicitationItem{
		ID:     id,
		Status: in.Status,
	}
	if in.CounterOfferPrice != nil {
		res.Price = *in.CounterOfferPrice
	}
	return &res, nil
}

func (f *fakeSolicitationRepo) GetProviderFeed(ctx context.Context, actor string) ([]domain.OpportunityLead, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.leads, nil
}

func (f *fakeSolicitationRepo) SubmitClientEvaluation(ctx context.Context, id, actor string, req domain.ClientEvaluationRequest) error {
	if f.err != nil {
		return f.err
	}
	f.evaluationCalled = true
	f.lastEvaluationRequest = req
	return nil
}

type solicitationMockServiceRepo struct {
	service *domain.Service
	err     error
}

func (f *solicitationMockServiceRepo) GetByID(ctx context.Context, id string) (*domain.Service, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.service, nil
}
func (f *solicitationMockServiceRepo) GetServiceByID(ctx context.Context, id string) (*domain.Service, error) {
	return f.GetByID(ctx, id)
}
func (f *solicitationMockServiceRepo) GetServicesNear(ctx context.Context, lat, lon, radius float64, limit int) ([]domain.Service, error) {
	return nil, nil
}
func (f *solicitationMockServiceRepo) GetFeaturedAdviceServices(ctx context.Context, limit int) ([]domain.Service, error) {
	return nil, nil
}
func (f *solicitationMockServiceRepo) GetActiveServicesByBikkerID(ctx context.Context, id string) ([]domain.Service, error) {
	return nil, nil
}
func (f *solicitationMockServiceRepo) GetServicesByBikkerID(ctx context.Context, id string) ([]domain.Service, error) {
	return nil, nil
}
func (f *solicitationMockServiceRepo) GetServicesByCategoryID(ctx context.Context, id string) ([]domain.Service, error) {
	return nil, nil
}
func (f *solicitationMockServiceRepo) SearchServices(ctx context.Context, query string) ([]domain.Service, error) {
	return nil, nil
}
func (f *solicitationMockServiceRepo) CreateService(ctx context.Context, s *domain.Service) (*domain.Service, error) {
	return nil, nil
}
func (f *solicitationMockServiceRepo) UpdateService(ctx context.Context, s *domain.Service) error {
	return nil
}
func (f *solicitationMockServiceRepo) DeleteService(ctx context.Context, id, bikkerID string) error {
	return nil
}
func (f *solicitationMockServiceRepo) AddServicePhoto(ctx context.Context, serviceID, photoURL string, order int) error {
	return nil
}

func TestSolicitationUseCase_CreateSolicitation_Validation(t *testing.T) {
	fakeRepo := &fakeSolicitationRepo{}
	fakeSvc := &solicitationMockServiceRepo{
		service: &domain.Service{
			ID:       "svc_1",
			BikkerID: "provider_1",
			Name:     "Instalação Elétrica",
			Category: &domain.Category{Name: "Elétrica"},
		},
	}
	uc := NewSolicitationUseCase(fakeRepo, fakeSvc)

	// Missing description
	_, err := uc.CreateSolicitation(context.Background(), "client_1", "key_1", model.CreateSolicitationRequest{
		ServiceID: "svc_1",
	})
	if err == nil {
		t.Fatal("expected error when description is missing")
	}

	// Mismatched provider
	_, err = uc.CreateSolicitation(context.Background(), "client_1", "key_1", model.CreateSolicitationRequest{
		ServiceID:   "svc_1",
		ProviderID:  "other_provider",
		Description: "Instalar tomadas",
	})
	if err == nil {
		t.Fatal("expected error when provider diverges from service")
	}

	// Valid creation
	item, err := uc.CreateSolicitation(context.Background(), "client_1", "key_1", model.CreateSolicitationRequest{
		ServiceID:   "svc_1",
		Description: "Instalar tomadas na sala",
		Location:    "Rua dos Pinheiros",
		HouseNumber: "100",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ID != "new_solicitation_uuid" {
		t.Fatalf("unexpected ID: %s", item.ID)
	}
	if fakeRepo.lastCreated.ProviderID != "provider_1" {
		t.Fatalf("expected provider_1, got %s", fakeRepo.lastCreated.ProviderID)
	}
	if fakeRepo.lastCreated.LocationAddress != "Rua dos Pinheiros, 100" {
		t.Fatalf("expected address formatted, got %s", fakeRepo.lastCreated.LocationAddress)
	}
}

func TestSolicitationUseCase_Negotiations(t *testing.T) {
	fakeRepo := &fakeSolicitationRepo{}
	uc := NewSolicitationUseCase(fakeRepo, nil)

	// Submit proposal
	res, err := uc.SubmitProposal(context.Background(), "provider_1", "quote_1", "key_p1", 150.0, "Orçamento padrão", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "BUDGET_RECEIVED" || fakeRepo.lastStatusInput.Status != "BUDGET_RECEIVED" {
		t.Fatalf("expected BUDGET_RECEIVED, got %s", res.Status)
	}

	// Submit counter offer
	res, err = uc.SubmitCounterOffer(context.Background(), "client_1", "quote_1", "key_c1", 120.0, "Contraproposta", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "NEGOTIATING" || fakeRepo.lastStatusInput.Status != "NEGOTIATING" {
		t.Fatalf("expected NEGOTIATING, got %s", res.Status)
	}

	// Accept counter offer
	res, err = uc.AcceptCounterOffer(context.Background(), "provider_1", "quote_1", "key_a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "ACTIVE" || fakeRepo.lastStatusInput.Status != "ACTIVE" {
		t.Fatalf("expected ACTIVE, got %s", res.Status)
	}

	// Reject
	res, err = uc.RejectSolicitation(context.Background(), "provider_1", "quote_1", "key_r1", "Agenda lotada")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "REJECTED" || fakeRepo.lastStatusInput.Status != "REJECTED" {
		t.Fatalf("expected REJECTED, got %s", res.Status)
	}
}

func TestSolicitationUseCase_SubmitEvaluation(t *testing.T) {
	fakeRepo := &fakeSolicitationRepo{}
	uc := NewSolicitationUseCase(fakeRepo, nil)

	req := domain.ClientEvaluationRequest{
		Stars:           5,
		Comment:         "Excelente serviço",
		TechnicalReport: "Tudo nos conformes",
	}
	err := uc.SubmitClientEvaluation(context.Background(), "provider_1", "quote_1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fakeRepo.evaluationCalled || fakeRepo.lastEvaluationRequest.Stars != 5 {
		t.Fatalf("evaluation not called properly")
	}
}
