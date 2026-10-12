package controller

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/http/middleware"
	"bikko-app/internal/infrastructure/persistence/postgres"
	"bikko-app/internal/infrastructure/push"
	"bikko-app/internal/model"

	"github.com/gin-gonic/gin"
)

type SolicitationController struct {
	mu            sync.RWMutex
	solicitations []model.SolicitationItem
	serviceRepo   domain.ServiceRepository
	pgRepo        *postgres.SolicitationPostgresRepository
	pushService   push.PushService
}

func NewSolicitationController(serviceRepo domain.ServiceRepository, pushService ...push.PushService) *SolicitationController {
	reason := "O provedor não tinha disponibilidade para a data/hora solicitada."
	cancelledByProvider := "PROVIDER"
	completedTechReport := "Higienização completa dos 2 quartos, sala, cozinha e banheiros, desengorduramento de bancadas e limpeza de vidros concluída sem pendências."

	var ps push.PushService
	if len(pushService) > 0 {
		ps = pushService[0]
	}

	return &SolicitationController{
		serviceRepo: serviceRepo,
		pushService: ps,
		solicitations: []model.SolicitationItem{
			{
				ID:               "sol_1",
				ClientID:         "11111111-0000-0000-0000-000000000099",
				ProviderID:       "11111111-0000-0000-0000-000000000009",
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
				ClientID:         "11111111-0000-0000-0000-000000000099",
				ProviderID:       "11111111-0000-0000-0000-000000000002",
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
				ClientID:           "11111111-0000-0000-0000-000000000099",
				ProviderID:         "11111111-0000-0000-0000-000000000003",
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
				ClientID:         "11111111-0000-0000-0000-000000000099",
				ProviderID:       "11111111-0000-0000-0000-000000000004",
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
				TechnicalReport:  &completedTechReport,
				Photos:           []string{},
				CreatedDate:      "14 de Junho",
			},
		},
	}
}

func (ctrl *SolicitationController) SetPushService(ps push.PushService) {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()
	ctrl.pushService = ps
}

func (ctrl *SolicitationController) SetPostgresRepository(repo *postgres.SolicitationPostgresRepository) {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()
	ctrl.pgRepo = repo
}

func (ctrl *SolicitationController) GetSolicitations(c *gin.Context) {
	ctrl.mu.RLock()
	pgRepo := ctrl.pgRepo
	ctrl.mu.RUnlock()

	if pgRepo != nil && pgRepo.HasPool() {
		items, err := pgRepo.GetSolicitationsByUser(c.Request.Context(), middleware.GetUserID(c))
		if err == nil && len(items) > 0 {
			c.JSON(http.StatusOK, items)
			return
		}
	}

	ctrl.mu.RLock()
	defer ctrl.mu.RUnlock()
	c.JSON(http.StatusOK, ctrl.solicitations)
}

