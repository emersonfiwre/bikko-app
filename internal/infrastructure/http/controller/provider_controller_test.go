package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	appCtrl "bikko-app/internal/controller"
	"bikko-app/internal/model"

	"github.com/gin-gonic/gin"
)

type fakeProviderPushCall struct {
	userID string
	title  string
	body   string
	data   map[string]string
}

type fakeProviderPushService struct {
	mu    sync.Mutex
	calls []fakeProviderPushCall
}

func (f *fakeProviderPushService) SendNotification(ctx context.Context, userID string, title, body string, data map[string]string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fakeProviderPushCall{
		userID: userID,
		title:  title,
		body:   body,
		data:   data,
	})
	return nil
}

func TestProviderController_ProposalAndCounterOfferAndRejection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("SubmitProposal updates status to BUDGET_RECEIVED and sends push to Client", func(t *testing.T) {
		pushSvc := &fakeProviderPushService{}
		solCtrl := appCtrl.NewSolicitationController(nil, pushSvc)
		provCtrl := NewProviderControllerWithDeps(nil, nil, nil, pushSvc)
		provCtrl.SetSolicitationStore(solCtrl)

		r := gin.New()
		r.POST("/provider/solicitations/:id/proposal", provCtrl.SubmitProposal)

		body, _ := json.Marshal(map[string]interface{}{
			"price": 420.0,
			"note":  "Inclui peças e mão de obra.",
		})
		req, _ := http.NewRequest(http.MethodPost, "/provider/solicitations/sol_1/proposal", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		updated := solCtrl.GetInMemorySolicitationByID("sol_1")
		if updated == nil || updated.Status != "BUDGET_RECEIVED" || updated.Price != 420.0 {
			t.Fatalf("expected sol_1 status BUDGET_RECEIVED and price 420.0, got %+v", updated)
		}

		if len(pushSvc.calls) != 1 {
			t.Fatalf("expected 1 push call, got %d", len(pushSvc.calls))
		}
		if pushSvc.calls[0].title != "Novo Orçamento Recebido! 💬" {
			t.Errorf("unexpected push title: %s", pushSvc.calls[0].title)
		}
		if pushSvc.calls[0].data["solicitation_id"] != "sol_1" {
			t.Errorf("expected solicitation_id sol_1, got %+v", pushSvc.calls[0].data)
		}
	})

	t.Run("SubmitProposal accepting client counter-offer sets status to ACTIVE and sends push to Client", func(t *testing.T) {
		pushSvc := &fakeProviderPushService{}
		solCtrl := appCtrl.NewSolicitationController(nil)
		provCtrl := NewProviderControllerWithDeps(nil, nil, nil, pushSvc)
		provCtrl.SetSolicitationStore(solCtrl)

		r := gin.New()
		r.POST("/solicitations/:id/status", solCtrl.UpdateStatus)
		r.POST("/provider/solicitations/:id/proposal", provCtrl.SubmitProposal)

		// 1. Client sends a counter-offer of R$ 300.00
		clientOffer, _ := json.Marshal(map[string]interface{}{
			"status":                "PENDING_BUDGET",
			"counter_offer_price":   300.0,
			"counter_offer_message": "Consegue fazer por 300?",
			"sender_role":           "CLIENT",
		})
		req1, _ := http.NewRequest(http.MethodPost, "/solicitations/sol_1/status", bytes.NewBuffer(clientOffer))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)

		// 2. Bikker accepts the client's counter-offer by submitting matching price 300.0
		acceptBody, _ := json.Marshal(map[string]interface{}{
			"price": 300.0,
			"note":  "Fechado por 300!",
		})
		req2, _ := http.NewRequest(http.MethodPost, "/provider/solicitations/sol_1/proposal", bytes.NewBuffer(acceptBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)

		if w2.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w2.Code, w2.Body.String())
		}

		updated := solCtrl.GetInMemorySolicitationByID("sol_1")
		if updated == nil || updated.Status != "ACTIVE" || updated.Price != 300.0 {
			t.Fatalf("expected sol_1 status ACTIVE and price 300.0, got %+v", updated)
		}

		if len(pushSvc.calls) != 1 {
			t.Fatalf("expected 1 push call, got %d", len(pushSvc.calls))
		}
		if pushSvc.calls[0].title != "Proposta Aceita pelo Profissional! 🎉" {
			t.Errorf("expected title 'Proposta Aceita pelo Profissional! 🎉', got '%s'", pushSvc.calls[0].title)
		}
		if pushSvc.calls[0].data["solicitation_id"] != "sol_1" {
			t.Errorf("expected solicitation_id sol_1, got %+v", pushSvc.calls[0].data)
		}
	})

	t.Run("SubmitCounterOffer updates status to BUDGET_RECEIVED and sends push to Client", func(t *testing.T) {
		pushSvc := &fakeProviderPushService{}
		solCtrl := appCtrl.NewSolicitationController(nil)
		provCtrl := NewProviderControllerWithDeps(nil, nil, nil, pushSvc)
		provCtrl.SetSolicitationStore(solCtrl)

		r := gin.New()
		r.POST("/provider/solicitations/:id/counter-offer", provCtrl.SubmitCounterOffer)

		body, _ := json.Marshal(map[string]interface{}{
			"counter_price": 330.0,
			"reason":        "Consigo chegar em 330 com material incluso.",
		})
		req, _ := http.NewRequest(http.MethodPost, "/provider/solicitations/sol_1/counter-offer", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		updated := solCtrl.GetInMemorySolicitationByID("sol_1")
		if updated == nil || updated.Status != "BUDGET_RECEIVED" || updated.Price != 330.0 {
			t.Fatalf("expected sol_1 status BUDGET_RECEIVED and price 330.0, got %+v", updated)
		}

		if len(pushSvc.calls) != 1 || pushSvc.calls[0].title != "Nova Contraproposta do Profissional! 💬" {
			t.Errorf("unexpected push calls: %+v", pushSvc.calls)
		}
	})

	t.Run("RejectSolicitation updates status to CANCELLED with cancelled_by PROVIDER and reason in store", func(t *testing.T) {
		pushSvc := &fakeProviderPushService{}
		solCtrl := appCtrl.NewSolicitationController(nil)
		provCtrl := NewProviderControllerWithDeps(nil, nil, nil, pushSvc)
		provCtrl.SetSolicitationStore(solCtrl)

		r := gin.New()
		r.POST("/provider/solicitations/:id/reject", provCtrl.RejectSolicitation)
		r.GET("/solicitations", solCtrl.GetSolicitations)

		body, _ := json.Marshal(map[string]interface{}{
			"reason": "Indisponível para atendimento nesta região",
		})
		req, _ := http.NewRequest(http.MethodPost, "/provider/solicitations/sol_1/reject", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		updated := solCtrl.GetInMemorySolicitationByID("sol_1")
		if updated == nil {
			t.Fatalf("expected sol_1 in memory")
		}
		if updated.Status != "CANCELLED" {
			t.Errorf("expected status CANCELLED, got %s", updated.Status)
		}
		if updated.CancelledBy == nil || *updated.CancelledBy != "PROVIDER" {
			t.Errorf("expected CancelledBy PROVIDER, got %+v", updated.CancelledBy)
		}
		if updated.CancellationReason == nil || *updated.CancellationReason != "Indisponível para atendimento nesta região" {
			t.Errorf("expected CancellationReason 'Indisponível para atendimento nesta região', got %+v", updated.CancellationReason)
		}

		// Verify it appears in GetSolicitations order history with CANCELLED status and reason
		getReq, _ := http.NewRequest(http.MethodGet, "/solicitations", nil)
		getW := httptest.NewRecorder()
		r.ServeHTTP(getW, getReq)

		var list []model.SolicitationItem
		_ = json.Unmarshal(getW.Body.Bytes(), &list)
		found := false
		for _, item := range list {
			if item.ID == "sol_1" {
				found = true
				if item.Status != "CANCELLED" || item.CancelledBy == nil || *item.CancelledBy != "PROVIDER" {
					t.Errorf("expected rejected sol_1 in history with PROVIDER cancellation, got %+v", item)
				}
			}
		}
		if !found {
			t.Errorf("expected sol_1 to be present in order history")
		}
	})
}
