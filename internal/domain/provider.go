package domain

import "time"

type LeadCounterOffer struct {
	ID             string  `json:"id"`
	Round          int     `json:"round"`
	Author         string  `json:"author"` // "CLIENT" or "PROVIDER"
	IsFromClient   bool    `json:"is_from_client"`
	Price          float64 `json:"price"`
	FormattedPrice string  `json:"formatted_price,omitempty"`
	Reason         string  `json:"reason"`
}

type OpportunityLead struct {
	ID                  string             `json:"id"`
	Title               string             `json:"title"`
	ServiceTitle        string             `json:"service_title,omitempty"`
	Description         string             `json:"description"`
	Category            string             `json:"category"`
	Urgency             string             `json:"urgency"` // "EMERGENCY" or "SCHEDULED"
	Photos              []string           `json:"photos"`
	ClientPhotos        []string           `json:"client_photos,omitempty"`
	PriceType           string             `json:"price_type"` // "BUDGET_TO_NEGOTIATE"
	EstimatedValue      string             `json:"estimated_value,omitempty"`
	DistanceKm          float64            `json:"distance_km"`
	Neighborhood        string             `json:"neighborhood"`
	Address             string             `json:"address,omitempty"`
	LocationAddress     string             `json:"location_address,omitempty"`
	ClientName          string             `json:"client_name"`
	ContactName         string             `json:"contact_name,omitempty"`
	ClientPhone         string             `json:"client_phone,omitempty"`
	ContactPhone        string             `json:"contact_phone,omitempty"`
	AccountPhone        string             `json:"account_phone,omitempty"`
	ClientAccountPhone  string             `json:"client_account_phone,omitempty"`
	ClientRating        string             `json:"client_rating,omitempty"`
	ClientVerified      bool               `json:"client_verified"`
	CreatedAt           time.Time          `json:"created_at"`
	ScheduledDate       string             `json:"scheduled_date,omitempty"`
	PreferredDate       string             `json:"preferred_date,omitempty"`
	CounterOffers       []LeadCounterOffer `json:"counter_offers,omitempty"`
	CounterOfferHistory []LeadCounterOffer `json:"counter_offer_history,omitempty"`
	NegotiationHistory  []LeadCounterOffer `json:"negotiation_history,omitempty"`
}

type PortfolioItem struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Category       string    `json:"category"`
	Description    string    `json:"description,omitempty"`
	ImageUrl       string    `json:"image_url"`
	Photos         []string  `json:"photos,omitempty"`
	IsActive       *bool     `json:"is_active,omitempty"`
	CompletedAt    time.Time `json:"completed_at"`
	ReviewsAverage float64   `json:"reviews_average"`
	TotalReviews   int       `json:"total_reviews"`
}

type PortfolioCreateRequest struct {
	ID           string   `json:"id,omitempty"`
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
	IsActive     *bool    `json:"is_active"`
}

type ProviderProfileUpdateRequest struct {
	Profession      string   `json:"profession"`
	ExperienceYears string   `json:"experience_years"`
	Description     *string  `json:"description"`
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
	Stars           int      `json:"stars"`
	Tags            []string `json:"tags"`
	Comment         string   `json:"comment"`
}
