package controller

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"bikko-app/internal/domain"
	"bikko-app/internal/model"

	"github.com/gin-gonic/gin"
)

type SolicitationController struct {
	mu            sync.RWMutex
	solicitations []model.SolicitationItem
	serviceRepo   domain.ServiceRepository
}

func NewSolicitationController(serviceRepo domain.ServiceRepository) *SolicitationController {
	reason := "O provedor não tinha disponibilidade para a data/hora solicitada."
	cancelledByProvider := "PROVIDER"

	return &SolicitationController{
		serviceRepo: serviceRepo,
		solicitations: []model.SolicitationItem{
			{
				ID:               "sol_1",
				ServiceID:        "22222222-0000-0000-0000-000000000012",
				ServiceName:      "Instalação de Ar Condicionado",
				ProviderName:     "Carlos Mendes",
				ProviderTitle:    "Técnico de Climatização",
				ProviderRating:   "4.9",
				ProviderPhotoURL: "https://images.pexels.com/photos/220453/pexels-photo-220453.jpeg",
				Status:           "BUDGET_RECEIVED",
				Price:            350.0,
				ScheduledDate:    "24 de Junho às 14:00",
				Description:      "Instalação de split de 12000 BTUs na sala de estar. Necessário furar a parede e passar tubulação.",
				Photos: []string{
					"https://images.pexels.com/photos/2310904/pexels-photo-2310904.jpeg",
				},
				CreatedDate: "20 de Junho",
			},
			{
				ID:               "sol_2",
				ServiceID:        "22222222-0000-0000-0000-000000000002",
				ServiceName:      "Pintura de Quarto",
				ProviderName:     "Marcos Souza",
				ProviderTitle:    "Pintor Residencial",
				ProviderRating:   "4.7",
				ProviderPhotoURL: "https://images.pexels.com/photos/614810/pexels-photo-614810.jpeg",
				Status:           "ACTIVE",
				Price:            850.0,
				ScheduledDate:    "26 de Junho às 08:00",
				Description:      "Pintura de duas paredes da sala com tinta acrílica fosca. Tinta e materiais já comprados.",
				Photos:           []string{},
				CreatedDate:      "19 de Junho",
			},
			{
				ID:                 "sol_3",
				ServiceID:          "22222222-0000-0000-0000-000000000003",
				ServiceName:        "Reparo de Disjuntor",
				ProviderName:       "Carlos Silva",
				ProviderTitle:      "Eletricista Residencial",
				ProviderRating:     "4.8",
				ProviderPhotoURL:   "https://images.pexels.com/photos/91227/pexels-photo-91227.jpeg",
				Status:             "CANCELLED",
				Price:              150.0,
				ScheduledDate:      "18 de Junho às 10:00",
				Description:        "Disjuntor geral caindo ao ligar o chuveiro elétrico. Necessário avaliar fiação e disjuntor.",
				CancellationReason: &reason,
				CancelledBy:        &cancelledByProvider,
				Photos:             []string{},
				CreatedDate:        "17 de Junho",
			},
			{
				ID:               "sol_4",
				ServiceID:        "22222222-0000-0000-0000-000000000008",
				ServiceName:      "Limpeza Residencial",
				ProviderName:     "Juliana Souza",
				ProviderTitle:    "Diarista Profissional",
				ProviderRating:   "4.9",
				ProviderPhotoURL: "https://images.pexels.com/photos/774909/pexels-photo-774909.jpeg",
				Status:           "COMPLETED",
				Price:            180.0,
				ScheduledDate:    "15 de Junho às 09:00",
				Description:      "Limpeza completa de apartamento de 2 quartos, sala, cozinha e 2 banheiros.",
				Photos:           []string{},
				CreatedDate:      "14 de Junho",
			},
		},
	}
}

func (ctrl *SolicitationController) GetSolicitations(c *gin.Context) {
	ctrl.mu.RLock()
	defer ctrl.mu.RUnlock()
	c.JSON(http.StatusOK, ctrl.solicitations)
}

func (ctrl *SolicitationController) GetSolicitationByID(c *gin.Context) {
	id := c.Param("id")
	ctrl.mu.RLock()
	defer ctrl.mu.RUnlock()

	for _, s := range ctrl.solicitations {
		if s.ID == id {
			c.JSON(http.StatusOK, s)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "solicitação não encontrada"})
}

