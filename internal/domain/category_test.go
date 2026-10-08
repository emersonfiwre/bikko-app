package domain

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

type memoryCategoryRepository struct {
	categories []Category
}

func newMemoryCategoryRepository(cats []Category) CategoryRepository {
	return &memoryCategoryRepository{categories: cats}
}

func (m *memoryCategoryRepository) GetAllActive(ctx context.Context) ([]Category, error) {
	var active []Category
	for _, c := range m.categories {
		if c.IsActive {
			active = append(active, c)
		}
	}
	return active, nil
}

func (m *memoryCategoryRepository) GetByID(ctx context.Context, id string) (*Category, error) {
	for _, c := range m.categories {
		if c.ID == id || strings.HasPrefix(c.ID, id) {
			copy := c
			return &copy, nil
		}
	}
	return nil, ErrNotFound
}

func (m *memoryCategoryRepository) SearchCategories(ctx context.Context, query string) ([]Category, error) {
	q := strings.ToLower(query)
	var matches []Category
	for _, c := range m.categories {
		if c.IsActive && (strings.Contains(strings.ToLower(c.Name), q) || strings.Contains(strings.ToLower(c.Description), q)) {
			matches = append(matches, c)
		}
	}
	return matches, nil
}

func TestCategoryDomain_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	cat := Category{
		ID:          "cat_123",
		Name:        "Pintura Residencial",
		IconURL:     "https://example.com/icons/paint.png",
		Description: "Serviços profissionais de pintura de paredes e tetos",
		IsActive:    true,
		CreatedAt:   now,
		Services: []Service{
			{
				ID:   "srv_1",
				Name: "Pintura de Fachada",
			},
		},
	}

	data, err := json.Marshal(cat)
	if err != nil {
		t.Fatalf("failed to marshal Category: %v", err)
	}

	var unmarshaled Category
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal Category: %v", err)
	}

	if unmarshaled.ID != cat.ID {
		t.Errorf("expected ID '%s', got '%s'", cat.ID, unmarshaled.ID)
	}
	if unmarshaled.Name != cat.Name {
		t.Errorf("expected Name '%s', got '%s'", cat.Name, unmarshaled.Name)
	}
	if unmarshaled.IconURL != cat.IconURL {
		t.Errorf("expected IconURL '%s', got '%s'", cat.IconURL, unmarshaled.IconURL)
	}
	if unmarshaled.Description != cat.Description {
		t.Errorf("expected Description '%s', got '%s'", cat.Description, unmarshaled.Description)
	}
	if unmarshaled.IsActive != cat.IsActive {
		t.Errorf("expected IsActive %v, got %v", cat.IsActive, unmarshaled.IsActive)
	}
	if len(unmarshaled.Services) != 1 || unmarshaled.Services[0].ID != "srv_1" {
		t.Errorf("expected 1 nested service, got %+v", unmarshaled.Services)
	}
}

func TestCategoryRepositoryContract(t *testing.T) {
	ctx := context.Background()
	cats := []Category{
		{ID: "c1", Name: "Elétrica", Description: "Instalação de tomadas e quadros", IsActive: true},
		{ID: "c2", Name: "Hidráulica", Description: "Reparo de canos e torneiras", IsActive: true},
		{ID: "c3", Name: "Alvenaria", Description: "Construção e reforma", IsActive: false},
	}
	repo := newMemoryCategoryRepository(cats)

	t.Run("GetAllActive returns only active categories", func(t *testing.T) {
		active, err := repo.GetAllActive(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(active) != 2 {
			t.Fatalf("expected 2 active categories, got %d", len(active))
		}
		for _, c := range active {
			if !c.IsActive {
				t.Errorf("found inactive category in active list: %+v", c)
			}
		}
	})

	t.Run("GetByID returns category when found", func(t *testing.T) {
		cat, err := repo.GetByID(ctx, "c1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cat.Name != "Elétrica" {
			t.Errorf("expected 'Elétrica', got '%s'", cat.Name)
		}
	})

	t.Run("GetByID returns ErrNotFound when not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "non_existent")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("SearchCategories matches name or description", func(t *testing.T) {
		matches, err := repo.SearchCategories(ctx, "canos")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(matches) != 1 || matches[0].ID != "c2" {
			t.Errorf("expected match c2 for 'canos', got %+v", matches)
		}

		// Inactive category should not appear in search
		alvMatches, err := repo.SearchCategories(ctx, "alvenaria")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(alvMatches) != 0 {
			t.Errorf("expected 0 matches for inactive category, got %d", len(alvMatches))
		}
	})
}
