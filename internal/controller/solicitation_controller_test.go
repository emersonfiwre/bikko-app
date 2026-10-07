package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"bikko-app/internal/domain"
	"bikko-app/internal/model"

	"github.com/gin-gonic/gin"
)

type fakeSolicitationServiceRepo struct {
	service *domain.Service
	err     error
}

func (f *fakeSolicitationServiceRepo) GetByID(ctx context.Context, id string) (*domain.Service, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.service, nil
}

func (f *fakeSolicitationServiceRepo) GetServiceByID(ctx context.Context, id string) (*domain.Service, error) {
	return f.GetByID(ctx, id)
}

func (f *fakeSolicitationServiceRepo) GetServicesNear(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]domain.Service, error) {
	return nil, nil
}

func (f *fakeSolicitationServiceRepo) GetFeaturedAdviceServices(ctx context.Context, limit int) ([]domain.Service, error) {
	return nil, nil
}

func (f *fakeSolicitationServiceRepo) GetActiveServicesByBikkerID(ctx context.Context, bikkerID string) ([]domain.Service, error) {
	return nil, nil
}

func (f *fakeSolicitationServiceRepo) GetServicesByCategoryID(ctx context.Context, categoryID string) ([]domain.Service, error) {
	return nil, nil
}

func (f *fakeSolicitationServiceRepo) SearchServices(ctx context.Context, query string) ([]domain.Service, error) {
	return nil, nil
}

func setupSolicitationRouter(ctrl *SolicitationController) *gin.Engine {
	r := gin.New()
	r.GET("/solicitations", ctrl.GetSolicitations)
	r.GET("/solicitations/:id", ctrl.GetSolicitationByID)
	r.POST("/solicitations", ctrl.CreateSolicitation)
	r.POST("/solicitations/:id/status", ctrl.UpdateStatus)
	return r
}

func TestSolicitationController_InitialStateAndListing(t *testing.T) {
	ctrl := NewSolicitationController(nil)
	router := setupSolicitationRouter(ctrl)

	t.Run("GetSolicitations returns initial solicitations", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/solicitations", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var items []model.SolicitationItem
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(items) != 4 {
			t.Errorf("expected 4 default solicitations, got %d", len(items))
		}
	})

	t.Run("GetSolicitationByID returns existing solicitation", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/solicitations/sol_1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var item model.SolicitationItem
		_ = json.Unmarshal(w.Body.Bytes(), &item)
		if item.ID != "sol_1" || item.ServiceName != "Instalação de Ar Condicionado" {
			t.Errorf("unexpected solicitation item: %+v", item)
		}
	})

	t.Run("GetSolicitationByID returns 404 for unknown ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/solicitations/non_existent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})
}

