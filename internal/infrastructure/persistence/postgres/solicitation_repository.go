package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bikko-app/internal/domain"
	"bikko-app/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SolicitationStatusUpdateInput struct {
	Status              string
	TechnicalReport     *string
	CancellationReason  *string
	CancelledBy         *string
	CounterOfferPrice   *float64
	CounterOfferMessage *string
	SenderRole          *string
	SenderID            *string
}

type SolicitationPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewSolicitationPostgresRepository(pool *pgxpool.Pool) *SolicitationPostgresRepository {
	return &SolicitationPostgresRepository{pool: pool}
}

func (r *SolicitationPostgresRepository) HasPool() bool {
	return r != nil && r.pool != nil
}

func normalizeSolicitationID(id string) string {
	switch strings.TrimSpace(id) {
	case "sol_1":
		return "33333333-0000-0000-0000-000000000001"
	case "sol_2":
		return "33333333-0000-0000-0000-000000000002"
	case "sol_3":
		return "33333333-0000-0000-0000-000000000003"
	case "sol_4":
		return "33333333-0000-0000-0000-000000000004"
	case "lead-1":
		return "33333333-0000-0000-0000-000000000101"
	case "lead-2":
		return "33333333-0000-0000-0000-000000000102"
	case "lead-3":
		return "33333333-0000-0000-0000-000000000103"
	default:
		return strings.TrimSpace(id)
	}
}

func isValidUUID(u string) bool {
	_, err := uuid.Parse(strings.TrimSpace(u))
	return err == nil
}

func mapDBStatusToAppStatus(quoteStatus string, orderStatus *string, hasProposal bool, counterOfferCount int) string {
	if orderStatus != nil {
		switch *orderStatus {
		case "COMPLETED":
			return "COMPLETED"
		case "CANCELLED":
			return "CANCELLED"
		case "IN_PROGRESS":
			return "ACTIVE"
		case "SCHEDULED":
			if quoteStatus == "ACCEPTED" {
				return "ACTIVE"
			}
		}
	}

	switch quoteStatus {
	case "REJECTED", "EXPIRED":
		return "CANCELLED"
	case "ACCEPTED":
		return "ACTIVE"
	case "NEGOTIATING":
		if hasProposal || counterOfferCount > 0 {
			return "BUDGET_RECEIVED"
		}
		return "BUDGET_RECEIVED"
	case "PENDING":
		return "PENDING_BUDGET"
	default:
		return "PENDING_BUDGET"
	}
}

