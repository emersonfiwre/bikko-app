package repository

import (
	"bikko-app/internal/model"
	"context"
	"errors"
	"testing"
)

type scopedOrders struct {
	items    []model.SolicitationItem
	err      error
	calledID string
}

func (s *scopedOrders) GetSolicitationsByUser(ctx context.Context, id string) ([]model.SolicitationItem, error) {
	s.calledID = id
	return s.items, s.err
}
func TestProfileRepositoryKeepsEmptyAccountEmpty(t *testing.T) {
	source := &scopedOrders{items: []model.SolicitationItem{}}
	profile, err := NewProfileRepository(source).GetProfileByID(context.Background(), "lucas")
	if err != nil || source.calledID != "lucas" || len(profile.Orders) != 0 || profile.Name != "" {
		t.Fatal(profile, err)
	}
}
func TestProfileRepositoryPropagatesFailure(t *testing.T) {
	want := errors.New("database down")
	source := &scopedOrders{err: want}
	if _, err := NewProfileRepository(source).GetProfileByID(context.Background(), "lucas"); !errors.Is(err, want) {
		t.Fatal(err)
	}
	if _, err := NewProfileRepository(source).GetProfileByID(context.Background(), ""); err == nil {
		t.Fatal("accepted empty actor")
	}
}
func TestProfileRepositoryMapsPersistedOrder(t *testing.T) {
	source := &scopedOrders{items: []model.SolicitationItem{{ID: "one", ServiceName: "Pintura", Status: "ACTIVE", ProviderID: "rafael"}}}
	profile, err := NewProfileRepository(source).GetProfileByID(context.Background(), "lucas")
	if err != nil || len(profile.Orders) != 1 || profile.Orders[0].ID != "one" || profile.Orders[0].BikkerID != "rafael" {
		t.Fatal(profile, err)
	}
}
