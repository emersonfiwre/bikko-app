package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bikko-app/internal/domain"

	"github.com/gin-gonic/gin"
)

type fakeBikkerControllerRepo struct {
	bikkerByID map[string]*domain.BikkerProfileResponse
	allBikkers []domain.BikkerProfileResponse
	getIDErr   error
	getAllErr  error
}

func (f *fakeBikkerControllerRepo) GetByID(ctx context.Context, id string) (*domain.BikkerProfileResponse, error) {
	if f.getIDErr != nil {
		return nil, f.getIDErr
	}
	if b, ok := f.bikkerByID[id]; ok {
		return b, nil
	}
	return nil, errors.New("bikker not found")
}

func (f *fakeBikkerControllerRepo) GetNearBikkers(ctx context.Context, lat, lon float64, limit int) ([]domain.NearBikkerItem, error) {
	return nil, nil
}

func (f *fakeBikkerControllerRepo) GetAll(ctx context.Context, limit int) ([]domain.BikkerProfileResponse, error) {
	if f.getAllErr != nil {
		return nil, f.getAllErr
	}
	return f.allBikkers, nil
}

func TestBikkerController_GetBikkerByID(t *testing.T) {
	repo := &fakeBikkerControllerRepo{
		bikkerByID: map[string]*domain.BikkerProfileResponse{
			"b1": {
				ID:         "b1",
				Name:       "Carlos Mendes",
				Profession: "Técnico de Climatização",
			},
		},
	}
	ctrl := NewBikkerController(repo)

	router := gin.New()
	router.GET("/bikkers/:id", ctrl.GetBikkerByID)

	t.Run("returns bikker profile when found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/bikkers/b1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp domain.BikkerProfileResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.ID != "b1" || resp.Name != "Carlos Mendes" {
			t.Errorf("unexpected profile: %+v", resp)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/bikkers/unknown", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestBikkerController_GetBikkers(t *testing.T) {
	t.Run("success returns list of bikkers", func(t *testing.T) {
		repo := &fakeBikkerControllerRepo{
			allBikkers: []domain.BikkerProfileResponse{
				{ID: "b1", Name: "Carlos"},
				{ID: "b2", Name: "Marcos"},
			},
		}
		ctrl := NewBikkerController(repo)
		router := gin.New()
		router.GET("/bikkers", ctrl.GetBikkers)

		req, _ := http.NewRequest(http.MethodGet, "/bikkers", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var bikkers []domain.BikkerProfileResponse
		_ = json.Unmarshal(w.Body.Bytes(), &bikkers)
		if len(bikkers) != 2 {
			t.Errorf("expected 2 bikkers, got %d", len(bikkers))
		}
	})

	t.Run("returns 500 when repository fails", func(t *testing.T) {
		repo := &fakeBikkerControllerRepo{
			getAllErr: errors.New("db error"),
		}
		ctrl := NewBikkerController(repo)
		router := gin.New()
		router.GET("/bikkers", ctrl.GetBikkers)

		req, _ := http.NewRequest(http.MethodGet, "/bikkers", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}