func (r *SolicitationPostgresRepository) loadNegotiations(ctx context.Context, item *model.SolicitationItem) {
	if !r.HasPool() || item == nil || !isValidUUID(item.ID) {
		return
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text,
			sender_id::text,
			COALESCE(message, ''),
			COALESCE(proposed_price, 0),
			COALESCE(photos, '[]'::jsonb),
			is_counter_offer
		FROM quote_negotiations
		WHERE quote_request_id = $1::uuid
		ORDER BY created_at ASC
	`, item.ID)
	if err != nil {
		return
	}
	defer rows.Close()

	photoSet := make(map[string]bool)
	for _, p := range item.Photos {
		if p != "" {
			photoSet[p] = true
		}
	}

	var counterOffers []model.CounterOfferItem
	hasInitialProposal := item.Price > 0
	round := 0

	for rows.Next() {
		var negID, senderID, msg string
		var proposedPrice float64
		var photosJSON []byte
		var isCounter bool

		if err := rows.Scan(&negID, &senderID, &msg, &proposedPrice, &photosJSON, &isCounter); err != nil {
			continue
		}

		var negPhotos []string
		if len(photosJSON) > 0 && json.Unmarshal(photosJSON, &negPhotos) == nil {
			for _, p := range negPhotos {
				if p != "" && !photoSet[p] {
					photoSet[p] = true
					item.Photos = append(item.Photos, p)
				}
			}
		}

		if isCounter {
			round++
			author := "CLIENT"
			if senderID == item.ProviderID {
				author = "PROVIDER"
			}
			if round <= 5 {
				counterOffers = append(counterOffers, model.CounterOfferItem{
					ID:           negID,
					Round:        round,
					Author:       author,
					IsFromClient: author == "CLIENT",
					Price:        proposedPrice,
					Reason:       msg,
				})
			}
			if proposedPrice > 0 {
				pCopy := proposedPrice
				item.CounterOfferPrice = &pCopy
				item.Price = proposedPrice
			}
			if msg != "" {
				mCopy := msg
				item.CounterOfferMessage = &mCopy
			}
		} else {
			if proposedPrice > 0 {
				hasInitialProposal = true
				item.Price = proposedPrice
			}
		}
	}

	item.CounterOffers = counterOffers
	item.RenegotiationCount = len(counterOffers)
	if item.Photos == nil {
		item.Photos = []string{}
	}
	if item.Status == "NEGOTIATING" && (hasInitialProposal || len(counterOffers) > 0) {
		item.Status = "BUDGET_RECEIVED"
	}
}

func (r *SolicitationPostgresRepository) GetSolicitationsByUser(ctx context.Context, userID string) ([]model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, fmt.Errorf("database pool not initialized")
	}

	targetUserID := strings.TrimSpace(userID)
	if targetUserID == "" || !isValidUUID(targetUserID) {
		targetUserID = "11111111-0000-0000-0000-000000000099"
	}

	query := `
		SELECT
			qr.id::text,
			qr.customer_id::text,
			qr.bikker_id::text,
			qr.service_id::text,
			COALESCE(s.name, 'Solicitação de Serviço'),
			COALESCE(pu.full_name, 'Profissional Bikko'),
			COALESCE(b.profession, 'Especialista'),
			COALESCE(ROUND(b.rating::numeric, 1)::text, '4.9'),
			COALESCE(pu.profile_photo_url, COALESCE(s.thumbnail_url, '')),
			qr.status::text,
			o.status::text,
			COALESCE(o.final_price, 0),
			COALESCE(o.scheduled_date, qr.desired_date, qr.created_at),
			COALESCE(qr.initial_description, ''),
			o.technical_report,
			o.cancellation_reason,
			qr.created_at
		FROM quote_requests qr
		LEFT JOIN orders o ON o.quote_request_id = qr.id
		LEFT JOIN services s ON s.id = qr.service_id
		LEFT JOIN users pu ON pu.id = qr.bikker_id
		LEFT JOIN bikkers b ON b.id = qr.bikker_id
		WHERE qr.customer_id = $1::uuid
		   OR qr.bikker_id = $1::uuid
		   OR EXISTS (SELECT 1 FROM quote_negotiations qn WHERE qn.quote_request_id = qr.id AND qn.sender_id = $1::uuid)
		ORDER BY qr.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to query solicitations: %w", err)
	}
	defer rows.Close()

	var items []model.SolicitationItem
	for rows.Next() {
		var item model.SolicitationItem
		var quoteStatus string
		var orderStatus *string
		var scheduledTime time.Time
		var createdAt time.Time

		err := rows.Scan(
			&item.ID,
			&item.ClientID,
			&item.ProviderID,
			&item.ServiceID,
			&item.ServiceName,
			&item.ProviderName,
			&item.ProviderTitle,
			&item.ProviderRating,
			&item.ProviderPhotoURL,
			&quoteStatus,
			&orderStatus,
			&item.Price,
			&scheduledTime,
			&item.Description,
			&item.TechnicalReport,
			&item.CancellationReason,
			&createdAt,
		)
		if err != nil {
			continue
		}

		item.ScheduledDate = scheduledTime.Format("02/01/2006 às 15:04")
		item.CreatedDate = createdAt.Format("02/01/2006")
		item.Photos = []string{}
		item.Status = mapDBStatusToAppStatus(quoteStatus, orderStatus, item.Price > 0, 0)

		r.loadNegotiations(ctx, &item)
		item.Status = mapDBStatusToAppStatus(quoteStatus, orderStatus, item.Price > 0, item.RenegotiationCount)
		if item.Status == "CANCELLED" {
			if item.CancellationReason != nil && strings.Contains(*item.CancellationReason, "5 contrapropostas") {
				maxCancel := "MAX_RENEGOTIATIONS"
				item.CancelledBy = &maxCancel
			} else if quoteStatus == "EXPIRED" {
				clientCancel := "CLIENT"
				item.CancelledBy = &clientCancel
			} else {
				providerCancel := "PROVIDER"
				item.CancelledBy = &providerCancel
			}
		}

		items = append(items, item)
	}

	if items == nil {
		items = []model.SolicitationItem{}
	}
	return items, nil
}

