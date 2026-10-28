package controller

import (
	"bikko-app/internal/domain"
	"bikko-app/internal/model"
	"bikko-app/internal/usecase"
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
	evalID  string
	evalReq domain.ClientEvaluationRequest
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
func (s *providerRepositoryStub) SubmitClientEvaluation(_ context.Context, id, actor string, req domain.ClientEvaluationRequest) error {
	s.evalID = id
	s.actor = actor
	s.evalReq = req
	return s.err
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

func providerDetailsRequest(ctrl *ProviderController, actor string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", actor)
		c.Set("isBikker", true)
	})
	r.GET("/provider/:id", ctrl.GetSolicitationDetails)
	req := httptest.NewRequest("GET", "/provider/solicitation-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestProviderDetailsUsesInjectedUseCaseWithoutRepositoryField(t *testing.T) {
	repo := &providerRepositoryStub{item: model.SolicitationItem{
		ID:         "solicitation-1",
		ProviderID: "provider",
		Status:     "PENDING",
	}}
	ctrl := NewProviderControllerWithUseCase(usecase.NewSolicitationUseCase(repo, nil), nil, nil)

	w := providerDetailsRequest(ctrl, "provider")
	if w.Code != 200 {
		t.Fatalf("expected details through injected use case, got %d: %s", w.Code, w.Body.String())
	}
	if repo.actor != "provider" {
		t.Fatalf("expected authenticated provider to reach repository through use case, got %q", repo.actor)
	}
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

func TestProviderEvaluationRequiresAndForwardsIdempotencyKey(t *testing.T) {
	repo := &providerRepositoryStub{}
	ctrl := NewProviderControllerWithDeps(repo, nil, nil)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", "provider")
		c.Set("isBikker", true)
	})
	r.POST("/provider/evaluations", ctrl.SubmitClientEvaluation)

	withoutKey := httptest.NewRequest("POST", "/provider/evaluations", strings.NewReader(`{"solicitation_id":"quote-1","stars":5}`))
	withoutKey.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, withoutKey)
	if w.Code != 400 || repo.evalID != "" {
		t.Fatalf("evaluation without key: code=%d called=%q", w.Code, repo.evalID)
	}

	withKey := httptest.NewRequest("POST", "/provider/evaluations", strings.NewReader(`{"solicitation_id":"quote-1","stars":5}`))
	withKey.Header.Set("Content-Type", "application/json")
	withKey.Header.Set("Idempotency-Key", "provider-evaluation-once")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, withKey)
	if w.Code != 200 || repo.evalID != "quote-1" || repo.evalReq.RequestKey != "provider-evaluation-once" {
		t.Fatalf("evaluation key was not forwarded: code=%d id=%q key=%q", w.Code, repo.evalID, repo.evalReq.RequestKey)
	}
}