func (ctrl *SolicitationController) GetSolicitationByID(c *gin.Context) {
	id := c.Param("id")

	ctrl.mu.RLock()
	pgRepo := ctrl.pgRepo
	ctrl.mu.RUnlock()

	if pgRepo != nil && pgRepo.HasPool() {
		item, err := pgRepo.GetSolicitationByID(c.Request.Context(), id)
		if err == nil && item != nil {
			c.JSON(http.StatusOK, item)
			return
		}
	}

	ctrl.mu.RLock()
	defer ctrl.mu.RUnlock()

	for _, s := range ctrl.solicitations {
		if matchSolicitationID(s.ID, id) {
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
	providerID := "11111111-0000-0000-0000-000000000009"

	if req.ServiceName == "" && req.ServiceID != "" {
		if ctrl.serviceRepo != nil {
			service, err := ctrl.serviceRepo.GetServiceByID(c.Request.Context(), req.ServiceID)
			if err == nil && service != nil {
				name = service.Name
				if service.BikkerID != "" {
					providerID = service.BikkerID
				}
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
	clientID := middleware.GetUserID(c)
	if clientID == "" {
		clientID = "11111111-0000-0000-0000-000000000099"
	}

	newSol := model.SolicitationItem{
		ID:                 fmt.Sprintf("sol_%d", time.Now().Unix()),
		ClientID:           clientID,
		ProviderID:         providerID,
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

	ctrl.mu.RLock()
	pgRepo := ctrl.pgRepo
	pushSvc := ctrl.pushService
	ctrl.mu.RUnlock()

	if pgRepo != nil && pgRepo.HasPool() {
		if persisted, err := pgRepo.CreateSolicitation(c.Request.Context(), newSol); err == nil && persisted != nil {
			newSol = *persisted
		}
	}

	ctrl.mu.Lock()
	ctrl.solicitations = append([]model.SolicitationItem{newSol}, ctrl.solicitations...)
	ctrl.mu.Unlock()

	if pushSvc != nil && newSol.ProviderID != "" {
		pushData := map[string]string{
			"solicitation_id": newSol.ID,
			"type":            "new_solicitation",
		}
		if err := pushSvc.SendNotification(
			c.Request.Context(),
			newSol.ProviderID,
			"Nova Solicitação de Orçamento",
			fmt.Sprintf("Você recebeu um novo pedido para %s.", newSol.ServiceName),
			pushData,
		); err != nil {
			log.Printf("[SolicitationController] Failed to send new solicitation push notification: %v\n", err)
		}
	}

	c.JSON(http.StatusCreated, newSol)
}

func (ctrl *SolicitationController) dispatchStatusPush(
	ctx context.Context,
	s model.SolicitationItem,
	status string,
	counterOfferPrice *float64,
	senderRole *string,
	senderID *string,
	currentUserID string,
) {
	if ctrl.pushService == nil {
		return
	}

	if senderID != nil && *senderID != "" {
		currentUserID = *senderID
	}

	var targetUserID string
	if senderRole != nil && strings.EqualFold(*senderRole, "PROVIDER") {
		targetUserID = s.ClientID
	} else if senderRole != nil && strings.EqualFold(*senderRole, "CLIENT") {
		targetUserID = s.ProviderID
	} else if currentUserID != "" && currentUserID == s.ProviderID {
		targetUserID = s.ClientID
	} else if currentUserID != "" && currentUserID == s.ClientID {
		targetUserID = s.ProviderID
	} else {
		if s.ProviderID != "" {
			targetUserID = s.ProviderID
		} else {
			targetUserID = s.ClientID
		}
	}

	if counterOfferPrice != nil && targetUserID != "" {
		pushData := map[string]string{
			"solicitation_id": s.ID,
			"type":            "counter_offer",
		}
		if err := ctrl.pushService.SendNotification(
			ctx,
			targetUserID,
			"Nova Contraproposta!",
			"Você recebeu uma nova oferta para a solicitação.",
			pushData,
		); err != nil {
			log.Printf("[SolicitationController] Failed to send counter offer push notification: %v\n", err)
		}
	} else if status == "ACCEPTED" && targetUserID != "" {
		pushData := map[string]string{
			"solicitation_id": s.ID,
			"type":            "accepted",
		}
		if err := ctrl.pushService.SendNotification(
			ctx,
			targetUserID,
			"Serviço Fechado!",
			"O serviço foi aceito e confirmado com sucesso.",
			pushData,
		); err != nil {
			log.Printf("[SolicitationController] Failed to send accepted push notification: %v\n", err)
		}
	} else if strings.EqualFold(status, "ACTIVE") && targetUserID != "" {
		pushData := map[string]string{
			"solicitation_id": s.ID,
			"type":            "accepted",
		}
		title := "Proposta Aceita! 🎉"
		body := "O cliente aceitou sua proposta! O serviço está confirmado no seu histórico de pedidos."
		if (senderRole != nil && strings.EqualFold(*senderRole, "PROVIDER")) || (targetUserID == s.ClientID && s.ClientID != s.ProviderID) {
			title = "Proposta Aceita pelo Profissional! 🎉"
			body = "O profissional aceitou sua contraproposta! O serviço está confirmado no seu histórico de pedidos."
		}
		if err := ctrl.pushService.SendNotification(
			ctx,
			targetUserID,
			title,
			body,
			pushData,
		); err != nil {
			log.Printf("[SolicitationController] Failed to send active push notification: %v\n", err)
		}
	}
}

func matchSolicitationID(itemID, targetID string) bool {
	if itemID == targetID {
		return true
	}
	alias := map[string]string{
		"sol_1": "33333333-0000-0000-0000-000000000001",
		"sol_2": "33333333-0000-0000-0000-000000000002",
		"sol_3": "33333333-0000-0000-0000-000000000003",
		"sol_4": "33333333-0000-0000-0000-000000000004",
	}
	if a, ok := alias[itemID]; ok && a == targetID {
		return true
	}
	if a, ok := alias[targetID]; ok && a == itemID {
		return true
	}
	return false
}

func (ctrl *SolicitationController) GetInMemorySolicitationByID(id string) *model.SolicitationItem {
	ctrl.mu.RLock()
	defer ctrl.mu.RUnlock()
	for _, s := range ctrl.solicitations {
		if matchSolicitationID(s.ID, id) {
			cp := s
			return &cp
		}
	}
	return nil
}

func (ctrl *SolicitationController) SyncSolicitationInMemory(item model.SolicitationItem) {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()
	for i, s := range ctrl.solicitations {
		if matchSolicitationID(s.ID, item.ID) {
			ctrl.solicitations[i] = item
			return
		}
	}
	ctrl.solicitations = append([]model.SolicitationItem{item}, ctrl.solicitations...)
}

func (ctrl *SolicitationController) ApplyProviderProposalInMemory(id, providerID string, price float64, note string, accepted bool) *model.SolicitationItem {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	for i, s := range ctrl.solicitations {
		if matchSolicitationID(s.ID, id) {
			if providerID != "" {
				ctrl.solicitations[i].ProviderID = providerID
			}
			if accepted {
				ctrl.solicitations[i].Status = "ACTIVE"
				if price > 0 {
					ctrl.solicitations[i].Price = price
				} else if s.CounterOfferPrice != nil && *s.CounterOfferPrice > 0 {
					ctrl.solicitations[i].Price = *s.CounterOfferPrice
				}
			} else {
				ctrl.solicitations[i].Status = "BUDGET_RECEIVED"
				if price > 0 {
					ctrl.solicitations[i].Price = price
				}
				if strings.TrimSpace(note) != "" {
					n := strings.TrimSpace(note)
					ctrl.solicitations[i].CounterOfferMessage = &n
				}
			}
			cp := ctrl.solicitations[i]
			return &cp
		}
	}

	status := "BUDGET_RECEIVED"
	if accepted {
		status = "ACTIVE"
	}
	newItem := model.SolicitationItem{
		ID:            id,
		ClientID:      "11111111-0000-0000-0000-000000000099",
		ProviderID:    providerID,
		ServiceName:   "Solicitação de Serviço",
		ProviderName:  "Profissional Bikko",
		Status:        status,
		Price:         price,
		Photos:        []string{},
		CreatedDate:   time.Now().Format("02/01/2006"),
		ScheduledDate: time.Now().Format("02/01/2006 às 14:00"),
	}
	if strings.TrimSpace(note) != "" {
		n := strings.TrimSpace(note)
		newItem.CounterOfferMessage = &n
	}
	ctrl.solicitations = append([]model.SolicitationItem{newItem}, ctrl.solicitations...)
	return &newItem
}

func (ctrl *SolicitationController) ApplyProviderCounterOfferInMemory(id, providerID string, counterPrice float64, reason string) *model.SolicitationItem {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	for i, s := range ctrl.solicitations {
		if matchSolicitationID(s.ID, id) {
			if providerID != "" {
				ctrl.solicitations[i].ProviderID = providerID
			}
			ctrl.solicitations[i].Status = "BUDGET_RECEIVED"
			ctrl.solicitations[i].Price = counterPrice
			pCopy := counterPrice
			ctrl.solicitations[i].CounterOfferPrice = &pCopy
			rCopy := strings.TrimSpace(reason)
			ctrl.solicitations[i].CounterOfferMessage = &rCopy
			ctrl.solicitations[i].RenegotiationCount++

			ctrl.solicitations[i].CounterOffers = append(ctrl.solicitations[i].CounterOffers, model.CounterOfferItem{
				ID:           fmt.Sprintf("co_%d", ctrl.solicitations[i].RenegotiationCount),
				Round:        ctrl.solicitations[i].RenegotiationCount,
				Author:       "PROVIDER",
				IsFromClient: false,
				Price:        counterPrice,
				Reason:       rCopy,
			})

			if ctrl.solicitations[i].RenegotiationCount >= 5 {
				ctrl.solicitations[i].Status = "CANCELLED"
				cancelledBy := "MAX_RENEGOTIATIONS"
				cancellationReason := "Limite máximo de 5 contrapropostas atingido sem acordo."
				ctrl.solicitations[i].CancelledBy = &cancelledBy
				ctrl.solicitations[i].CancellationReason = &cancellationReason
			}

			cp := ctrl.solicitations[i]
			return &cp
		}
	}

	pCopy := counterPrice
	rCopy := strings.TrimSpace(reason)
	newItem := model.SolicitationItem{
		ID:                  id,
		ClientID:            "11111111-0000-0000-0000-000000000099",
		ProviderID:          providerID,
		ServiceName:         "Solicitação de Serviço",
		ProviderName:        "Profissional Bikko",
		Status:              "BUDGET_RECEIVED",
		Price:               counterPrice,
		CounterOfferPrice:   &pCopy,
		CounterOfferMessage: &rCopy,
		RenegotiationCount:  1,
		CounterOffers: []model.CounterOfferItem{
			{
				ID:           "co_1",
				Round:        1,
				Author:       "PROVIDER",
				IsFromClient: false,
				Price:        counterPrice,
				Reason:       rCopy,
			},
		},
		Photos:        []string{},
		CreatedDate:   time.Now().Format("02/01/2006"),
		ScheduledDate: time.Now().Format("02/01/2006 às 14:00"),
	}
	ctrl.solicitations = append([]model.SolicitationItem{newItem}, ctrl.solicitations...)
	return &newItem
}

func (ctrl *SolicitationController) ApplyProviderRejectInMemory(id, providerID, reason string) *model.SolicitationItem {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	cancelledBy := "PROVIDER"
	rCopy := strings.TrimSpace(reason)

	for i, s := range ctrl.solicitations {
		if matchSolicitationID(s.ID, id) {
			if providerID != "" {
				ctrl.solicitations[i].ProviderID = providerID
			}
			ctrl.solicitations[i].Status = "CANCELLED"
			ctrl.solicitations[i].CancelledBy = &cancelledBy
			ctrl.solicitations[i].CancellationReason = &rCopy
			cp := ctrl.solicitations[i]
			return &cp
		}
	}

	newItem := model.SolicitationItem{
		ID:                 id,
		ClientID:           "11111111-0000-0000-0000-000000000099",
		ProviderID:         providerID,
		ServiceName:        "Solicitação de Serviço",
		ProviderName:       "Profissional Bikko",
		Status:             "CANCELLED",
		CancelledBy:        &cancelledBy,
		CancellationReason: &rCopy,
		Photos:             []string{},
		CreatedDate:        time.Now().Format("02/01/2006"),
		ScheduledDate:      time.Now().Format("02/01/2006 às 14:00"),
	}
	ctrl.solicitations = append([]model.SolicitationItem{newItem}, ctrl.solicitations...)
	return &newItem
}

func (ctrl *SolicitationController) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status              string   `json:"status" binding:"required"`
		TechnicalReport     *string  `json:"technical_report"`
		CancellationReason  *string  `json:"cancellation_reason"`
		CancelledBy         *string  `json:"cancelled_by"`
		CounterOfferPrice   *float64 `json:"counter_offer_price"`
		CounterOfferMessage *string  `json:"counter_offer_message"`
		SenderRole          *string  `json:"sender_role"`
		SenderID            *string  `json:"sender_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TechnicalReport != nil {
		runes := []rune(strings.TrimSpace(*req.TechnicalReport))
		if len(runes) > 280 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "relatório técnico deve ter no máximo 280 caracteres"})
			return
		}
		trimmed := string(runes)
		req.TechnicalReport = &trimmed
	}

	ctrl.mu.RLock()
	pgRepo := ctrl.pgRepo
	ctrl.mu.RUnlock()

	if pgRepo != nil && pgRepo.HasPool() {
		updated, err := pgRepo.UpdateSolicitationStatus(c.Request.Context(), id, postgres.SolicitationStatusUpdateInput{
			Status:              req.Status,
			TechnicalReport:     req.TechnicalReport,
			CancellationReason:  req.CancellationReason,
			CancelledBy:         req.CancelledBy,
			CounterOfferPrice:   req.CounterOfferPrice,
			CounterOfferMessage: req.CounterOfferMessage,
			SenderRole:          req.SenderRole,
			SenderID:            req.SenderID,
		})
		if err == nil && updated != nil {
			ctrl.SyncSolicitationInMemory(*updated)
			ctrl.dispatchStatusPush(
				c.Request.Context(),
				*updated,
				req.Status,
				req.CounterOfferPrice,
				req.SenderRole,
				req.SenderID,
				middleware.GetUserID(c),
			)
			c.JSON(http.StatusOK, updated)
			return
		}
	}

	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	for i, s := range ctrl.solicitations {
		if matchSolicitationID(s.ID, id) {
			ctrl.solicitations[i].Status = req.Status
			if req.TechnicalReport != nil && *req.TechnicalReport != "" {
				ctrl.solicitations[i].TechnicalReport = req.TechnicalReport
			}
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

				author := "CLIENT"
				if req.SenderRole != nil && strings.EqualFold(*req.SenderRole, "PROVIDER") {
					author = "PROVIDER"
				}
				reason := ""
				if req.CounterOfferMessage != nil {
					reason = *req.CounterOfferMessage
				}
				ctrl.solicitations[i].CounterOffers = append(ctrl.solicitations[i].CounterOffers, model.CounterOfferItem{
					ID:           fmt.Sprintf("co_%d", ctrl.solicitations[i].RenegotiationCount),
					Round:        ctrl.solicitations[i].RenegotiationCount,
					Author:       author,
					IsFromClient: author == "CLIENT",
					Price:        *req.CounterOfferPrice,
					Reason:       reason,
				})

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

			ctrl.dispatchStatusPush(
				c.Request.Context(),
				ctrl.solicitations[i],
				req.Status,
				req.CounterOfferPrice,
				req.SenderRole,
				req.SenderID,
				middleware.GetUserID(c),
			)

			c.JSON(http.StatusOK, ctrl.solicitations[i])
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "solicitação não encontrada"})
}

func (ctrl *SolicitationController) GetActiveSolicitations() []model.SolicitationItem {
	ctrl.mu.RLock()
	pgRepo := ctrl.pgRepo
	ctrl.mu.RUnlock()

	if pgRepo != nil && pgRepo.HasPool() {
		if items, err := pgRepo.GetSolicitationsByUser(context.Background(), ""); err == nil && len(items) > 0 {
			var active []model.SolicitationItem
			for _, s := range items {
				if s.Status == "PENDING_BUDGET" || s.Status == "BUDGET_RECEIVED" || s.Status == "NEGOTIATING" || s.Status == "ACTIVE" {
					active = append(active, s)
					if len(active) >= 3 {
						break
					}
				}
			}
			return active
		}
	}

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
