package repository

import (
	"context"
	"strings"
	"testing"

	"bikko-app/internal/model"
)

type fakeSolicitationProvider struct {
	items []model.SolicitationItem
}

func (f *fakeSolicitationProvider) GetActiveSolicitations() []model.SolicitationItem {
	return f.items
}

func TestMockProfileRepository_GetProfileByID(t *testing.T) {
	t.Run("returns default profile when solicitation provider is nil", func(t *testing.T) {
		repo := NewMockProfileRepository(nil)
		ctx := context.Background()

		profile, err := repo.GetProfileByID(ctx, "user_custom_1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if profile == nil {
			t.Fatal("expected profile, got nil")
		}
		if profile.ID != "user_custom_1" {
			t.Errorf("expected ID 'user_custom_1', got '%s'", profile.ID)
		}
		if profile.Name != "Emerson Torres" {
			t.Errorf("expected Name 'Emerson Torres', got '%s'", profile.Name)
		}
		if len(profile.Orders) != 0 {
			t.Errorf("expected 0 orders, got %d", len(profile.Orders))
		}
		if profile.Rating != "4.9" {
			t.Errorf("expected rating '4.9', got '%s'", profile.Rating)
		}
		if profile.TotalRatings != 18 {
			t.Errorf("expected total ratings 18, got %d", profile.TotalRatings)
		}
	})

	t.Run("correctly maps orders from solicitation provider with different statuses", func(t *testing.T) {
		provider := &fakeSolicitationProvider{
			items: []model.SolicitationItem{
				{
					ID:               "sol_pending",
					ServiceName:      "Instalação Ar",
					Status:           "PENDING_BUDGET",
					Price:            250.0,
					ProviderRating:   "4.8",
					ProviderName:     "Carlos Silva",
					ProviderPhotoURL: "https://example.com/carlos.jpg",
					Photos:           []string{"https://example.com/photo1.jpg"},
					Description:      "Instalação simples",
				},
				{
					ID:               "sol_budget",
					ServiceName:      "Pintura Sala",
					Status:           "BUDGET_RECEIVED",
					Price:            350.50,
					ProviderRating:   "5.0",
					ProviderName:     "Maria Oliveira",
					ProviderPhotoURL: "https://example.com/maria.jpg",
					Photos:           []string{},
					Description:      "Pintura 2 quartos",
				},
				{
					ID:               "sol_active",
					ServiceName:      "Reparo Elétrico",
					Status:           "ACTIVE",
					Price:            120.0,
					ProviderRating:   "4.6",
					ProviderName:     "João Souza",
					ProviderPhotoURL: "https://example.com/joao.jpg",
					Photos:           []string{"https://example.com/reparo.jpg"},
					Description:      "Troca de fiação",
				},
				{
					ID:               "sol_custom",
					ServiceName:      "Faxina",
					Status:           "SCHEDULED",
					ScheduledDate:    "28 de Outubro às 10:00",
					Price:            200.0,
					ProviderRating:   "4.9",
					ProviderName:     "Ana Dias",
					ProviderPhotoURL: "https://example.com/ana.jpg",
					Photos:           nil,
					Description:      "Faxina pesada",
				},
			},
		}

		repo := NewMockProfileRepository(provider)
		ctx := context.Background()

		profile, err := repo.GetProfileByID(ctx, "user_999")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(profile.Orders) != 4 {
			t.Fatalf("expected 4 orders, got %d", len(profile.Orders))
		}

		// Check order 0: PENDING_BUDGET
		o0 := profile.Orders[0]
		if o0.ID != "sol_pending" {
			t.Errorf("expected ID sol_pending, got %s", o0.ID)
		}
		if o0.Distance != "Aguardando Orçamento" {
			t.Errorf("expected 'Aguardando Orçamento', got '%s'", o0.Distance)
		}
		if o0.Thumbnail != "https://example.com/photo1.jpg" {
			t.Errorf("expected thumbnail from photos[0], got '%s'", o0.Thumbnail)
		}
		if o0.Category == nil || o0.Category.Name != "Carlos Silva" {
			t.Errorf("expected category provider name 'Carlos Silva', got %+v", o0.Category)
		}

		// Check order 1: BUDGET_RECEIVED
		o1 := profile.Orders[1]
		if !strings.Contains(o1.Distance, "Orçamento Recebido · R$ 350.50") {
			t.Errorf("expected 'Orçamento Recebido · R$ 350.50', got '%s'", o1.Distance)
		}
		if o1.Thumbnail != "https://example.com/maria.jpg" {
			t.Errorf("expected fallback thumbnail to ProviderPhotoURL, got '%s'", o1.Thumbnail)
		}

		// Check order 2: ACTIVE
		o2 := profile.Orders[2]
		if o2.Distance != "Em Andamento" {
			t.Errorf("expected 'Em Andamento', got '%s'", o2.Distance)
		}

		// Check order 3: Other status -> ScheduledDate
		o3 := profile.Orders[3]
		if o3.Distance != "28 de Outubro às 10:00" {
			t.Errorf("expected '28 de Outubro às 10:00', got '%s'", o3.Distance)
		}
	})
}

func TestMockProfileRepository_DeleteAccount(t *testing.T) {
	repo := NewMockProfileRepository(nil)
	err := repo.DeleteAccount(context.Background(), "user_123")
	if err != nil {
		t.Errorf("expected nil error on DeleteAccount, got %v", err)
	}
}
