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

type PortfolioCreateRequest struct {
	Title        string   `json:"title"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	CategoryID   string   `json:"category_id"`
	Description  string   `json:"description"`
	ImageURL     string   `json:"image_url"`
	ThumbnailURL string   `json:"thumbnail_url"`
	Photos       []string `json:"photos"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
}

type ProviderProfileUpdateRequest struct {
	Profession      string   `json:"profession"`
	ExperienceYears string   `json:"experience_years"`
	Description     string   `json:"description"`
	LocationName    string   `json:"location_name"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	IsActive        *bool    `json:"is_active"`
}

type ProposalRequest struct {
	Price              float64 `json:"price"`
	Note               string  `json:"note"`
	Accepted           bool    `json:"accepted"`
	AcceptCounterOffer bool    `json:"accept_counter_offer"`
}

type CounterOfferRequest struct {
	CounterPrice       float64 `json:"counter_price"`
	Price              float64 `json:"price"`
	Reason             string  `json:"reason"`
	Note               string  `json:"note"`
	Accepted           bool    `json:"accepted"`
	AcceptCounterOffer bool    `json:"accept_counter_offer"`
}

type RejectRequest struct {
	ReasonChip         string `json:"reason_chip"`
	Reason             string `json:"reason"`
	CancellationReason string `json:"cancellation_reason"`
	Note               string `json:"note"`
}

type ClientEvaluationRequest struct {
	SolicitationID  string   `json:"solicitation_id"`
	TechnicalReport string   `json:"technical_report"`
	Stars           int      `json:"stars" binding:"required,min=1,max=5"`
	Tags            []string `json:"tags"`
	Comment         string   `json:"comment"`
}
