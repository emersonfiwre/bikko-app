package controller

import (
	"bikko-app/internal/domain"
	"bikko-app/internal/model"
	"context"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

type providerRepositoryStub struct {
	model.SolicitationRepository
	actor   string
	item    model.SolicitationItem
	command model.SolicitationStatusUpdateInput
	err     error
}

func (s *providerRepositoryStub) GetSolicitationForUser(_ context.Context, _ string, actor string) (*model.SolicitationItem, error) {
	s.actor = actor
	return &s.item, s.err
}
func (s *providerRepositoryStub) UpdateSolicitationStatus(_ context.Context, _ string, in model.SolicitationStatusUpdateInput) (*model.SolicitationItem, error) {
	s.command = in
	return &s.item, s.err
}
func providerRequest(ctrl *ProviderController, actor string, isProvider bool, action, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if actor != "" {
			c.Set("userID", actor)
		}
		c.Set("isBikker", isProvider)
	})
	r.POST("/provider/:id/proposal", ctrl.SubmitProposal)
	r.POST("/provider/:id/accept", ctrl.AcceptCounterOffer)
	req := httptest.NewRequest("POST", "/provider/foreign/"+action, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func TestProviderActionsRequireProfessionalRole(t *testing.T) {
	ctrl := NewProviderController()
	w := providerRequest(ctrl, "client", false, "proposal", `{"price":50}`)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	w = providerRequest(ctrl, "", true, "proposal", `{"price":50}`)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
}
func TestProviderDoesNotReassignAnotherProvidersOrder(t *testing.T) {
	repo := &providerRepositoryStub{item: model.SolicitationItem{ProviderID: "owner"}}
	ctrl := NewProviderControllerWithDeps(repo, nil, nil)
	w := providerRequest(ctrl, "outsider", true, "proposal", `{"price":50}`)
	if w.Code != 403 || repo.command.ActorID != "" {
		t.Fatal(w.Code, repo.command)
	}
}
func TestProviderAcceptUsesPersistedProposal(t *testing.T) {
	repo := &providerRepositoryStub{item: model.SolicitationItem{ProviderID: "owner", Photos: []string{}}}
	ctrl := NewProviderControllerWithDeps(repo, nil, nil)
	w := providerRequest(ctrl, "owner", true, "accept", `{"price":999999}`)
	if w.Code != 200 || repo.command.Status != "ACTIVE" || repo.command.CounterOfferPrice != nil {
		t.Fatal(w.Code, repo.command)
	}
}
func TestProviderDatabaseFailureNeverReturnsFakeDetails(t *testing.T) {
	repo := &providerRepositoryStub{err: domain.ErrUnavailable}
	ctrl := NewProviderControllerWithDeps(repo, nil, nil)
	w := providerRequest(ctrl, "owner", true, "proposal", `{"price":50}`)
	if w.Code != 503 {
		t.Fatal(w.Code, w.Body.String())
	}
}
