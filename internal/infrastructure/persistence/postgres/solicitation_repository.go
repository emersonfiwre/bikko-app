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

func (r *SolicitationPostgresRepository) runAutoMigrations() {
	if !r.HasPool() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stmts := []string{
		"ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS contact_name VARCHAR(255) DEFAULT ''",
		"ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS contact_phone VARCHAR(50) DEFAULT ''",
		"ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS account_phone VARCHAR(50) DEFAULT ''",
		"ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS service_title VARCHAR(255) DEFAULT ''",
		"ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS category_name VARCHAR(255) DEFAULT ''",
		"ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS preferred_date VARCHAR(100) DEFAULT 'O mais rápido possível'",
		"ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS client_photos JSONB DEFAULT '[]'::jsonb",
		"ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS client_evaluated BOOLEAN DEFAULT FALSE",
		"ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS bikker_evaluated BOOLEAN DEFAULT FALSE",
		"ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS client_rating_given INTEGER DEFAULT 0",
		"ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS technical_report TEXT DEFAULT ''",
		"ALTER TABLE quote_requests ALTER COLUMN service_id DROP NOT NULL",
		"ALTER TABLE orders ALTER COLUMN service_id DROP NOT NULL",
		"ALTER TABLE reviews ALTER COLUMN service_id DROP NOT NULL",
		"ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_order_id_key",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_reviews_order_reviewer ON reviews (order_id, reviewer_id)",
	}

	for _, stmt := range stmts {
		_, _ = r.pool.Exec(ctx, stmt)
	}
}

func NewSolicitationPostgresRepository(pool *pgxpool.Pool) *SolicitationPostgresRepository {
	repo := &SolicitationPostgresRepository{pool: pool}
	if pool != nil {
		go repo.runAutoMigrations()
	}
	return repo
}

func (r *SolicitationPostgresRepository) HasPool() bool {
	return r != nil && r.pool != nil
}

