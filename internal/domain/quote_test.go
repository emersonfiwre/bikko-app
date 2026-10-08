package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestQuoteDomain_EnumsAndSerialization(t *testing.T) {
	t.Run("QuoteStatus enum values", func(t *testing.T) {
		statuses := []QuoteStatus{
			QuoteStatusPending,
			QuoteStatusNegotiating,
			QuoteStatusAccepted,
			QuoteStatusRejected,
			QuoteStatusExpired,
		}
		expected := []string{"PENDING", "NEGOTIATING", "ACCEPTED", "REJECTED", "EXPIRED"}

		for i, s := range statuses {
			if string(s) != expected[i] {
				t.Errorf("expected '%s', got '%s'", expected[i], string(s))
			}
		}
	})

	t.Run("OrderStatus enum values", func(t *testing.T) {
		statuses := []OrderStatus{
			OrderStatusScheduled,
			OrderStatusInProgress,
			OrderStatusCompleted,
			OrderStatusCancelled,
		}
		expected := []string{"SCHEDULED", "IN_PROGRESS", "COMPLETED", "CANCELLED"}

		for i, s := range statuses {
			if string(s) != expected[i] {
				t.Errorf("expected '%s', got '%s'", expected[i], string(s))
			}
		}
	})

	t.Run("QuoteRequest JSON serialization", func(t *testing.T) {
		now := time.Now()
		req := QuoteRequest{
			ID:                 "qr_1",
			CustomerID:         "cust_1",
			BikkerID:           "bikker_1",
			ServiceID:          "srv_1",
			Status:             QuoteStatusPending,
			InitialDescription: "Vazamento no banheiro",
			DesiredDate:        now,
			LocationAddress:    "Rua das Flores, 123",
			Latitude:           -23.5505,
			Longitude:          -46.6333,
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal QuoteRequest: %v", err)
		}

		var unmarshaled QuoteRequest
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal QuoteRequest: %v", err)
		}

		if unmarshaled.ID != "qr_1" || unmarshaled.Status != QuoteStatusPending {
			t.Errorf("mismatch in unmarshaled QuoteRequest: %+v", unmarshaled)
		}
	})

	t.Run("QuoteNegotiation JSON serialization with counter offer", func(t *testing.T) {
		price := 250.0
		neg := QuoteNegotiation{
			ID:             "neg_1",
			QuoteRequestID: "qr_1",
			SenderID:       "bikker_1",
			Message:        "Consigo fazer por 250",
			ProposedPrice:  &price,
			Photos:         []string{"https://example.com/p1.jpg"},
			IsCounterOffer: true,
			CreatedAt:      time.Now(),
		}

		data, err := json.Marshal(neg)
		if err != nil {
			t.Fatalf("failed to marshal QuoteNegotiation: %v", err)
		}

		var unmarshaled QuoteNegotiation
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal QuoteNegotiation: %v", err)
		}

		if unmarshaled.ProposedPrice == nil || *unmarshaled.ProposedPrice != 250.0 {
			t.Errorf("expected proposed price 250.0, got %+v", unmarshaled.ProposedPrice)
		}
		if !unmarshaled.IsCounterOffer {
			t.Errorf("expected is_counter_offer true")
		}
	})

	t.Run("Order JSON serialization", func(t *testing.T) {
		now := time.Now()
		order := Order{
			ID:             "ord_1",
			QuoteRequestID: "qr_1",
			CustomerID:     "cust_1",
			BikkerID:       "bikker_1",
			ServiceID:      "srv_1",
			Status:         OrderStatusCompleted,
			FinalPrice:     320.0,
			ScheduledDate:  now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		data, err := json.Marshal(order)
		if err != nil {
			t.Fatalf("failed to marshal Order: %v", err)
		}

		var unmarshaled Order
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal Order: %v", err)
		}

		if unmarshaled.FinalPrice != 320.0 || unmarshaled.Status != OrderStatusCompleted {
			t.Errorf("mismatch in unmarshaled Order: %+v", unmarshaled)
		}
	})
}