func TestSolicitationController_CreateSolicitation(t *testing.T) {
	t.Run("returns 400 when description is missing", func(t *testing.T) {
		ctrl := NewSolicitationController(nil)
		router := setupSolicitationRouter(ctrl)

		payload := []byte(`{"service_name":"Troca de Piso"}`)
		req, _ := http.NewRequest(http.MethodPost, "/solicitations", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("creates solicitation with explicit service name and prepends to list", func(t *testing.T) {
		ctrl := NewSolicitationController(nil)
		router := setupSolicitationRouter(ctrl)

		payload := map[string]interface{}{
			"service_name":   "Instalação de Torneira",
			"description":    "Torneira da pia da cozinha vazando",
			"scheduled_date": "29 de Outubro às 14:00",
			"photos":         []string{"https://example.com/pia.jpg"},
		}
		data, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/solicitations", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", w.Code)
		}

		var created model.SolicitationItem
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if created.ServiceName != "Instalação de Torneira" {
			t.Errorf("expected ServiceName 'Instalação de Torneira', got '%s'", created.ServiceName)
		}
		if created.Status != "PENDING_BUDGET" {
			t.Errorf("expected Status 'PENDING_BUDGET', got '%s'", created.Status)
		}
		if created.Price != 0.0 {
			t.Errorf("expected initial price 0.0, got %f", created.Price)
		}
		if created.RenegotiationCount != 0 {
			t.Errorf("expected RenegotiationCount 0, got %d", created.RenegotiationCount)
		}
		if len(created.Photos) != 1 {
			t.Errorf("expected 1 photo, got %d", len(created.Photos))
		}

		// Verify it was prepended to the controller's list
		activeList := ctrl.GetActiveSolicitations()
		if len(activeList) == 0 || activeList[0].ID != created.ID {
			t.Errorf("expected newly created item to be first in active list, got %+v", activeList)
		}
	})

	t.Run("resolves service name and provider info from serviceRepo when service_name is omitted", func(t *testing.T) {
		fakeRepo := &fakeSolicitationServiceRepo{
			service: &domain.Service{
				ID:               "srv_abc",
				Name:             "Conserto de Fechadura",
				ProviderName:     "Roberto Chaveiro",
				ProviderTitle:    "Chaveiro 24h",
				ProviderPhotoURL: "https://example.com/roberto.jpg",
				ReviewsAverage:   4.95,
			},
		}
		ctrl := NewSolicitationController(fakeRepo)
		router := setupSolicitationRouter(ctrl)

		payload := map[string]interface{}{
			"service_id":  "srv_abc",
			"description": "Fechadura emperrada na porta principal",
		}
		data, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/solicitations", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", w.Code)
		}

		var created model.SolicitationItem
		_ = json.Unmarshal(w.Body.Bytes(), &created)

		if created.ServiceName != "Conserto de Fechadura" {
			t.Errorf("expected ServiceName 'Conserto de Fechadura', got '%s'", created.ServiceName)
		}
		if created.ProviderName != "Roberto Chaveiro" {
			t.Errorf("expected ProviderName 'Roberto Chaveiro', got '%s'", created.ProviderName)
		}
		if created.ProviderTitle != "Chaveiro 24h" {
			t.Errorf("expected ProviderTitle 'Chaveiro 24h', got '%s'", created.ProviderTitle)
		}
		if created.ProviderRating != "5.0" {
			t.Errorf("expected ProviderRating '5.0', got '%s'", created.ProviderRating)
		}
	})

	t.Run("defaults to 'Solicitação de Serviço' when service not found in repo", func(t *testing.T) {
		ctrl := NewSolicitationController(nil)
		router := setupSolicitationRouter(ctrl)

		payload := map[string]interface{}{
			"description": "Preciso de ajuda urgente",
		}
		data, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/solicitations", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", w.Code)
		}

		var created model.SolicitationItem
		_ = json.Unmarshal(w.Body.Bytes(), &created)

		if created.ServiceName != "Solicitação de Serviço" {
			t.Errorf("expected default ServiceName, got '%s'", created.ServiceName)
		}
	})
}

