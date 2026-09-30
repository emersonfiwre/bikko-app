package domain

import (
	"encoding/json"
	"testing"
)

func TestHomeDomain_Errors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrNotFound", ErrNotFound, "recurso não encontrado"},
		{"ErrOrderNotCompleted", ErrOrderNotCompleted, "avaliação só é permitida em ordens com status COMPLETED"},
		{"ErrAlreadyReviewed", ErrAlreadyReviewed, "esta ordem já possui uma avaliação cadastrada"},
		{"ErrInvalidRating", ErrInvalidRating, "o rating deve ser um valor entre 1 e 5"},
		{"ErrUnauthorized", ErrUnauthorized, "usuário não autorizado"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestHomeDomain_JSONSerialization(t *testing.T) {
	t.Run("HomeResponse and HomeCollection", func(t *testing.T) {
		home := HomeResponse{
			Collections: []HomeCollection{
				{
					ID:   "cat_col",
					Name: "Categorias",
					Type: "Category",
					Items: []Category{
						{ID: "c1", Name: "Pintura"},
					},
				},
			},
		}

		data, err := json.Marshal(home)
		if err != nil {
			t.Fatalf("failed to marshal HomeResponse: %v", err)
		}

		var unmarshaled HomeResponse
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal HomeResponse: %v", err)
		}

		if len(unmarshaled.Collections) != 1 {
			t.Fatalf("expected 1 collection, got %d", len(unmarshaled.Collections))
		}
		if unmarshaled.Collections[0].ID != "cat_col" || unmarshaled.Collections[0].Type != "Category" {
			t.Errorf("collection mismatch: %+v", unmarshaled.Collections[0])
		}
	})

	t.Run("SearchResponse and SearchItem", func(t *testing.T) {
		search := SearchResponse{
			Items: []SearchItem{
				{
					ID:          "s1",
					Name:        "Pintor",
					Type:        "service",
					ImageURL:    "https://example.com/p.jpg",
					Description: "Serviço de pintura",
				},
			},
		}

		data, err := json.Marshal(search)
		if err != nil {
			t.Fatalf("failed to marshal SearchResponse: %v", err)
		}

		var unmarshaled SearchResponse
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal SearchResponse: %v", err)
		}

		if len(unmarshaled.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(unmarshaled.Items))
		}
		item := unmarshaled.Items[0]
		if item.ID != "s1" || item.Name != "Pintor" || item.Type != "service" {
			t.Errorf("search item mismatch: %+v", item)
		}
	})
}
