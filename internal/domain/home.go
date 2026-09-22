package domain

import "errors"

var (
	ErrNotFound          = errors.New("recurso não encontrado")
	ErrOrderNotCompleted = errors.New("avaliação só é permitida em ordens com status COMPLETED")
	ErrAlreadyReviewed   = errors.New("esta ordem já possui uma avaliação cadastrada")
	ErrInvalidRating     = errors.New("o rating deve ser um valor entre 1 e 5")
	ErrUnauthorized      = errors.New("usuário não autorizado")
)

type HomeCollection struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Items interface{} `json:"items"`
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