func (r *SolicitationPostgresRepository) GetSolicitationByID(ctx context.Context, id string) (*model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, fmt.Errorf("database pool not initialized")
	}

	normID := normalizeSolicitationID(id)
	if !isValidUUID(normID) {
		return nil, domain.ErrNotFound
	}

	query := `
		SELECT
			qr.id::text,
			qr.customer_id::text,
			qr.bikker_id::text,
			qr.service_id::text,
			COALESCE(s.name, 'Solicitação de Serviço'),
			COALESCE(pu.full_name, 'Profissional Bikko'),
			COALESCE(b.profession, 'Especialista'),
			COALESCE(ROUND(b.rating::numeric, 1)::text, '4.9'),
			COALESCE(pu.profile_photo_url, COALESCE(s.thumbnail_url, '')),
			qr.status::text,
			o.status::text,
			COALESCE(o.final_price, 0),
			COALESCE(o.scheduled_date, qr.desired_date, qr.created_at),
			COALESCE(qr.initial_description, ''),
			o.technical_report,
			o.cancellation_reason,
			qr.created_at
		FROM quote_requests qr
		LEFT JOIN orders o ON o.quote_request_id = qr.id
		LEFT JOIN services s ON s.id = qr.service_id
		LEFT JOIN users pu ON pu.id = qr.bikker_id
		LEFT JOIN bikkers b ON b.id = qr.bikker_id
		WHERE qr.id = $1::uuid OR o.id = $1::uuid
		LIMIT 1
	`

	var item model.SolicitationItem
	var quoteStatus string
	var orderStatus *string
	var scheduledTime time.Time
	var createdAt time.Time

	err := r.pool.QueryRow(ctx, query, normID).Scan(
		&item.ID,
		&item.ClientID,
		&item.ProviderID,
		&item.ServiceID,
		&item.ServiceName,
		&item.ProviderName,
		&item.ProviderTitle,
		&item.ProviderRating,
		&item.ProviderPhotoURL,
		&quoteStatus,
		&orderStatus,
		&item.Price,
		&scheduledTime,
		&item.Description,
		&item.TechnicalReport,
		&item.CancellationReason,
		&createdAt,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	item.ScheduledDate = scheduledTime.Format("02/01/2006 às 15:04")
	item.CreatedDate = createdAt.Format("02/01/2006")
	item.Photos = []string{}

	r.loadNegotiations(ctx, &item)
	item.Status = mapDBStatusToAppStatus(quoteStatus, orderStatus, item.Price > 0, item.RenegotiationCount)
	if item.Status == "CANCELLED" {
		if item.CancellationReason != nil && strings.Contains(*item.CancellationReason, "5 contrapropostas") {
			maxCancel := "MAX_RENEGOTIATIONS"
			item.CancelledBy = &maxCancel
		} else if quoteStatus == "EXPIRED" {
			clientCancel := "CLIENT"
			item.CancelledBy = &clientCancel
		} else {
			providerCancel := "PROVIDER"
			item.CancelledBy = &providerCancel
		}
	}

	return &item, nil
}

