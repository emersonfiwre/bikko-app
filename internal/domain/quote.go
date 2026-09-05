package domain

import (
	"context"
	"time"
)

type QuoteStatus string

const (
	QuoteStatusPending     QuoteStatus = "PENDING"
	QuoteStatusNegotiating QuoteStatus = "NEGOTIATING"
	QuoteStatusAccepted    QuoteStatus = "ACCEPTED"
	QuoteStatusRejected    QuoteStatus = "REJECTED"
	QuoteStatusExpired     QuoteStatus = "EXPIRED"
)

type OrderStatus string

const (
	OrderStatusScheduled  OrderStatus = "SCHEDULED"
	OrderStatusInProgress OrderStatus = "IN_PROGRESS"
	OrderStatusCompleted  OrderStatus = "COMPLETED"
	OrderStatusCancelled  OrderStatus = "CANCELLED"
)

type QuoteRequest struct {
	ID                 string      `json:"id"`
	CustomerID         string      `json:"customer_id"`
	BikkerID           string      `json:"bikker_id"`
	ServiceID          string      `json:"service_id"`
	Status             QuoteStatus `json:"status"`
	InitialDescription string      `json:"initial_description"`
	DesiredDate        time.Time   `json:"desired_date"`
	LocationAddress    string      `json:"location_address"`
	Latitude           float64     `json:"latitude"`
	Longitude          float64     `json:"longitude"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

type QuoteNegotiation struct {
	ID             string    `json:"id"`
	QuoteRequestID string    `json:"quote_request_id"`
	SenderID       string    `json:"sender_id"`
	Message        string    `json:"message"`
	ProposedPrice  *float64  `json:"proposed_price,omitempty"`
	Photos         []string  `json:"photos"`
	IsCounterOffer bool      `json:"is_counter_offer"`
	CreatedAt      time.Time `json:"created_at"`
}

type Order struct {
	ID                 string      `json:"id"`
	QuoteRequestID     string      `json:"quote_request_id"`
	CustomerID         string      `json:"customer_id"`
	BikkerID           string      `json:"bikker_id"`
	ServiceID          string      `json:"service_id"`
	Status             OrderStatus `json:"status"`
	FinalPrice         float64     `json:"final_price"`
	ScheduledDate      time.Time   `json:"scheduled_date"`
	CancellationReason string      `json:"cancellation_reason,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

type OrderRepository interface {
	GetByID(ctx context.Context, id string) (*Order, error)
}