func TestSolicitationController_UpdateStatus(t *testing.T) {
	t.Run("returns 400 when status is missing", func(t *testing.T) {
		ctrl := NewSolicitationController(nil)
		router := setupSolicitationRouter(ctrl)

		payload := []byte(`{"cancelled_by":"USER"}`)
		req, _ := http.NewRequest(http.MethodPost, "/solicitations/sol_1/status", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 404 for unknown solicitation", func(t *testing.T) {
		ctrl := NewSolicitationController(nil)
		router := setupSolicitationRouter(ctrl)

		payload := []byte(`{"status":"ACTIVE"}`)
		req, _ := http.NewRequest(http.MethodPost, "/solicitations/unknown_id/status", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("updates status and handles counter offer", func(t *testing.T) {
		ctrl := NewSolicitationController(nil)
		router := setupSolicitationRouter(ctrl)

		price := 400.0
		msg := "Faço por 400 se for amanhã de manhã"
		payload := map[string]interface{}{
			"status":                "NEGOTIATING",
			"counter_offer_price":   price,
			"counter_offer_message": msg,
		}
		data, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/solicitations/sol_1/status", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var updated model.SolicitationItem
		_ = json.Unmarshal(w.Body.Bytes(), &updated)

		if updated.Status != "NEGOTIATING" {
			t.Errorf("expected Status 'NEGOTIATING', got '%s'", updated.Status)
		}
		if updated.Price != 400.0 {
			t.Errorf("expected Price 400.0, got %f", updated.Price)
		}
		if updated.RenegotiationCount != 1 {
			t.Errorf("expected RenegotiationCount 1, got %d", updated.RenegotiationCount)
		}
		if updated.CounterOfferMessage == nil || *updated.CounterOfferMessage != msg {
			t.Errorf("expected CounterOfferMessage '%s', got %+v", msg, updated.CounterOfferMessage)
		}
	})

	t.Run("cancels automatically when renegotiation limit reaches 5", func(t *testing.T) {
		ctrl := NewSolicitationController(nil)
		router := setupSolicitationRouter(ctrl)

		price := 300.0
		// Send 5 counter-offers
		for i := 1; i <= 5; i++ {
			price += 10.0
			payload := map[string]interface{}{
				"status":              "NEGOTIATING",
				"counter_offer_price": price,
			}
			data, _ := json.Marshal(payload)
			req, _ := http.NewRequest(http.MethodPost, "/solicitations/sol_1/status", bytes.NewBuffer(data))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("iteration %d: expected 200, got %d", i, w.Code)
			}

			if i == 5 {
				var finalItem model.SolicitationItem
				_ = json.Unmarshal(w.Body.Bytes(), &finalItem)

				if finalItem.Status != "CANCELLED" {
					t.Errorf("expected Status 'CANCELLED' on 5th counter-offer, got '%s'", finalItem.Status)
				}
				if finalItem.CancelledBy == nil || *finalItem.CancelledBy != "MAX_RENEGOTIATIONS" {
					t.Errorf("expected CancelledBy 'MAX_RENEGOTIATIONS', got %+v", finalItem.CancelledBy)
				}
				if finalItem.CancellationReason == nil || *finalItem.CancellationReason != "Limite máximo de 5 contrapropostas atingido sem acordo." {
					t.Errorf("unexpected CancellationReason: %+v", finalItem.CancellationReason)
				}
			}
		}
	})
}

func TestSolicitationController_GetActiveSolicitations(t *testing.T) {
	t.Run("returns only active/pending/budget solicitations up to 3 items", func(t *testing.T) {
		ctrl := NewSolicitationController(nil)

		// Initial list has: sol_1 (BUDGET_RECEIVED), sol_2 (ACTIVE), sol_3 (CANCELLED), sol_4 (COMPLETED)
		// Active items should be: sol_1 and sol_2
		actives := ctrl.GetActiveSolicitations()
		if len(actives) != 2 {
			t.Fatalf("expected 2 active solicitations, got %d", len(actives))
		}
		if actives[0].ID != "sol_1" || actives[1].ID != "sol_2" {
			t.Errorf("unexpected active solicitations: %+v", actives)
		}

		// Add 2 more active solicitations to test max 3 limit
		router := setupSolicitationRouter(ctrl)
		for _, name := range []string{"Serviço Novo 1", "Serviço Novo 2"} {
			data, _ := json.Marshal(map[string]string{
				"service_name": name,
				"description":  "Desc",
			})
			req, _ := http.NewRequest(http.MethodPost, "/solicitations", bytes.NewBuffer(data))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}

		limitedActives := ctrl.GetActiveSolicitations()
		if len(limitedActives) != 3 {
			t.Errorf("expected exactly 3 items due to limit, got %d", len(limitedActives))
		}
	})
}

type fakePushNotificationCall struct {
	userID string
	title  string
	body   string
	data   map[string]string
}

type fakePushServiceForTest struct {
	mu    sync.Mutex
	calls []fakePushNotificationCall
}

func (f *fakePushServiceForTest) SendNotification(ctx context.Context, userID string, title, body string, data map[string]string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fakePushNotificationCall{
		userID: userID,
		title:  title,
		body:   body,
		data:   data,
	})
	return nil
}

func TestSolicitationController_PushNotifications(t *testing.T) {
	t.Run("sends push to provider when client sends counter-offer", func(t *testing.T) {
		fakePush := &fakePushServiceForTest{}
		ctrl := NewSolicitationController(nil, fakePush)
		router := setupSolicitationRouter(ctrl)

		price := 320.0
		senderRole := "CLIENT"
		payload := map[string]interface{}{
			"status":              "NEGOTIATING",
			"counter_offer_price": price,
			"sender_role":         senderRole,
		}
		data, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/solicitations/sol_1/status", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		if len(fakePush.calls) != 1 {
			t.Fatalf("expected 1 push notification call, got %d", len(fakePush.calls))
		}

		call := fakePush.calls[0]
		// sol_1 provider is 11111111-0000-0000-0000-000000000009
		if call.userID != "11111111-0000-0000-0000-000000000009" {
			t.Errorf("expected provider ID recipient, got %s", call.userID)
		}
		if call.title != "Nova Contraproposta!" {
			t.Errorf("expected title 'Nova Contraproposta!', got '%s'", call.title)
		}
		if call.body != "Você recebeu uma nova oferta para a solicitação." {
			t.Errorf("expected body 'Você recebeu uma nova oferta para a solicitação.', got '%s'", call.body)
		}
		if call.data["solicitation_id"] != "sol_1" || call.data["type"] != "counter_offer" {
			t.Errorf("unexpected data payload: %+v", call.data)
		}
	})

	t.Run("sends push to client when provider sends counter-offer", func(t *testing.T) {
		fakePush := &fakePushServiceForTest{}
		ctrl := NewSolicitationController(nil)
		ctrl.SetPushService(fakePush)
		router := setupSolicitationRouter(ctrl)

		price := 380.0
		senderRole := "PROVIDER"
		payload := map[string]interface{}{
			"status":              "NEGOTIATING",
			"counter_offer_price": price,
			"sender_role":         senderRole,
		}
		data, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/solicitations/sol_1/status", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		if len(fakePush.calls) != 1 {
			t.Fatalf("expected 1 push notification call, got %d", len(fakePush.calls))
		}

		call := fakePush.calls[0]
		// sol_1 client is 11111111-0000-0000-0000-000000000099
		if call.userID != "11111111-0000-0000-0000-000000000099" {
			t.Errorf("expected client ID recipient, got %s", call.userID)
		}
		if call.title != "Nova Contraproposta!" {
			t.Errorf("expected title 'Nova Contraproposta!', got '%s'", call.title)
		}
		if call.data["solicitation_id"] != "sol_1" || call.data["type"] != "counter_offer" {
			t.Errorf("unexpected data payload: %+v", call.data)
		}
	})

	t.Run("sends push when order is ACCEPTED", func(t *testing.T) {
		fakePush := &fakePushServiceForTest{}
		ctrl := NewSolicitationController(nil, fakePush)
		router := setupSolicitationRouter(ctrl)

		senderRole := "CLIENT"
		payload := map[string]interface{}{
			"status":      "ACCEPTED",
			"sender_role": senderRole,
		}
		data, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/solicitations/sol_1/status", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		if len(fakePush.calls) != 1 {
			t.Fatalf("expected 1 push notification call, got %d", len(fakePush.calls))
		}

		call := fakePush.calls[0]
		if call.userID != "11111111-0000-0000-0000-000000000009" {
			t.Errorf("expected provider recipient, got %s", call.userID)
		}
		if call.title != "Serviço Fechado!" {
			t.Errorf("expected title 'Serviço Fechado!', got '%s'", call.title)
		}
		if call.data["solicitation_id"] != "sol_1" || call.data["type"] != "accepted" {
			t.Errorf("unexpected data payload: %+v", call.data)
		}
	})
}