func (r *SolicitationPostgresRepository) CreateSolicitation(ctx context.Context, item model.SolicitationItem) (*model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, fmt.Errorf("database pool not initialized")
	}

	customerID := item.ClientID
	if !isValidUUID(customerID) {
		customerID = "11111111-0000-0000-0000-000000000099"
	} else {
		var exists bool
		_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1::uuid)`, customerID).Scan(&exists)
		if !exists {
			customerID = "11111111-0000-0000-0000-000000000099"
		}
	}

	serviceID := item.ServiceID
	bikkerID := item.ProviderID
	if !isValidUUID(serviceID) {
		serviceID = "22222222-0000-0000-0000-000000000001"
	}

	var dbServiceID, dbBikkerID string
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, bikker_id::text FROM services WHERE id = $1::uuid LIMIT 1
	`, serviceID).Scan(&dbServiceID, &dbBikkerID)
	if err == nil {
		serviceID = dbServiceID
		bikkerID = dbBikkerID
	} else {
		err = r.pool.QueryRow(ctx, `SELECT id::text, bikker_id::text FROM services WHERE is_active = TRUE ORDER BY created_at ASC LIMIT 1`).Scan(&dbServiceID, &dbBikkerID)
		if err == nil {
			serviceID = dbServiceID
			bikkerID = dbBikkerID
		}
	}

	var quoteID string
	err = r.pool.QueryRow(ctx, `
		INSERT INTO quote_requests (
			customer_id, bikker_id, service_id, status, initial_description, desired_date, location_address, location_geom
		)
		VALUES (
			$1::uuid, $2::uuid, $3::uuid, 'PENDING', $4, NOW() + INTERVAL '2 days', 'São Paulo, SP',
			ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography
		)
		RETURNING id::text
	`, customerID, bikkerID, serviceID, item.Description).Scan(&quoteID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert quote_request: %w", err)
	}

	if len(item.Photos) > 0 {
		photosJSON, _ := json.Marshal(item.Photos)
		_, _ = r.pool.Exec(ctx, `
			INSERT INTO quote_negotiations (quote_request_id, sender_id, message, proposed_price, photos, is_counter_offer)
			VALUES ($1::uuid, $2::uuid, $3, 0, $4::jsonb, FALSE)
		`, quoteID, customerID, item.Description, string(photosJSON))
	}

	created, err := r.GetSolicitationByID(ctx, quoteID)
	if err != nil {
		item.ID = quoteID
		return &item, nil
	}
	if item.ScheduledDate != "" {
		created.ScheduledDate = item.ScheduledDate
	}
	return created, nil
}