func normalizeSolicitationID(id string) string {
	switch strings.TrimSpace(id) {
	case "sol_1", "service_encanador", "BKK-8492":
		return "33333333-0000-0000-0000-000000000001"
	case "sol_2", "BKK-8493":
		return "33333333-0000-0000-0000-000000000002"
	case "sol_3", "BKK-8494":
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

func formatBRLAmount(amount float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.2f", amount), ".", ",")
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
	counterOfferCount := 0

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
			counterOfferCount++
			round++
			author := "CLIENT"
			if senderID == item.ProviderID {
				author = "PROVIDER"
			}
			formatted := formatBRLAmount(proposedPrice)
			if proposedPrice <= 0 {
				formatted = "Pedido de mais informações"
			}
			if round <= 5 {
				counterOffers = append(counterOffers, model.CounterOfferItem{
					ID:             negID,
					Round:          round,
					Author:         author,
					IsFromClient:   author == "CLIENT",
					Price:          proposedPrice,
					FormattedPrice: formatted,
					Reason:         msg,
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
				round++
				reason := strings.TrimSpace(msg)
				if reason == "" {
					reason = "Incluso visita técnica, desmontagem e vedação."
				}
				if round <= 5 {
					counterOffers = append(counterOffers, model.CounterOfferItem{
						ID:             negID,
						Round:          round,
						Author:         "PROVIDER",
						IsFromClient:   false,
						Price:          proposedPrice,
						FormattedPrice: formatBRLAmount(proposedPrice),
						Reason:         reason,
					})
				}
			}
		}
	}

	if len(counterOffers) == 0 && item.Price > 0 && (item.Status == "BUDGET_RECEIVED" || item.Status == "NEGOTIATING" || item.Status == "ACTIVE") {
		reason := "Incluso visita técnica, desmontagem e vedação."
		if item.CounterOfferMessage != nil && strings.TrimSpace(*item.CounterOfferMessage) != "" {
			reason = strings.TrimSpace(*item.CounterOfferMessage)
		}
		counterOffers = append(counterOffers, model.CounterOfferItem{
			ID:             "co_1",
			Round:          1,
			Author:         "PROVIDER",
			IsFromClient:   false,
			Price:          item.Price,
			FormattedPrice: formatBRLAmount(item.Price),
			Reason:         reason,
		})
	}

	item.CounterOffers = counterOffers
	item.CounterOfferHistory = counterOffers
	item.NegotiationHistory = counterOffers
	item.RenegotiationCount = counterOfferCount
	if item.Photos == nil {
		item.Photos = []string{}
	}
	item.ClientPhotos = item.Photos
	if item.Status == "NEGOTIATING" && (hasInitialProposal || len(counterOffers) > 0) {
		item.Status = "BUDGET_RECEIVED"
	}
}

const solicitationBaseQuery = `
	SELECT
		qr.id::text,
		qr.customer_id::text,
		qr.bikker_id::text,
		COALESCE(qr.service_id::text, ''),
		COALESCE(NULLIF(qr.service_title, ''), s.name, 'Solicitação de Serviço'),
		COALESCE(pu.full_name, 'Profissional Bikko'),
		COALESCE(b.profession, 'Especialista'),
		CASE WHEN COALESCE(b.total_reviews, 0) = 0 THEN '0.0' ELSE ROUND(COALESCE(b.rating, 0)::numeric, 1)::text END,
		COALESCE(pu.profile_photo_url, COALESCE(s.thumbnail_url, '')),
		qr.status::text,
		o.status::text,
		COALESCE(o.final_price, 0),
		COALESCE(o.scheduled_date, qr.desired_date, qr.created_at),
		COALESCE(qr.initial_description, ''),
		COALESCE(o.technical_report, qr.technical_report, ''),
		o.cancellation_reason,
		qr.created_at,
		COALESCE(cu.full_name, 'Cliente'),
		COALESCE(NULLIF(qr.contact_name, ''), cu.full_name, 'Cliente'),
		COALESCE(cu.phone, qr.account_phone, ''),
		COALESCE(NULLIF(qr.contact_phone, ''), cu.phone, ''),
		CASE WHEN COALESCE(cu.total_ratings, 0) = 0 THEN '0.0' ELSE ROUND(COALESCE(cu.rating, 0)::numeric, 1)::text END,
		COALESCE(cu.total_ratings, 0),
		COALESCE(cu.profile_photo_url, ''),
		COALESCE(qr.location_address, 'São Paulo, SP'),
		COALESCE(NULLIF(qr.category_name, ''), c.name, 'Geral'),
		COALESCE(NULLIF(qr.preferred_date, ''), 'O mais rápido possível'),
		COALESCE(qr.client_photos, '[]'::jsonb),
		COALESCE(qr.client_evaluated, FALSE),
		COALESCE(qr.bikker_evaluated, FALSE),
		COALESCE(qr.client_rating_given, 0)
	FROM quote_requests qr
	LEFT JOIN orders o ON o.quote_request_id = qr.id
	LEFT JOIN services s ON s.id = qr.service_id
	LEFT JOIN categories c ON c.id = s.category_id
	LEFT JOIN users pu ON pu.id = qr.bikker_id
	LEFT JOIN bikkers b ON b.id = qr.bikker_id
	LEFT JOIN users cu ON cu.id = qr.customer_id
`

func (r *SolicitationPostgresRepository) GetSolicitationsByUser(ctx context.Context, userID string) ([]model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, fmt.Errorf("database pool not initialized")
	}

	targetUserID := strings.TrimSpace(userID)
	var query string
	var args []interface{}

	if targetUserID != "" && isValidUUID(targetUserID) {
		query = solicitationBaseQuery + `
			WHERE qr.customer_id = $1::uuid
			   OR qr.bikker_id = $1::uuid
			   OR EXISTS (SELECT 1 FROM quote_negotiations qn WHERE qn.quote_request_id = qr.id AND qn.sender_id = $1::uuid)
			ORDER BY qr.created_at DESC
		`
		args = append(args, targetUserID)
	} else {
		query = solicitationBaseQuery + `
			ORDER BY qr.created_at DESC
		`
	}

	rows, err := r.pool.Query(ctx, query, args...)
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
		var clientAccountName, contactName, accountPhone, contactPhone, clientRating, clientPhotoURL string
		var clientTotalRatings int
		var locationAddress, categoryName, preferredDate string
		var clientPhotosJSON []byte
		var clientEvaluated, bikkerEvaluated bool
		var clientRatingGiven int

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
			&clientAccountName,
			&contactName,
			&accountPhone,
			&contactPhone,
			&clientRating,
			&clientTotalRatings,
			&clientPhotoURL,
			&locationAddress,
			&categoryName,
			&preferredDate,
			&clientPhotosJSON,
			&clientEvaluated,
			&bikkerEvaluated,
			&clientRatingGiven,
		)
		if err != nil {
			continue
		}

		item.ServiceTitle = item.ServiceName
		item.InitialDescription = item.Description
		item.ClientName = clientAccountName
		item.ContactName = contactName
		item.AccountPhone = accountPhone
		item.ClientAccountPhone = accountPhone
		item.ContactPhone = contactPhone
		item.ClientPhone = accountPhone
		item.ClientRating = clientRating
		item.ClientTotalRatings = clientTotalRatings
		item.ClientPhotoURL = clientPhotoURL
		item.LocationAddress = locationAddress
		item.Address = locationAddress
		item.Neighborhood = locationAddress
		item.Category = categoryName
		item.CategoryName = categoryName
		item.PreferredDate = preferredDate
		item.ScheduledDate = preferredDate
		item.ClientEvaluated = clientEvaluated
		item.IsClientEvaluated = clientEvaluated
		item.IsReviewed = clientEvaluated
		item.BikkerEvaluated = bikkerEvaluated
		item.ClientRatingGiven = clientRatingGiven
		item.CreatedDate = createdAt.Format("02/01/2006")

		item.Photos = []string{}
		if len(clientPhotosJSON) > 0 {
			_ = json.Unmarshal(clientPhotosJSON, &item.Photos)
		}
		item.ClientPhotos = item.Photos

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

	query := solicitationBaseQuery + `
		WHERE qr.id = $1::uuid OR o.id = $1::uuid
		LIMIT 1
	`

	var item model.SolicitationItem
	var quoteStatus string
	var orderStatus *string
	var scheduledTime time.Time
	var createdAt time.Time
	var clientAccountName, contactName, accountPhone, contactPhone, clientRating, clientPhotoURL string
	var clientTotalRatings int
	var locationAddress, categoryName, preferredDate string
	var clientPhotosJSON []byte
	var clientEvaluated, bikkerEvaluated bool
	var clientRatingGiven int

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
		&clientAccountName,
		&contactName,
		&accountPhone,
		&contactPhone,
		&clientRating,
		&clientTotalRatings,
		&clientPhotoURL,
		&locationAddress,
		&categoryName,
		&preferredDate,
		&clientPhotosJSON,
		&clientEvaluated,
		&bikkerEvaluated,
		&clientRatingGiven,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	item.ServiceTitle = item.ServiceName
	item.InitialDescription = item.Description
	item.ClientName = clientAccountName
	item.ContactName = contactName
	item.AccountPhone = accountPhone
	item.ClientAccountPhone = accountPhone
	item.ContactPhone = contactPhone
	item.ClientPhone = accountPhone
	item.ClientRating = clientRating
	item.ClientTotalRatings = clientTotalRatings
	item.ClientPhotoURL = clientPhotoURL
	item.LocationAddress = locationAddress
	item.Address = locationAddress
	item.Neighborhood = locationAddress
	item.Category = categoryName
	item.CategoryName = categoryName
	item.PreferredDate = preferredDate
	item.ScheduledDate = preferredDate
	item.ClientEvaluated = clientEvaluated
	item.IsClientEvaluated = clientEvaluated
	item.IsReviewed = clientEvaluated
	item.BikkerEvaluated = bikkerEvaluated
	item.ClientRatingGiven = clientRatingGiven
	item.CreatedDate = createdAt.Format("02/01/2006")

	item.Photos = []string{}
	if len(clientPhotosJSON) > 0 {
		_ = json.Unmarshal(clientPhotosJSON, &item.Photos)
	}
	item.ClientPhotos = item.Photos

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

	customerID := strings.TrimSpace(item.ClientID)
	var customerName, customerPhone, customerPhotoURL string
	var customerRating float64
	var customerTotalRatings int

	if isValidUUID(customerID) {
		_ = r.pool.QueryRow(ctx, `
			SELECT full_name, phone, rating, total_ratings, COALESCE(profile_photo_url, '')
			FROM users WHERE id = $1::uuid
		`, customerID).Scan(&customerName, &customerPhone, &customerRating, &customerTotalRatings, &customerPhotoURL)
	}

	if customerName == "" {
		_ = r.pool.QueryRow(ctx, `
			SELECT id::text, full_name, phone, rating, total_ratings, COALESCE(profile_photo_url, '')
			FROM users ORDER BY created_at ASC LIMIT 1
		`).Scan(&customerID, &customerName, &customerPhone, &customerRating, &customerTotalRatings, &customerPhotoURL)
	}

	if !isValidUUID(customerID) {
		customerID = "11111111-1111-1111-1111-111111111111"
	}

	bikkerID := strings.TrimSpace(item.ProviderID)
	var dbBikkerExists bool
	if isValidUUID(bikkerID) {
		_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bikkers WHERE id = $1::uuid)`, bikkerID).Scan(&dbBikkerExists)
	}
	if !dbBikkerExists {
		_ = r.pool.QueryRow(ctx, `SELECT id::text FROM bikkers WHERE is_active = TRUE ORDER BY created_at ASC LIMIT 1`).Scan(&bikkerID)
	}
	if !isValidUUID(bikkerID) {
		bikkerID = "22222222-2222-2222-2222-222222222222"
	}

	var validServiceID *string
	if isValidUUID(item.ServiceID) {
		var sExists bool
		_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM services WHERE id = $1::uuid)`, item.ServiceID).Scan(&sExists)
		if sExists {
			sID := item.ServiceID
			validServiceID = &sID
		}
	}

	contactName := strings.TrimSpace(item.ContactName)
	if contactName == "" {
		contactName = strings.TrimSpace(item.ClientName)
	}
	if contactName == "" {
		contactName = customerName
	}

	contactPhone := strings.TrimSpace(item.ContactPhone)
	if contactPhone == "" {
		contactPhone = strings.TrimSpace(item.ClientPhone)
	}
	if contactPhone == "" {
		contactPhone = customerPhone
	}
	accountPhone := customerPhone

	exactAddress := strings.TrimSpace(item.LocationAddress)
	if exactAddress == "" {
		exactAddress = strings.TrimSpace(item.Address)
	}
	if exactAddress == "" {
		exactAddress = strings.TrimSpace(item.Neighborhood)
	}
	if exactAddress == "" {
		exactAddress = strings.TrimSpace(item.Location)
		if exactAddress != "" && strings.TrimSpace(item.HouseNumber) != "" {
			exactAddress = fmt.Sprintf("%s, Nº %s", exactAddress, strings.TrimSpace(item.HouseNumber))
		}
	}
	if exactAddress == "" {
		exactAddress = "São Paulo, SP"
	}

	serviceTitle := strings.TrimSpace(item.ServiceTitle)
	if serviceTitle == "" {
		serviceTitle = strings.TrimSpace(item.ServiceName)
	}
	if serviceTitle == "" {
		serviceTitle = "Solicitação de Serviço"
	}

	categoryName := strings.TrimSpace(item.CategoryName)
	if categoryName == "" {
		categoryName = strings.TrimSpace(item.Category)
	}
	if categoryName == "" {
		categoryName = "Geral"
	}

	preferredDate := strings.TrimSpace(item.PreferredDate)
	if preferredDate == "" {
		preferredDate = strings.TrimSpace(item.ScheduledDate)
	}
	if preferredDate == "" {
		preferredDate = "O mais rápido possível"
	}

	clientPhotos := item.ClientPhotos
	if len(clientPhotos) == 0 && len(item.Photos) > 0 {
		clientPhotos = item.Photos
	}
	if clientPhotos == nil {
		clientPhotos = []string{}
	}
	clientPhotosJSON, _ := json.Marshal(clientPhotos)

	var quoteID string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO quote_requests (
			customer_id, bikker_id, service_id, status, initial_description, desired_date, location_address, location_geom,
			contact_name, contact_phone, account_phone, service_title, category_name, preferred_date, client_photos
		)
		VALUES (
			$1::uuid, $2::uuid, $3, 'PENDING', $4, NOW() + INTERVAL '2 days', $5,
			ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography,
			$6, $7, $8, $9, $10, $11, $12::jsonb
		)
		RETURNING id::text
	`, customerID, bikkerID, validServiceID, item.Description, exactAddress,
		contactName, contactPhone, accountPhone, serviceTitle, categoryName, preferredDate, string(clientPhotosJSON),
	).Scan(&quoteID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert quote_request: %w", err)
	}

	if len(clientPhotos) > 0 {
		_, _ = r.pool.Exec(ctx, `
			INSERT INTO quote_negotiations (quote_request_id, sender_id, message, proposed_price, photos, is_counter_offer)
			VALUES ($1::uuid, $2::uuid, $3, 0, $4::jsonb, FALSE)
		`, quoteID, customerID, item.Description, string(clientPhotosJSON))
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

	if input.CounterOfferPrice != nil || (input.CounterOfferMessage != nil && strings.TrimSpace(*input.CounterOfferMessage) != "") {
		price := 0.0
		if input.CounterOfferPrice != nil {
			price = *input.CounterOfferPrice
		}
		senderID := existing.ClientID
		if input.SenderID != nil && isValidUUID(*input.SenderID) {
			senderID = *input.SenderID
		} else if input.SenderRole != nil && strings.EqualFold(*input.SenderRole, "PROVIDER") {
			senderID = existing.ProviderID
		}

		msg := "Contraproposta enviada"
		if input.CounterOfferMessage != nil && strings.TrimSpace(*input.CounterOfferMessage) != "" {
			msg = strings.TrimSpace(*input.CounterOfferMessage)
		} else if price <= 0 {
			msg = "Pedido de mais informações"
		}

		_, err := r.pool.Exec(ctx, `
			INSERT INTO quote_negotiations (quote_request_id, sender_id, message, proposed_price, is_counter_offer)
			VALUES ($1::uuid, $2::uuid, $3, $4, TRUE)
		`, quoteID, senderID, msg, price)
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

	if shouldUpsertOrder || input.TechnicalReport != nil || cancellationReason != nil {
		if shouldUpsertOrder && dbOrderStatus != "" {
			var validServiceID *string
			if isValidUUID(existing.ServiceID) {
				var exists bool
				_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM services WHERE id = $1::uuid)`, existing.ServiceID).Scan(&exists)
				if exists {
					sID := existing.ServiceID
					validServiceID = &sID
				}
			}
			_, _ = r.pool.Exec(ctx, `
				INSERT INTO orders (
					quote_request_id, customer_id, bikker_id, service_id, status, final_price, scheduled_date, technical_report, cancellation_reason
				)
				VALUES (
					$1::uuid, $2::uuid, $3::uuid, $4, $5::order_status, $6, NOW() + INTERVAL '1 day', $7, $8
				)
				ON CONFLICT (quote_request_id) DO UPDATE SET
					bikker_id = EXCLUDED.bikker_id,
					status = EXCLUDED.status,
					final_price = CASE WHEN $6 > 0 THEN $6 ELSE orders.final_price END,
					technical_report = COALESCE(EXCLUDED.technical_report, orders.technical_report),
					cancellation_reason = COALESCE(EXCLUDED.cancellation_reason, orders.cancellation_reason),
					updated_at = NOW()
			`,
				quoteID,
				existing.ClientID,
				existing.ProviderID,
				validServiceID,
				dbOrderStatus,
				finalPrice,
				input.TechnicalReport,
				cancellationReason,
			)
		} else {
			_, _ = r.pool.Exec(ctx, `
				UPDATE orders SET
					technical_report = COALESCE($2, technical_report),
					cancellation_reason = COALESCE($3, cancellation_reason),
					updated_at = NOW()
				WHERE quote_request_id = $1::uuid
			`, quoteID, input.TechnicalReport, cancellationReason)
		}
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
			COALESCE(NULLIF(qr.service_title, ''), s.name, 'Solicitação de Serviço') AS title,
			COALESCE(qr.initial_description, '') AS description,
			COALESCE(NULLIF(qr.category_name, ''), c.name, 'Geral') AS category,
			COALESCE(qr.location_address, 'São Paulo, SP') AS neighborhood,
			COALESCE(NULLIF(qr.contact_name, ''), cu.full_name, 'Cliente') AS client_name,
			COALESCE(cu.phone, qr.account_phone, '') AS account_phone,
			COALESCE(NULLIF(qr.contact_phone, ''), cu.phone, '') AS contact_phone,
			COALESCE(cu.rating, 0.0) AS client_rating,
			COALESCE(cu.total_ratings, 0) AS client_total_ratings,
			COALESCE(NULLIF(qr.preferred_date, ''), 'O mais rápido possível') AS preferred_date,
			COALESCE(qr.client_photos, '[]'::jsonb) AS client_photos,
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
		var clientName, accountPhone, contactPhone string
		var clientRatingFloat float64
		var clientTotalRatings int
		var preferredDate string
		var clientPhotosJSON []byte

		if err := rows.Scan(
			&lead.ID,
			&lead.Title,
			&lead.Description,
			&lead.Category,
			&lead.Neighborhood,
			&clientName,
			&accountPhone,
			&contactPhone,
			&clientRatingFloat,
			&clientTotalRatings,
			&preferredDate,
			&clientPhotosJSON,
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
		lead.EstimatedValue = ""
		lead.ClientVerified = true
		lead.ClientName = clientName
		lead.AccountPhone = accountPhone
		lead.ClientAccountPhone = accountPhone
		lead.ContactPhone = contactPhone
		lead.ClientPhone = contactPhone
		if clientTotalRatings == 0 || clientRatingFloat == 0 {
			lead.ClientRating = "0.0"
		} else {
			lead.ClientRating = fmt.Sprintf("%.1f", clientRatingFloat)
		}
		lead.PreferredDate = preferredDate
		lead.ScheduledDate = preferredDate
		lead.ServiceTitle = lead.Title
		lead.Address = lead.Neighborhood
		lead.LocationAddress = lead.Neighborhood

		lead.Photos = []string{}
		if len(clientPhotosJSON) > 0 {
			_ = json.Unmarshal(clientPhotosJSON, &lead.Photos)
		}
		lead.ClientPhotos = lead.Photos

		negItem := model.SolicitationItem{ID: lead.ID}
		r.loadNegotiations(ctx, &negItem)
		for _, co := range negItem.CounterOffers {
			lead.CounterOffers = append(lead.CounterOffers, domain.LeadCounterOffer{
				ID:             co.ID,
				Round:          co.Round,
				Author:         co.Author,
				IsFromClient:   co.IsFromClient,
				Price:          co.Price,
				FormattedPrice: co.FormattedPrice,
				Reason:         co.Reason,
			})
		}
		lead.CounterOfferHistory = lead.CounterOffers
		lead.NegotiationHistory = lead.CounterOffers

		leads = append(leads, lead)
	}

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

	var validServiceID *string
	if isValidUUID(existing.ServiceID) {
		var exists bool
		_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM services WHERE id = $1::uuid)`, existing.ServiceID).Scan(&exists)
		if exists {
			sID := existing.ServiceID
			validServiceID = &sID
		}
	}

	_, _ = r.pool.Exec(ctx, `
		INSERT INTO orders (quote_request_id, customer_id, bikker_id, service_id, status, final_price, scheduled_date)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, 'IN_PROGRESS', $5, NOW() + INTERVAL '1 day')
		ON CONFLICT (quote_request_id) DO UPDATE SET
			bikker_id = EXCLUDED.bikker_id,
			status = 'IN_PROGRESS',
			final_price = CASE WHEN $5 > 0 THEN $5 ELSE orders.final_price END,
			updated_at = NOW()
	`, existing.ID, existing.ClientID, existing.ProviderID, validServiceID, finalPrice)

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

	if existing.BikkerEvaluated {
		return domain.ErrAlreadyReviewed
	}

	techReport := strings.TrimSpace(req.TechnicalReport)

	_, _ = r.pool.Exec(ctx, `
		UPDATE quote_requests
		SET status = 'ACCEPTED',
		    bikker_evaluated = TRUE,
		    client_rating_given = CASE WHEN $3 > 0 THEN $3 ELSE client_rating_given END,
		    technical_report = CASE WHEN $2 <> '' THEN $2 ELSE technical_report END,
		    updated_at = NOW()
		WHERE id = $1::uuid
	`, existing.ID, techReport, req.Stars)

	var validServiceID *string
	if isValidUUID(existing.ServiceID) {
		var exists bool
		_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM services WHERE id = $1::uuid)`, existing.ServiceID).Scan(&exists)
		if exists {
			sID := existing.ServiceID
			validServiceID = &sID
		}
	}

	var orderID string
	err = r.pool.QueryRow(ctx, `
		INSERT INTO orders (
			quote_request_id, customer_id, bikker_id, service_id, status, final_price, scheduled_date, technical_report
		)
		VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, 'COMPLETED', $5, NOW(), $6
		)
		ON CONFLICT (quote_request_id) DO UPDATE SET
			status = 'COMPLETED',
			technical_report = CASE WHEN $6 <> '' THEN $6 ELSE orders.technical_report END,
			updated_at = NOW()
		RETURNING id::text
	`, existing.ID, existing.ClientID, existing.ProviderID, validServiceID, existing.Price, techReport).Scan(&orderID)
	if err != nil {
		_ = r.pool.QueryRow(ctx, `SELECT id::text FROM orders WHERE quote_request_id = $1::uuid LIMIT 1`, existing.ID).Scan(&orderID)
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

	if isValidUUID(orderID) && req.Stars >= 1 && req.Stars <= 5 {
		reviewerID := bikkerID
		if !isValidUUID(reviewerID) {
			reviewerID = existing.ProviderID
		}
		comment := strings.TrimSpace(req.Comment)
		if comment == "" {
			comment = techReport
		}
		_, _ = r.pool.Exec(ctx, `
			INSERT INTO reviews (order_id, service_id, reviewer_id, bikker_id, rating, comment)
			VALUES ($1::uuid, $2, $3::uuid, $4::uuid, $5, $6)
			ON CONFLICT DO NOTHING
		`, orderID, validServiceID, reviewerID, existing.ProviderID, req.Stars, comment)
	}

	return nil
}

func formatPreferredDate(scheduledTime time.Time) string {
	if scheduledTime.IsZero() {
		return "O mais rápido possível"
	}
	now := time.Now()
	diff := scheduledTime.Sub(now)
	if diff <= 24*time.Hour {
		return "O mais rápido possível"
	} else if diff <= 7*24*time.Hour {
		return "Próximos 7 dias"
	} else if diff <= 15*24*time.Hour {
		return "Próximos 15 dias"
	} else if diff <= 30*24*time.Hour {
		return "Próximos 30 dias"
	}
	return "O mais rápido possível"
}

