package controller

import (
	"bikko-app/internal/domain"
	"bikko-app/internal/model"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

type solicitationStub struct {
	model.SolicitationRepository
	actor     string
	createErr error
	command   model.SolicitationStatusUpdateInput
	getErr    error
}

func (s *solicitationStub) GetSolicitationsByUser(_ context.Context, actor string) ([]model.SolicitationItem, error) {
	s.actor = actor
	return []model.SolicitationItem{}, nil
}
func (s *solicitationStub) GetSolicitationForUser(_ context.Context, _ string, actor string) (*model.SolicitationItem, error) {
	s.actor = actor
	return nil, s.getErr
}
func (s *solicitationStub) CreateSolicitation(_ context.Context, item model.SolicitationItem) (*model.SolicitationItem, error) {
	s.actor = item.ClientID
	return &item, s.createErr
}
func (s *solicitationStub) UpdateSolicitationStatus(_ context.Context, _ string, in model.SolicitationStatusUpdateInput) (*model.SolicitationItem, error) {
	s.command = in
	return &model.SolicitationItem{ID: "persisted"}, nil
}
func solicitationRequest(ctrl *SolicitationController, actor, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if actor != "" {
		r.Use(func(c *gin.Context) { c.Set("userID", actor) })
	}
	r.GET("/solicitations", ctrl.GetSolicitations)
	r.GET("/solicitations/:id", ctrl.GetSolicitationByID)
	r.POST("/solicitations", ctrl.CreateSolicitation)
	r.PATCH("/solicitations/:id/status", ctrl.UpdateStatus)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func TestSolicitationsRequireIdentityAndPersistence(t *testing.T) {
	ctrl := NewSolicitationController(nil)
	w := solicitationRequest(ctrl, "", "GET", "/solicitations", "")
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	w = solicitationRequest(ctrl, "actor", "GET", "/solicitations", "")
	if w.Code != 503 {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestEmptyPersonalListStaysEmpty(t *testing.T) {
	repo := &solicitationStub{}
	ctrl := NewSolicitationController(nil)
	ctrl.SetPostgresRepository(repo)
	w := solicitationRequest(ctrl, "new-client", "GET", "/solicitations", "")
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" || repo.actor != "new-client" {
		t.Fatal(w.Code, w.Body.String(), repo.actor)
	}
}
func TestDetailUsesAuthenticatedScope(t *testing.T) {
	repo := &solicitationStub{getErr: domain.ErrNotFound}
	ctrl := NewSolicitationController(nil)
	ctrl.SetPostgresRepository(repo)
	w := solicitationRequest(ctrl, "outsider", "GET", "/solicitations/foreign", "")
	if w.Code != 404 || repo.actor != "outsider" {
		t.Fatal(w.Code, repo.actor)
	}
}
func TestPersistenceFailureDoesNotCreateMemoryOrder(t *testing.T) {
	repo := &solicitationStub{createErr: errors.New("database disconnected")}
	ctrl := NewSolicitationController(nil)
	ctrl.SetPostgresRepository(repo)
	w := solicitationRequest(ctrl, "real-actor", "POST", "/solicitations", `{"description":"Repair","provider_id":"provider"}`)
	if w.Code != 500 || repo.actor != "real-actor" {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestStatusIdentityCannotBeSpoofed(t *testing.T) {
	repo := &solicitationStub{}
	ctrl := NewSolicitationController(nil)
	ctrl.SetPostgresRepository(repo)
	w := solicitationRequest(ctrl, "actual-client", "PATCH", "/solicitations/id/status", `{"status":"NEGOTIATING","sender_id":"another-user","sender_role":"PROVIDER","counter_offer_price":50}`)
	if w.Code != 200 || repo.command.ActorID != "actual-client" {
		t.Fatal(w.Code, repo.command)
	}
}
