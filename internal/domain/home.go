package domain

import "errors"

var (
	ErrNotFound          = errors.New("recurso não encontrado")
	ErrOrderNotCompleted = errors.New("avaliação só é permitida em ordens com status COMPLETED")
	ErrAlreadyReviewed   = errors.New("esta ordem já possui uma avaliação cadastrada")
	ErrInvalidRating     = errors.New("o rating deve ser um valor entre 1 e 5")
	ErrUnauthorized      = errors.New("usuário não autorizado")
)

type HomeItem struct {
	ID          string  `json:"id"`
	Title       string  `json:"title,omitempty"`
	Subtitle    string  `json:"subtitle,omitempty"`
	ImageURL    string  `json:"image_url,omitempty"`
	IconURL     string  `json:"icon_url,omitempty"`
	Name        string  `json:"name,omitempty"`
	Distance    string  `json:"distance,omitempty"`
	Rating      float64 `json:"rating,omitempty"`
	PriceAmount float64 `json:"price_amount,omitempty"`
	Badge       string  `json:"badge,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	ProviderName string `json:"provider_name,omitempty"`
	CategoryID  string  `json:"category_id,omitempty"`
	BikkerID    string  `json:"bikker_id,omitempty"`
}

type HomeCollection struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Type  string     `json:"type"`
	Items []HomeItem `json:"items"`
}

type HomeResponse struct {
	Collections []HomeCollection `json:"collections"`
}

type SearchItem struct {
	ID          string `json:"id"`
	Name        string `json:"title"`
	Type        string `json:"type"`
	ImageURL    string `json:"image_url"`
	Description string `json:"subtitle"`
}

type SearchResponse struct {
	Items []SearchItem `json:"items"`
}