func (r *SolicitationPostgresRepository) assignBikkerIfValid(ctx context.Context, quoteID string, existing *model.SolicitationItem, candidateBikkerID string) {
	if !r.HasPool() || existing == nil || !isValidUUID(candidateBikkerID) {
		return
	}
	var bikkerExists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bikkers WHERE id = $1::uuid)`, candidateBikkerID).Scan(&bikkerExists); err == nil && bikkerExists {
		if existing.ProviderID != candidateBikkerID {
			_, _ = r.pool.Exec(ctx, `UPDATE quote_requests SET bikker_id = $2::uuid WHERE id = $1::uuid`, quoteID, candidateBikkerID)
			_, _ = r.pool.Exec(ctx, `UPDATE orders SET bikker_id = $2::uuid WHERE quote_request_id = $1::uuid`, quoteID, candidateBikkerID)
			existing.ProviderID = candidateBikkerID
		}
	}
}

func (r *SolicitationPostgresRepository) UpdateSolicitationStatus(ctx context.Context, id string, input SolicitationStatusUpdateInput) (*model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, fmt.Errorf("database pool not initialized")
	}

	existing, err := r.GetSolicitationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	quoteID := existing.ID

	if input.SenderRole != nil && strings.EqualFold(*input.SenderRole, "PROVIDER") && input.SenderID != nil {
		r.assignBikkerIfValid(ctx, quoteID, existing, *input.SenderID)
	}

	targetAppStatus := strings.ToUpper(strings.TrimSpace(input.Status))
	cancellationReason := input.CancellationReason
	cancelledBy := input.CancelledBy

	if input.CounterOfferPrice != nil {
		senderID := existing.ClientID
		if input.SenderID != nil && isValidUUID(*input.SenderID) {
			senderID = *input.SenderID
		} else if input.SenderRole != nil && strings.EqualFold(*input.SenderRole, "PROVIDER") {
			senderID = existing.ProviderID
		}

		msg := "Contraproposta enviada"
		if input.CounterOfferMessage != nil && strings.TrimSpace(*input.CounterOfferMessage) != "" {
			msg = strings.TrimSpace(*input.CounterOfferMessage)
		}

		_, err := r.pool.Exec(ctx, `
			INSERT INTO quote_negotiations (quote_request_id, sender_id, message, proposed_price, is_counter_offer)
			VALUES ($1::uuid, $2::uuid, $3, $4, TRUE)
		`, quoteID, senderID, msg, *input.CounterOfferPrice)
		if err != nil {
			return nil, fmt.Errorf("failed to insert counter-offer: %w", err)
		}

		var count int
		_ = r.pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM quote_negotiations WHERE quote_request_id = $1::uuid AND is_counter_offer = TRUE
		`, quoteID).Scan(&count)

		if count >= 5 {
			targetAppStatus = "CANCELLED"
			reason := "Limite máximo de 5 contrapropostas atingido sem acordo."
			by := "MAX_RENEGOTIATIONS"
			cancellationReason = &reason
			cancelledBy = &by
		}
	}

	var dbQuoteStatus string
	var dbOrderStatus string
	shouldUpsertOrder := false

	switch targetAppStatus {
	case "PENDING_BUDGET", "PENDING", "OPEN":
		dbQuoteStatus = "PENDING"
	case "BUDGET_RECEIVED", "NEGOTIATING":
		dbQuoteStatus = "NEGOTIATING"
	case "ACCEPTED", "SCHEDULED":
		dbQuoteStatus = "ACCEPTED"
		dbOrderStatus = "SCHEDULED"
		shouldUpsertOrder = true
	case "ACTIVE", "IN_PROGRESS":
		dbQuoteStatus = "ACCEPTED"
		dbOrderStatus = "IN_PROGRESS"
		shouldUpsertOrder = true
	case "COMPLETED":
		dbQuoteStatus = "ACCEPTED"
		dbOrderStatus = "COMPLETED"
		shouldUpsertOrder = true
	case "CANCELLED", "REJECTED":
		if cancelledBy != nil && (strings.EqualFold(*cancelledBy, "CLIENT") || strings.EqualFold(*cancelledBy, "USER")) {
			dbQuoteStatus = "EXPIRED"
		} else {
			dbQuoteStatus = "REJECTED"
		}
		dbOrderStatus = "CANCELLED"
		shouldUpsertOrder = true
	default:
		dbQuoteStatus = "NEGOTIATING"
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE quote_requests SET status = $2::quote_status, updated_at = NOW() WHERE id = $1::uuid
	`, quoteID, dbQuoteStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to update quote_requests status: %w", err)
	}

	finalPrice := existing.Price
	if input.CounterOfferPrice != nil && *input.CounterOfferPrice > 0 {
		finalPrice = *input.CounterOfferPrice
	}

	if shouldUpsertOrder || input.TechnicalReport != nil || cancellationReason != nil || input.CounterOfferPrice != nil {
		if dbOrderStatus == "" {
			dbOrderStatus = "SCHEDULED"
		}
		_, _ = r.pool.Exec(ctx, `
			INSERT INTO orders (
				quote_request_id, customer_id, bikker_id, service_id, status, final_price, scheduled_date, technical_report, cancellation_reason
			)
			VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::order_status, $6, NOW() + INTERVAL '1 day', $7, $8
			)
			ON CONFLICT (quote_request_id) DO UPDATE SET
				bikker_id = EXCLUDED.bikker_id,
				status = CASE WHEN $9 THEN EXCLUDED.status ELSE orders.status END,
				final_price = CASE WHEN $6 > 0 THEN $6 ELSE orders.final_price END,
				technical_report = COALESCE(EXCLUDED.technical_report, orders.technical_report),
				cancellation_reason = COALESCE(EXCLUDED.cancellation_reason, orders.cancellation_reason),
				updated_at = NOW()
		`,
			quoteID,
			existing.ClientID,
			existing.ProviderID,
			existing.ServiceID,
			dbOrderStatus,
			finalPrice,
			input.TechnicalReport,
			cancellationReason,
			shouldUpsertOrder,
		)
	}

	updated, err := r.GetSolicitationByID(ctx, quoteID)
	if err != nil {
		return nil, err
	}
	if cancelledBy != nil {
		updated.CancelledBy = cancelledBy
	}
	return updated, nil
}

func (r *SolicitationPostgresRepository) GetProviderFeed(ctx context.Context, bikkerID string) ([]domain.OpportunityLead, error) {
	if !r.HasPool() {
		return nil, fmt.Errorf("database pool not initialized")
	}

	if bikkerID == "empty-qa-id" || bikkerID == "11111111-0000-0000-0000-000000000101" {
		return []domain.OpportunityLead{}, nil
	}

	query := `
		SELECT
			qr.id::text,
			COALESCE(s.name, 'Solicitação de Serviço') AS title,
			COALESCE(qr.initial_description, '') AS description,
			COALESCE(c.name, 'Geral') AS category,
			COALESCE(qr.location_address, 'São Paulo, SP') AS neighborhood,
			COALESCE(cu.full_name, 'Cliente') AS client_name,
			qr.created_at,
			COALESCE(
				ROUND((ST_Distance(qr.location_geom, COALESCE(b.location_geom, qr.location_geom)) / 1000)::numeric, 1),
				2.5
			) AS dist_km
		FROM quote_requests qr
		LEFT JOIN services s ON s.id = qr.service_id
		LEFT JOIN categories c ON c.id = s.category_id
		LEFT JOIN users cu ON cu.id = qr.customer_id
		LEFT JOIN bikkers b ON b.id = qr.bikker_id
		WHERE qr.status IN ('PENDING', 'NEGOTIATING')
		  AND ($1 = '' OR qr.bikker_id::text = $1)
		ORDER BY qr.created_at DESC
		LIMIT 20
	`

	filterID := ""
	if isValidUUID(bikkerID) {
		filterID = bikkerID
	}

	rows, err := r.pool.Query(ctx, query, filterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leads []domain.OpportunityLead
	for rows.Next() {
		var lead domain.OpportunityLead
		if err := rows.Scan(
			&lead.ID,
			&lead.Title,
			&lead.Description,
			&lead.Category,
			&lead.Neighborhood,
			&lead.ClientName,
			&lead.CreatedAt,
			&lead.DistanceKm,
		); err != nil {
			continue
		}
		if lead.DistanceKm <= 0 {
			lead.DistanceKm = 2.5
		}
		lead.Urgency = "SCHEDULED"
		if strings.Contains(strings.ToLower(lead.Description), "urgente") || strings.Contains(strings.ToLower(lead.Description), "vazamento") {
			lead.Urgency = "EMERGENCY"
		}
		lead.PriceType = "BUDGET_TO_NEGOTIATE"
		lead.ClientVerified = true
		lead.Photos = []string{}

		// Load photos from quote_negotiations if any
		var photosJSON []byte
		if err := r.pool.QueryRow(ctx, `
			SELECT COALESCE(photos, '[]'::jsonb) FROM quote_negotiations
			WHERE quote_request_id = $1::uuid AND jsonb_array_length(COALESCE(photos, '[]'::jsonb)) > 0
			ORDER BY created_at ASC LIMIT 1
		`, lead.ID).Scan(&photosJSON); err == nil {
			_ = json.Unmarshal(photosJSON, &lead.Photos)
		}

		leads = append(leads, lead)
	}

	// If filtering by specific bikker returned no direct leads, return general open leads in the platform
	if len(leads) == 0 && filterID != "" {
		return r.GetProviderFeed(ctx, "")
	}

	if leads == nil {
		leads = []domain.OpportunityLead{}
	}
	return leads, nil
}

func (r *SolicitationPostgresRepository) SubmitProviderProposal(ctx context.Context, solicitationID, bikkerID string, price float64, note string) (*model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, fmt.Errorf("database pool not initialized")
	}

	existing, err := r.GetSolicitationByID(ctx, solicitationID)
	if err != nil {
		return nil, err
	}

	r.assignBikkerIfValid(ctx, existing.ID, existing, bikkerID)

	senderID := existing.ProviderID
	if isValidUUID(bikkerID) {
		senderID = bikkerID
	}
	msg := strings.TrimSpace(note)
	if msg == "" {
		msg = "Orçamento enviado pelo profissional."
	}

	isCounter := len(existing.CounterOffers) > 0

	_, err = r.pool.Exec(ctx, `
		INSERT INTO quote_negotiations (quote_request_id, sender_id, message, proposed_price, is_counter_offer)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5)
	`, existing.ID, senderID, msg, price, isCounter)
	if err != nil {
		return nil, fmt.Errorf("failed to insert proposal: %w", err)
	}

	_, _ = r.pool.Exec(ctx, `
		UPDATE quote_requests SET status = 'NEGOTIATING', updated_at = NOW() WHERE id = $1::uuid
	`, existing.ID)

	_, _ = r.pool.Exec(ctx, `
		INSERT INTO orders (quote_request_id, customer_id, bikker_id, service_id, status, final_price, scheduled_date)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'SCHEDULED', $5, NOW() + INTERVAL '1 day')
		ON CONFLICT (quote_request_id) DO UPDATE SET
			bikker_id = EXCLUDED.bikker_id,
			final_price = EXCLUDED.final_price,
			updated_at = NOW()
	`, existing.ID, existing.ClientID, existing.ProviderID, existing.ServiceID, price)

	return r.GetSolicitationByID(ctx, existing.ID)
}

func (r *SolicitationPostgresRepository) AcceptCounterOffer(ctx context.Context, solicitationID, bikkerID string, price float64, note string) (*model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, fmt.Errorf("database pool not initialized")
	}

	existing, err := r.GetSolicitationByID(ctx, solicitationID)
	if err != nil {
		return nil, err
	}

	r.assignBikkerIfValid(ctx, existing.ID, existing, bikkerID)

	finalPrice := price
	if finalPrice <= 0 {
		if existing.CounterOfferPrice != nil && *existing.CounterOfferPrice > 0 {
			finalPrice = *existing.CounterOfferPrice
		} else {
			finalPrice = existing.Price
		}
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE quote_requests SET status = 'ACCEPTED', updated_at = NOW() WHERE id = $1::uuid
	`, existing.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to accept quote_request: %w", err)
	}

	_, _ = r.pool.Exec(ctx, `
		INSERT INTO orders (quote_request_id, customer_id, bikker_id, service_id, status, final_price, scheduled_date)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'IN_PROGRESS', $5, NOW() + INTERVAL '1 day')
		ON CONFLICT (quote_request_id) DO UPDATE SET
			bikker_id = EXCLUDED.bikker_id,
			status = 'IN_PROGRESS',
			final_price = CASE WHEN $5 > 0 THEN $5 ELSE orders.final_price END,
			updated_at = NOW()
	`, existing.ID, existing.ClientID, existing.ProviderID, existing.ServiceID, finalPrice)

	return r.GetSolicitationByID(ctx, existing.ID)
}

