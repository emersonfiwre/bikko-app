package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bikko-app/internal/domain"
	infraCtrl "bikko-app/internal/infrastructure/http/controller"
	"bikko-app/internal/usecase"

	"github.com/gin-gonic/gin"
)

type mockReviewRepo struct {
	createFn func(ctx context.Context, review *domain.Review) error
}

func (m *mockReviewRepo) CreateReviewInTx(ctx context.Context, review *domain.Review) error {
	if m.createFn != nil {
		return m.createFn(ctx, review)
	}
	review.ID = "review-test-id"
	review.CreatedAt = time.Now()
	return nil
}

func setupReviewTestRouter(repo domain.ReviewRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	ratingUC := usecase.NewRatingUseCase(repo)
	ctrl := infraCtrl.NewReviewController(ratingUC)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/reviews", ctrl.CreateReview)
		v1.POST("/solicitations/:id/review", ctrl.CreateSolicitationReview)
		v1.POST("/orders/:id/review", ctrl.CreateOrderReview)
	}

	return r
}

func TestReviewController_CreateReview(t *testing.T) {
	repo := &mockReviewRepo{}
	r := setupReviewTestRouter(repo)

	payload := map[string]interface{}{
		"order_id":    "order-123",
		"reviewer_id": "user-456",
		"rating":      5,
		"comment":     "Excelente trabalho!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/reviews", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReviewController_CreateSolicitationReview(t *testing.T) {
	repo := &mockReviewRepo{
		createFn: func(ctx context.Context, review *domain.Review) error {
			if review.OrderID != "sol_4" {
				t.Errorf("expected order_id sol_4, got %s", review.OrderID)
			}
			if review.Rating != 5 {
				t.Errorf("expected rating 5, got %d", review.Rating)
			}
			review.ID = "rev_sol_4"
			review.CreatedAt = time.Now()
			return nil
		},
	}
	r := setupReviewTestRouter(repo)

	payload := map[string]interface{}{
		"rating":  5,
		"comment": "Muito pontual e eficiente!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/solicitations/sol_4/review", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReviewController_CreateOrderReview(t *testing.T) {
	repo := &mockReviewRepo{
		createFn: func(ctx context.Context, review *domain.Review) error {
			if review.OrderID != "44444444-0000-0000-0000-000000000004" {
				t.Errorf("expected order_id 44444444-0000-0000-0000-000000000004, got %s", review.OrderID)
			}
			review.ID = "rev_order_4"
			review.CreatedAt = time.Now()
			return nil
		},
	}
	r := setupReviewTestRouter(repo)

	payload := map[string]interface{}{
		"rating":  4,
		"comment": "Bom atendimento",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/orders/44444444-0000-0000-0000-000000000004/review", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReviewController_InvalidRating(t *testing.T) {
	repo := &mockReviewRepo{}
	r := setupReviewTestRouter(repo)

	payload := map[string]interface{}{
		"rating":  6,
		"comment": "Rating invalido",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/solicitations/sol_4/review", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for rating 6, got %d", w.Code)
	}
}

func TestReviewController_OrderNotCompleted(t *testing.T) {
	repo := &mockReviewRepo{
		createFn: func(ctx context.Context, review *domain.Review) error {
			return domain.ErrOrderNotCompleted
		},
	}
	r := setupReviewTestRouter(repo)

	payload := map[string]interface{}{
		"rating":  5,
		"comment": "Tentando avaliar antes de concluir",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/solicitations/sol_1/review", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for uncompleted order, got %d", w.Code)
	}
}

func TestReviewController_AlreadyReviewed(t *testing.T) {
	repo := &mockReviewRepo{
		createFn: func(ctx context.Context, review *domain.Review) error {
			return domain.ErrAlreadyReviewed
		},
	}
	r := setupReviewTestRouter(repo)

	payload := map[string]interface{}{
		"rating":  5,
		"comment": "Segunda avaliação",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/solicitations/sol_4/review", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for already reviewed, got %d", w.Code)
	}
}
