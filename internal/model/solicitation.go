package model

type SolicitationItem struct {
	ID                  string   `json:"id"`
	ServiceID           string   `json:"service_id"`
	ServiceName         string   `json:"service_name"`
	ProviderName        string   `json:"provider_name"`
	ProviderTitle       string   `json:"provider_title"`
	ProviderRating      string   `json:"provider_rating"`
	ProviderPhotoURL    string   `json:"provider_photo_url"`
	Status              string   `json:"status"`
	Price               float64  `json:"price"`
	ScheduledDate       string   `json:"scheduled_date"`
	Description         string   `json:"description"`
	CancellationReason  *string  `json:"cancellation_reason,omitempty"`
	CancelledBy         *string  `json:"cancelled_by,omitempty"`
	CounterOfferPrice   *float64 `json:"counter_offer_price,omitempty"`
	CounterOfferMessage *string  `json:"counter_offer_message,omitempty"`
	RenegotiationCount  int      `json:"renegotiation_count"`
	Photos              []string `json:"photos"`
	CreatedDate         string   `json:"created_date"`
}
