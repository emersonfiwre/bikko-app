package domain

import "errors"

var (
	ErrNotFound          = errors.New("recurso não encontrado")
	ErrOrderNotCompleted = errors.New("avaliação só é permitida em ordens com status COMPLETED")
	ErrAlreadyReviewed   = errors.New("esta ordem já possui uma avaliação cadastrada")
	ErrInvalidRating     = errors.New("o rating deve ser um valor entre 1 e 5")
	ErrUnauthorized      = errors.New("usuário não autorizado")
)

type HomeResponse struct {
	Categories   []Category `json:"categories"`
	NearServices []Service  `json:"near_services"`
	Advice       []Service  `json:"advice"`
}

type SearchResponse struct {
	Services   []Service  `json:"services"`
	Categories []Category `json:"categories"`
}