func (ctrl *SolicitationController) CreateSolicitation(c *gin.Context) {
	var req struct {
		ServiceID     string   `json:"service_id"`
		ServiceName   string   `json:"service_name"`
		Description   string   `json:"description" binding:"required"`
		ScheduledDate string   `json:"scheduled_date"`
		Photos        []string `json:"photos"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := req.ServiceName
	serviceID := req.ServiceID
	if serviceID == "" {
		serviceID = "22222222-0000-0000-0000-000000000001"
	}

	providerName := "Carlos Mendes"
	providerTitle := "Especialista Residencial"
	providerRating := "4.9"
	providerPhotoURL := "https://images.pexels.com/photos/220453/pexels-photo-220453.jpeg"

	if req.ServiceName == "" && req.ServiceID != "" {
		if ctrl.serviceRepo != nil {
			service, err := ctrl.serviceRepo.GetServiceByID(c.Request.Context(), req.ServiceID)
			if err == nil && service != nil {
				name = service.Name
				if service.ProviderName != "" {
					providerName = service.ProviderName
				}
				if service.ProviderTitle != "" {
					providerTitle = service.ProviderTitle
				} else if service.Category != nil && service.Category.Name != "" {
					providerTitle = service.Category.Name
				}
				if service.ProviderPhotoURL != "" {
					providerPhotoURL = service.ProviderPhotoURL
				} else if service.ThumbnailURL != "" {
					providerPhotoURL = service.ThumbnailURL
				}
				if service.ReviewsAverage > 0 {
					providerRating = fmt.Sprintf("%.1f", service.ReviewsAverage)
				}
			}
		}
		if name == "" {
			name = "Solicitação de Serviço"
		}
	} else if name == "" {
		name = "Solicitação de Serviço"
	}

	photos := req.Photos
	if photos == nil {
		photos = []string{}
	}

	dateStr := time.Now().Format("02/01/2006")
	newSol := model.SolicitationItem{
		ID:                 fmt.Sprintf("sol_%d", time.Now().Unix()),
		ServiceID:          serviceID,
		ServiceName:        name,
		ProviderName:       providerName,
		ProviderTitle:      providerTitle,
		ProviderRating:     providerRating,
		ProviderPhotoURL:   providerPhotoURL,
		Status:             "PENDING_BUDGET",
		Price:              0.0,
		ScheduledDate:      req.ScheduledDate,
		Description:        req.Description,
		RenegotiationCount: 0,
		Photos:             photos,
		CreatedDate:        dateStr,
	}

	ctrl.mu.Lock()
	ctrl.solicitations = append([]model.SolicitationItem{newSol}, ctrl.solicitations...)
	ctrl.mu.Unlock()

	c.JSON(http.StatusCreated, newSol)
}

func (ctrl *SolicitationController) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status              string   `json:"status" binding:"required"`
		CancellationReason  *string  `json:"cancellation_reason"`
		CancelledBy         *string  `json:"cancelled_by"`
		CounterOfferPrice   *float64 `json:"counter_offer_price"`
		CounterOfferMessage *string  `json:"counter_offer_message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	for i, s := range ctrl.solicitations {
		if s.ID == id {
			ctrl.solicitations[i].Status = req.Status
			if req.CancellationReason != nil {
				ctrl.solicitations[i].CancellationReason = req.CancellationReason
			}
			if req.CancelledBy != nil {
				ctrl.solicitations[i].CancelledBy = req.CancelledBy
			}
			if req.CounterOfferPrice != nil {
				ctrl.solicitations[i].CounterOfferPrice = req.CounterOfferPrice
				ctrl.solicitations[i].Price = *req.CounterOfferPrice
				ctrl.solicitations[i].RenegotiationCount++
				if ctrl.solicitations[i].RenegotiationCount >= 5 {
					ctrl.solicitations[i].Status = "CANCELLED"
					cancelledBy := "MAX_RENEGOTIATIONS"
					cancellationReason := "Limite máximo de 5 contrapropostas atingido sem acordo."
					ctrl.solicitations[i].CancelledBy = &cancelledBy
					ctrl.solicitations[i].CancellationReason = &cancellationReason
				}
			}
			if req.CounterOfferMessage != nil {
				ctrl.solicitations[i].CounterOfferMessage = req.CounterOfferMessage
			}
			c.JSON(http.StatusOK, ctrl.solicitations[i])
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "solicitação não encontrada"})
}

func (ctrl *SolicitationController) GetActiveSolicitations() []model.SolicitationItem {
	ctrl.mu.RLock()
	defer ctrl.mu.RUnlock()

	var active []model.SolicitationItem
	for _, s := range ctrl.solicitations {
		if s.Status == "PENDING_BUDGET" || s.Status == "BUDGET_RECEIVED" || s.Status == "ACTIVE" {
			active = append(active, s)
			if len(active) >= 3 {
				break
			}
		}
	}
	return active
}
