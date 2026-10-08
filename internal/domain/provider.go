package domain

import "time"

type OpportunityLead struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Category       string    `json:"category"`
	Urgency        string    `json:"urgency"` // "EMERGENCY" or "SCHEDULED"
	Photos         []string  `json:"photos"`
	PriceType      string    `json:"price_type"` // "BUDGET_TO_NEGOTIATE"
	DistanceKm     float64   `json:"distance_km"`
	Neighborhood   string    `json:"neighborhood"`
	ClientName     string    `json:"client_name"`
	ClientVerified bool      `json:"client_verified"`
	CreatedAt      time.Time `json:"created_at"`
}

type PortfolioItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Category    string    `json:"category"`
	ImageUrl    string    `json:"image_url"`
	CompletedAt time.Time `json:"completed_at"`
}

type ProposalRequest struct {
	Price float64 `json:"price" binding:"required"`
	Note  string  `json:"note"`
}

type CounterOfferRequest struct {
	CounterPrice float64 `json:"counter_price" binding:"required"`
	Reason       string  `json:"reason"`
}

type RejectRequest struct {
	ReasonChip string `json:"reason_chip" binding:"required"`
	Note       string `json:"note"`
}

type ClientEvaluationRequest struct {
	Stars   int      `json:"stars" binding:"required,min=1,max=5"`
	Tags    []string `json:"tags"`
	Comment string   `json:"comment"`
}
