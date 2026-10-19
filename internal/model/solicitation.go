package model

type CounterOfferItem struct {
	ID             string   `json:"id"`
	Round          int      `json:"round"`
	Author         string   `json:"author"` // "CLIENT" or "PROVIDER"
	IsFromClient   bool     `json:"is_from_client"`
	Price          float64  `json:"price"`
	FormattedPrice string   `json:"formatted_price,omitempty"`
	Reason         string   `json:"reason"`
	Photos         []string `json:"photos,omitempty"`
}

type SolicitationItem struct {
	ID                  string             `json:"id"`
	ClientID            string             `json:"client_id,omitempty"`
	ClientName          string             `json:"client_name,omitempty"`
	ClientPhone         string             `json:"client_phone,omitempty"`
	ClientAccountPhone  string             `json:"client_account_phone,omitempty"`
	ContactName         string             `json:"contact_name,omitempty"`
	ContactPhone        string             `json:"contact_phone,omitempty"`
	AccountPhone        string             `json:"account_phone,omitempty"`
	ClientRating        string             `json:"client_rating,omitempty"`
	ClientTotalRatings  int                `json:"client_total_ratings,omitempty"`
	ClientPhotoURL      string             `json:"client_photo_url,omitempty"`
	ProviderID          string             `json:"provider_id,omitempty"`
	ServiceID           string             `json:"service_id"`
	ServiceName         string             `json:"service_name"`
	ServiceTitle        string             `json:"service_title,omitempty"`
	Category            string             `json:"category,omitempty"`
	CategoryName        string             `json:"category_name,omitempty"`
	Neighborhood        string             `json:"neighborhood,omitempty"`
	Location            string             `json:"location,omitempty"`
	Address             string             `json:"address,omitempty"`
	LocationAddress     string             `json:"location_address,omitempty"`
	HouseNumber         string             `json:"house_number,omitempty"`
	ProviderName        string             `json:"provider_name"`
	ProviderTitle       string             `json:"provider_title"`
	ProviderRating      string             `json:"provider_rating"`
	ProviderPhotoURL    string             `json:"provider_photo_url"`
	Status              string             `json:"status"`
	Price               float64            `json:"price"`
	ScheduledDate       string             `json:"scheduled_date"`
	PreferredDate       string             `json:"preferred_date,omitempty"`
	Description         string             `json:"description"`
	InitialDescription  string             `json:"initial_description,omitempty"`
	TechnicalReport     *string            `json:"technical_report,omitempty"`
	CancellationReason  *string            `json:"cancellation_reason,omitempty"`
	CancelledBy         *string            `json:"cancelled_by,omitempty"`
	CounterOfferPrice   *float64           `json:"counter_offer_price,omitempty"`
	CounterOfferMessage *string            `json:"counter_offer_message,omitempty"`
	RenegotiationCount  int                `json:"renegotiation_count"`
	CounterOffers       []CounterOfferItem `json:"counter_offers,omitempty"`
	CounterOfferHistory []CounterOfferItem `json:"counter_offer_history,omitempty"`
	NegotiationHistory []CounterOfferItem `json:"negotiation_history,omitempty"`
	Photos              []string           `json:"photos"`
	ClientPhotos        []string           `json:"client_photos,omitempty"`
	CreatedDate         string             `json:"created_date"`
	IsReviewed          bool               `json:"is_reviewed,omitempty"`
	IsClientEvaluated   bool               `json:"is_client_evaluated,omitempty"`
	ClientEvaluated     bool               `json:"client_evaluated,omitempty"`
	BikkerEvaluated     bool               `json:"bikker_evaluated,omitempty"`
	ClientRatingGiven   int                `json:"client_rating_given,omitempty"`
}

type CreateSolicitationRequest struct {
	ServiceID       string   `json:"service_id"`
	BikkerID        string   `json:"bikker_id"`
	ProviderID      string   `json:"provider_id"`
	ServiceName     string   `json:"service_name"`
	Title           string   `json:"title"`
	ServiceTitle    string   `json:"service_title"`
	CategoryName    string   `json:"category_name"`
	Description     string   `json:"description" binding:"required"`
	ScheduledDate   string   `json:"scheduled_date"`
	PreferredDate   string   `json:"preferred_date"`
	Location        string   `json:"location"`
	Address         string   `json:"address"`
	LocationAddress string   `json:"location_address"`
	HouseNumber     string   `json:"house_number"`
	ContactName     string   `json:"contact_name"`
	ContactPhone    string   `json:"contact_phone"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	Photos          []string `json:"photos"`
	ClientPhotos    []string `json:"client_photos"`
}

type SolicitationStore interface {
	GetInMemorySolicitationByID(id string) *SolicitationItem
	SyncSolicitationInMemory(item SolicitationItem)
	ApplyProviderProposalInMemory(id, providerID string, price float64, note string, accepted bool) *SolicitationItem
	ApplyProviderCounterOfferInMemory(id, providerID string, counterPrice float64, reason string) *SolicitationItem
	ApplyProviderRejectInMemory(id, providerID, reason string) *SolicitationItem
}