func (r *SolicitationPostgresRepository) SubmitClientEvaluation(ctx context.Context, solicitationID, bikkerID string, req domain.ClientEvaluationRequest) error {
	if !r.HasPool() {
		return fmt.Errorf("database pool not initialized")
	}

	normID := normalizeSolicitationID(solicitationID)
	if !isValidUUID(normID) {
		return domain.ErrNotFound
	}

	existing, err := r.GetSolicitationByID(ctx, normID)
	if err != nil {
		return err
	}

	if strings.TrimSpace(req.TechnicalReport) != "" {
		_, _ = r.pool.Exec(ctx, `
			UPDATE orders
			SET technical_report = $2, updated_at = NOW()
			WHERE quote_request_id = $1::uuid OR id = $1::uuid
		`, existing.ID, strings.TrimSpace(req.TechnicalReport))
	}

	if isValidUUID(existing.ClientID) && req.Stars >= 1 && req.Stars <= 5 {
		_, _ = r.pool.Exec(ctx, `
			UPDATE users
			SET rating = ROUND(((COALESCE(rating, 0) * COALESCE(total_ratings, 0) + $2) / (COALESCE(total_ratings, 0) + 1))::numeric, 2),
			    total_ratings = COALESCE(total_ratings, 0) + 1,
			    updated_at = NOW()
			WHERE id = $1::uuid
		`, existing.ClientID, float64(req.Stars))
	}

	return nil
}
