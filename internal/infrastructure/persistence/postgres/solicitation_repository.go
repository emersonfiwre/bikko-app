package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"bikko-app/internal/domain"
	"bikko-app/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SolicitationStatusUpdateInput = model.SolicitationStatusUpdateInput

type SolicitationPostgresRepository struct{ pool *pgxpool.Pool }

func NewSolicitationPostgresRepository(pool *pgxpool.Pool) *SolicitationPostgresRepository {
	return &SolicitationPostgresRepository{pool: pool}
}
func (r *SolicitationPostgresRepository) HasPool() bool { return r != nil && r.pool != nil }
func isValidUUID(id string) bool                        { _, err := uuid.Parse(id); return err == nil }

type contractQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

const solicitationBaseQuery = `SELECT qr.id::text, qr.customer_id::text, qr.bikker_id::text,
 COALESCE(qr.service_id::text,''), COALESCE(NULLIF(qr.service_title,''),s.name,''),
 COALESCE(pu.full_name,''), COALESCE(b.profession,''), COALESCE(b.rating,0)::text,
 COALESCE(pu.profile_photo_url,''), qr.status::text, o.status::text, COALESCE(o.final_price,0),
 COALESCE(qr.preferred_date,''), COALESCE(qr.initial_description,''),
 COALESCE(o.technical_report,qr.technical_report,''), COALESCE(o.cancellation_reason,qr.cancellation_reason), qr.created_at,
 COALESCE(cu.full_name,''), COALESCE(NULLIF(qr.contact_name,''),cu.full_name,''),
 COALESCE(cu.phone,''), COALESCE(NULLIF(qr.contact_phone,''),cu.phone,''),
 COALESCE(cu.rating,0)::text, COALESCE(cu.total_ratings,0), COALESCE(cu.profile_photo_url,''),
 COALESCE(qr.location_address,''), COALESCE(NULLIF(qr.category_name,''),c.name,''),
 COALESCE(qr.client_photos,'[]'::jsonb), COALESCE(qr.client_evaluated,FALSE),
 COALESCE(qr.bikker_evaluated,FALSE), COALESCE(qr.client_rating_given,0),
 ST_Y(qr.location_geom::geometry),ST_X(qr.location_geom::geometry),
 COALESCE(ST_Distance(qr.location_geom,b.location_geom)/1000,0)
 FROM quote_requests qr
 LEFT JOIN orders o ON o.quote_request_id=qr.id
 LEFT JOIN services s ON s.id=qr.service_id
 LEFT JOIN categories c ON c.id=s.category_id
 LEFT JOIN bikkers b ON b.id=qr.bikker_id
 LEFT JOIN users pu ON pu.id=qr.bikker_id
 LEFT JOIN users cu ON cu.id=qr.customer_id `

func mapDBStatusToAppStatus(quote string, order *string, _ bool, _ int) string {
	if order != nil {
		switch *order {
		case "COMPLETED":
			return "COMPLETED"
		case "CANCELLED":
			return "CANCELLED"
		case "IN_PROGRESS", "SCHEDULED":
			return "ACTIVE"
		}
	}
	switch quote {
	case "ACCEPTED":
		return "ACTIVE"
	case "REJECTED", "EXPIRED":
		return "CANCELLED"
	case "NEGOTIATING":
		return "NEGOTIATING"
	default:
		return "PENDING_BUDGET"
	}
}

func scanSolicitation(row pgx.Row) (*model.SolicitationItem, error) {
	var s model.SolicitationItem
	var quote string
	var order *string
	var photos []byte
	err := row.Scan(&s.ID, &s.ClientID, &s.ProviderID, &s.ServiceID, &s.ServiceName, &s.ProviderName, &s.ProviderTitle,
		&s.ProviderRating, &s.ProviderPhotoURL, &quote, &order, &s.Price, &s.PreferredDate, &s.Description, &s.TechnicalReport,
		&s.CancellationReason, &s.CreatedAt, &s.ClientName, &s.ContactName, &s.AccountPhone, &s.ContactPhone,
		&s.ClientRating, &s.ClientTotalRatings, &s.ClientPhotoURL, &s.LocationAddress, &s.CategoryName, &photos,
		&s.ClientEvaluated, &s.BikkerEvaluated, &s.ClientRatingGiven, &s.Latitude, &s.Longitude, &s.DistanceKm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("read solicitation: %w", err)
	}
	if err = json.Unmarshal(photos, &s.Photos); err != nil {
		return nil, fmt.Errorf("decode solicitation photos: %w", err)
	}
	if s.Photos == nil {
		s.Photos = []string{}
	}
	s.ClientPhotos = append([]string{}, s.Photos...)
	s.ServiceTitle = s.ServiceName
	s.InitialDescription = s.Description
	s.ClientAccountPhone = s.AccountPhone
	s.ClientPhone = s.AccountPhone
	s.Address = s.LocationAddress
	s.Neighborhood = s.LocationAddress
	s.Category = s.CategoryName
	s.ScheduledDate = s.PreferredDate
	s.CreatedDate = s.CreatedAt.Format("02/01/2006")
	s.IsClientEvaluated = s.ClientEvaluated
	s.IsReviewed = s.ClientEvaluated
	s.Status = mapDBStatusToAppStatus(quote, order, s.Price > 0, 0)
	if s.Status == "CANCELLED" {
		by := "PROVIDER"
		if quote == "EXPIRED" {
			by = "CLIENT"
		}
		s.CancelledBy = &by
	}
	return &s, nil
}

func formatBRLAmount(amount float64) string {
	cents := int64(math.Round(amount * 100))
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	whole := strconv.FormatInt(cents/100, 10)
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "." + whole[i:]
	}
	return fmt.Sprintf("%s%s,%02d", sign, whole, cents%100)
}

// loadNegotiationsBatch releases its result before any subsequent query. Lists do not issue N+1 reads.
func loadNegotiationsBatch(ctx context.Context, q contractQuerier, items []*model.SolicitationItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	byID := map[string]*model.SolicitationItem{}
	for _, s := range items {
		ids = append(ids, s.ID)
		byID[s.ID] = s
		s.CounterOffers = []model.CounterOfferItem{}
	}
	rows, err := q.Query(ctx, `SELECT quote_request_id::text,id::text,sender_id::text,COALESCE(message,''),COALESCE(proposed_price,0),COALESCE(photos,'[]'::jsonb),is_counter_offer FROM quote_negotiations WHERE quote_request_id=ANY($1::uuid[]) ORDER BY created_at,id`, ids)
	if err != nil {
		return fmt.Errorf("read negotiations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var quoteID, id, sender, msg string
		var price float64
		var photosJSON []byte
		var counter bool
		if err = rows.Scan(&quoteID, &id, &sender, &msg, &price, &photosJSON, &counter); err != nil {
			return err
		}
		s := byID[quoteID]
		photos := []string{}
		if err = json.Unmarshal(photosJSON, &photos); err != nil {
			return err
		}
		author := "CLIENT"
		if sender == s.ProviderID {
			author = "PROVIDER"
		}
		if counter && price > 0 {
			s.RenegotiationCount++
		}
		round := s.RenegotiationCount
		offer := model.CounterOfferItem{ID: id, Round: round, Author: author, IsFromClient: author == "CLIENT", Price: price, Reason: msg, Photos: photos}
		if price > 0 {
			offer.FormattedPrice = formatBRLAmount(price)
			if author == "PROVIDER" && s.Status != "ACTIVE" && s.Status != "COMPLETED" && s.Status != "CANCELLED" {
				s.Price = price
			}
			if counter {
				v := price
				s.CounterOfferPrice = &v
			}
			if s.Status == "PENDING_BUDGET" || s.Status == "NEGOTIATING" || s.Status == "BUDGET_RECEIVED" {
				if author == "CLIENT" {
					s.Status = "NEGOTIATING"
				} else {
					s.Status = "BUDGET_RECEIVED"
				}
			}
		}
		if msg != "" {
			v := msg
			s.CounterOfferMessage = &v
		}
		s.CounterOffers = append(s.CounterOffers, offer)
		for _, photo := range photos {
			found := false
			for _, p := range s.Photos {
				if p == photo {
					found = true
					break
				}
			}
			if !found {
				s.Photos = append(s.Photos, photo)
			}
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for _, s := range items {
		s.ClientPhotos = append([]string{}, s.Photos...)
		s.CounterOfferHistory = s.CounterOffers
		s.NegotiationHistory = s.CounterOffers
	}
	return nil
}

func (r *SolicitationPostgresRepository) GetSolicitationsByUser(ctx context.Context, actor string) ([]model.SolicitationItem, error) {
	return r.GetSolicitationsPage(ctx, actor, 50, 0)
}
func (r *SolicitationPostgresRepository) GetSolicitationsPage(ctx context.Context, actor string, limit, offset int) ([]model.SolicitationItem, error) {
	return r.listSolicitations(ctx, actor, limit, offset, false)
}
func (r *SolicitationPostgresRepository) listSolicitations(ctx context.Context, actor string, limit, offset int, pendingOnly bool) ([]model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, domain.ErrUnavailable
	}
	if !isValidUUID(actor) {
		return nil, domain.ErrForbidden
	}
	if limit < 1 || limit > 100 || offset < 0 {
		return nil, domain.ErrInvalidInput
	}
	where := ` WHERE (qr.customer_id=$1::uuid OR qr.bikker_id=$1::uuid)`
	if pendingOnly {
		where = ` WHERE qr.bikker_id=$1::uuid AND qr.status IN ('PENDING','NEGOTIATING')`
	}
	rows, err := r.pool.Query(ctx, solicitationBaseQuery+where+` ORDER BY qr.created_at DESC,qr.id LIMIT $2 OFFSET $3`, actor, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list solicitations: %w", err)
	}
	pointers := []*model.SolicitationItem{}
	for rows.Next() {
		s, e := scanSolicitation(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		pointers = append(pointers, s)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = loadNegotiationsBatch(ctx, r.pool, pointers); err != nil {
		return nil, err
	}
	result := make([]model.SolicitationItem, 0, len(pointers))
	for _, s := range pointers {
		result = append(result, *s)
	}
	return result, nil
}
func getSolicitation(ctx context.Context, q contractQuerier, id, actor string) (*model.SolicitationItem, error) {
	if !isValidUUID(id) {
		return nil, domain.ErrNotFound
	}
	query := solicitationBaseQuery + ` WHERE (qr.id=$1::uuid OR o.id=$1::uuid)`
	args := []any{id}
	if actor != "" {
		if !isValidUUID(actor) {
			return nil, domain.ErrForbidden
		}
		query += ` AND (qr.customer_id=$2::uuid OR qr.bikker_id=$2::uuid)`
		args = append(args, actor)
	}
	s, err := scanSolicitation(q.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	if err = loadNegotiationsBatch(ctx, q, []*model.SolicitationItem{s}); err != nil {
		return nil, err
	}
	return s, nil
}
func (r *SolicitationPostgresRepository) GetSolicitationByID(ctx context.Context, id string) (*model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, domain.ErrUnavailable
	}
	return getSolicitation(ctx, r.pool, id, "")
}
func (r *SolicitationPostgresRepository) GetSolicitationForUser(ctx context.Context, id, actor string) (*model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, domain.ErrUnavailable
	}
	if !isValidUUID(actor) {
		return nil, domain.ErrForbidden
	}
	return getSolicitation(ctx, r.pool, id, actor)
}

func validCoordinates(lat, lon *float64) bool {
	return lat != nil && lon != nil && !math.IsNaN(*lat) && !math.IsNaN(*lon) && !math.IsInf(*lat, 0) && !math.IsInf(*lon, 0) && *lat >= -90 && *lat <= 90 && *lon >= -180 && *lon <= 180
}
func preferredTime(label string) time.Time {
	days := 0
	switch label {
	case "Próximos 7 dias":
		days = 7
	case "Próximos 15 dias":
		days = 15
	case "Próximos 30 dias":
		days = 30
	}
	return time.Now().UTC().AddDate(0, 0, days)
}

func (r *SolicitationPostgresRepository) CreateSolicitation(ctx context.Context, s model.SolicitationItem) (*model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, domain.ErrUnavailable
	}
	if !isValidUUID(s.ClientID) || !isValidUUID(s.ProviderID) || s.ClientID == s.ProviderID || strings.TrimSpace(s.Description) == "" || len([]rune(s.Description)) > 280 || !validCoordinates(s.Latitude, s.Longitude) || strings.TrimSpace(s.LocationAddress) == "" || len(s.Photos) > 10 || len(s.RequestKey) > 128 {
		return nil, domain.ErrInvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var fingerprint string
	if s.RequestKey != "" {
		requestBody, e := json.Marshal(struct {
			Provider, Service, Title, Description, Address, ContactName, ContactPhone, Preferred string
			Latitude, Longitude                                                                  *float64
			Photos                                                                               []string
		}{s.ProviderID, s.ServiceID, s.ServiceName, s.Description, s.LocationAddress, s.ContactName, s.ContactPhone, s.PreferredDate, s.Latitude, s.Longitude, s.Photos})
		if e != nil {
			return nil, e
		}
		fingerprint = fmt.Sprintf("%x", sha256.Sum256(requestBody))
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, s.ClientID+":"+s.RequestKey); err != nil {
			return nil, err
		}
		var previousID, previousFingerprint string
		err = tx.QueryRow(ctx, `SELECT id::text,COALESCE(request_fingerprint,'') FROM quote_requests WHERE customer_id=$1::uuid AND request_key=$2`, s.ClientID, s.RequestKey).Scan(&previousID, &previousFingerprint)
		if err == nil {
			if previousFingerprint != fingerprint {
				return nil, domain.ErrConflict
			}
			if err = tx.Commit(ctx); err != nil {
				return nil, err
			}
			return r.GetSolicitationForUser(ctx, previousID, s.ClientID)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	var active bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bikkers b JOIN users u ON u.id=b.id WHERE b.id=$1::uuid AND b.is_active AND u.deleted_at IS NULL AND u.is_bikker)`, s.ProviderID).Scan(&active); err != nil {
		return nil, err
	}
	if !active {
		return nil, domain.ErrNotFound
	}
	var serviceID *string
	if s.ServiceID != "" {
		if !isValidUUID(s.ServiceID) {
			return nil, domain.ErrInvalidInput
		}
		var name, category string
		if err = tx.QueryRow(ctx, `SELECT s.name,c.name FROM services s JOIN categories c ON c.id=s.category_id WHERE s.id=$1::uuid AND s.bikker_id=$2::uuid AND s.is_active AND s.deleted_at IS NULL`, s.ServiceID, s.ProviderID).Scan(&name, &category); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrNotFound
			}
			return nil, err
		}
		serviceID = &s.ServiceID
		s.ServiceName = name
		s.CategoryName = category
	}
	var customerName, customerPhone string
	if err = tx.QueryRow(ctx, `SELECT full_name,COALESCE(phone,'') FROM users WHERE id=$1::uuid AND deleted_at IS NULL`, s.ClientID).Scan(&customerName, &customerPhone); err != nil {
		return nil, err
	}
	if s.ContactName == "" {
		s.ContactName = customerName
	}
	if s.ContactPhone == "" {
		s.ContactPhone = customerPhone
	}
	photos, err := json.Marshal(s.Photos)
	if err != nil {
		return nil, err
	}
	if s.Photos == nil {
		photos = []byte("[]")
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO quote_requests(customer_id,bikker_id,service_id,status,initial_description,desired_date,location_address,location_geom,contact_name,contact_phone,account_phone,service_title,category_name,preferred_date,client_photos,request_key,request_fingerprint) VALUES($1::uuid,$2::uuid,$3,'PENDING',$4,$5,$6,ST_SetSRID(ST_MakePoint($7,$8),4326)::geography,$9,$10,$11,$12,$13,$14,$15::jsonb,NULLIF($16,''),NULLIF($17,'')) RETURNING id::text`, s.ClientID, s.ProviderID, serviceID, s.Description, preferredTime(s.PreferredDate), s.LocationAddress, *s.Longitude, *s.Latitude, s.ContactName, s.ContactPhone, customerPhone, s.ServiceName, s.CategoryName, s.PreferredDate, string(photos), s.RequestKey, fingerprint).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create solicitation: %w", err)
	}
	if err = enqueueContractNotification(ctx, tx, s.ProviderID, id, "Nova solicitação", "Você recebeu uma solicitação de orçamento.", "new_solicitation"); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetSolicitationForUser(ctx, id, s.ClientID)
}

type lockedContract struct {
	id, client, provider, status string
	service                      *string
	desired                      time.Time
	orderStatus                  *string
}

func lockContract(ctx context.Context, tx pgx.Tx, id, actor string) (*lockedContract, error) {
	if !isValidUUID(id) || !isValidUUID(actor) {
		return nil, domain.ErrNotFound
	}
	var q lockedContract
	err := tx.QueryRow(ctx, `SELECT qr.id::text,qr.customer_id::text,qr.bikker_id::text,qr.service_id::text,qr.status::text,qr.desired_date FROM quote_requests qr WHERE (qr.id=$1::uuid OR EXISTS(SELECT 1 FROM orders o WHERE o.quote_request_id=qr.id AND o.id=$1::uuid)) AND (qr.customer_id=$2::uuid OR qr.bikker_id=$2::uuid) FOR UPDATE OF qr`, id, actor).Scan(&q.id, &q.client, &q.provider, &q.service, &q.status, &q.desired)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	var orderStatus string
	err = tx.QueryRow(ctx, `SELECT status::text FROM orders WHERE quote_request_id=$1::uuid FOR UPDATE`, q.id).Scan(&orderStatus)
	if err == nil {
		q.orderStatus = &orderStatus
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return &q, nil
}

type latestPrice struct{ id, sender, amount, message string }

func latestPriced(ctx context.Context, tx pgx.Tx, id string) (*latestPrice, error) {
	var p latestPrice
	err := tx.QueryRow(ctx, `SELECT id::text,sender_id::text,proposed_price::text,COALESCE(message,'') FROM quote_negotiations WHERE quote_request_id=$1::uuid AND proposed_price>0 ORDER BY created_at DESC,id DESC LIMIT 1`, id).Scan(&p.id, &p.sender, &p.amount, &p.message)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}
func moneyAmount(price float64) (string, error) {
	if math.IsNaN(price) || math.IsInf(price, 0) || price <= 0 || price > 99999999.99 {
		return "", domain.ErrInvalidInput
	}
	cents := math.Round(price * 100)
	if math.Abs(price*100-cents) > 0.00001 {
		return "", domain.ErrInvalidInput
	}
	return fmt.Sprintf("%.2f", cents/100), nil
}
func contractTerminal(q *lockedContract) bool {
	return q.status == "REJECTED" || q.status == "EXPIRED" || (q.orderStatus != nil && (*q.orderStatus == "COMPLETED" || *q.orderStatus == "CANCELLED"))
}

func (r *SolicitationPostgresRepository) UpdateSolicitationStatus(ctx context.Context, id string, in SolicitationStatusUpdateInput) (*model.SolicitationItem, error) {
	if !r.HasPool() {
		return nil, domain.ErrUnavailable
	}
	if len(in.RequestKey) > 128 {
		return nil, domain.ErrInvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	q, err := lockContract(ctx, tx, id, in.ActorID)
	if err != nil {
		return nil, err
	}
	if in.ActorID == q.provider {
		var active bool
		// Availability controls new work only. A provider who went offline must
		// still be able to finish or negotiate an already assigned contract.
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bikkers b JOIN users u ON u.id=b.id WHERE b.id=$1::uuid AND u.is_bikker AND u.deleted_at IS NULL)`, in.ActorID).Scan(&active); err != nil {
			return nil, err
		}
		if !active {
			return nil, domain.ErrForbidden
		}
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	hasMessage := in.CounterOfferMessage != nil && strings.TrimSpace(*in.CounterOfferMessage) != ""
	hasPrice := in.CounterOfferPrice != nil && *in.CounterOfferPrice > 0
	if in.CounterOfferPrice != nil && (*in.CounterOfferPrice < 0 || math.IsNaN(*in.CounterOfferPrice) || math.IsInf(*in.CounterOfferPrice, 0)) {
		return nil, domain.ErrInvalidInput
	}
	if in.RequestKey != "" {
		var oldPrice *float64
		var oldMessage string
		var oldPhotos []byte
		err = tx.QueryRow(ctx, `SELECT proposed_price,COALESCE(message,''),COALESCE(photos,'[]'::jsonb) FROM quote_negotiations WHERE quote_request_id=$1::uuid AND sender_id=$2::uuid AND request_key=$3`, q.id, in.ActorID, in.RequestKey).Scan(&oldPrice, &oldMessage, &oldPhotos)
		if err == nil {
			message := ""
			if in.CounterOfferMessage != nil {
				message = strings.TrimSpace(*in.CounterOfferMessage)
			}
			expectedPhotos, e := json.Marshal(in.Photos)
			if e != nil {
				return nil, e
			}
			if in.Photos == nil {
				expectedPhotos = []byte("[]")
			}
			var actualPhotos, requestedPhotos []string
			if e = json.Unmarshal(oldPhotos, &actualPhotos); e != nil {
				return nil, e
			}
			if e = json.Unmarshal(expectedPhotos, &requestedPhotos); e != nil {
				return nil, e
			}
			equalPhotos := len(actualPhotos) == len(requestedPhotos)
			for i := range actualPhotos {
				if !equalPhotos || actualPhotos[i] != requestedPhotos[i] {
					equalPhotos = false
					break
				}
			}
			samePrice := (!hasPrice && oldPrice == nil) || (hasPrice && oldPrice != nil && math.Abs(*in.CounterOfferPrice-*oldPrice) < 0.00001)
			if !samePrice || message != oldMessage || !equalPhotos || (status != "NEGOTIATING" && status != "BUDGET_RECEIVED" && status != "PENDING_BUDGET") {
				return nil, domain.ErrConflict
			}
			if err = tx.Commit(ctx); err != nil {
				return nil, err
			}
			return r.GetSolicitationForUser(ctx, q.id, in.ActorID)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	switch status {
	case "BUDGET_RECEIVED", "NEGOTIATING", "PENDING_BUDGET":
		if contractTerminal(q) || q.status == "ACCEPTED" {
			return nil, domain.ErrConflict
		}
		if !hasMessage && !hasPrice && len(in.Photos) == 0 {
			return nil, domain.ErrInvalidInput
		}
		if len(in.Photos) > 10 {
			return nil, domain.ErrInvalidInput
		}
		latest, e := latestPriced(ctx, tx, q.id)
		if e != nil {
			return nil, e
		}
		var amount any
		counter := false
		message := ""
		if in.CounterOfferMessage != nil {
			message = strings.TrimSpace(*in.CounterOfferMessage)
		}
		if len([]rune(message)) > 280 {
			return nil, domain.ErrInvalidInput
		}
		if hasPrice {
			a, e := moneyAmount(*in.CounterOfferPrice)
			if e != nil {
				return nil, e
			}
			amount = a
			if latest == nil && in.ActorID != q.provider {
				return nil, domain.ErrConflict
			}
			if latest != nil && latest.sender == in.ActorID {
				if latest.amount == a && latest.message == message {
					if err = tx.Commit(ctx); err != nil {
						return nil, err
					}
					return r.GetSolicitationForUser(ctx, q.id, in.ActorID)
				}
				return nil, domain.ErrConflict
			}
			counter = latest != nil
			var count int
			if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM quote_negotiations WHERE quote_request_id=$1::uuid AND is_counter_offer AND proposed_price>0`, q.id).Scan(&count); err != nil {
				return nil, err
			}
			if counter && count >= 5 {
				return nil, domain.ErrConflict
			}
		}
		photos, e := json.Marshal(in.Photos)
		if e != nil {
			return nil, e
		}
		if in.Photos == nil {
			photos = []byte("[]")
		}
		_, err = tx.Exec(ctx, `INSERT INTO quote_negotiations(quote_request_id,sender_id,message,proposed_price,photos,is_counter_offer,created_at,request_key) VALUES($1::uuid,$2::uuid,$3,$4,$5::jsonb,$6,clock_timestamp(),NULLIF($7,''))`, q.id, in.ActorID, message, amount, string(photos), counter, in.RequestKey)
		if err != nil {
			return nil, err
		}
		if hasPrice {
			_, err = tx.Exec(ctx, `UPDATE quote_requests SET status='NEGOTIATING',updated_at=NOW() WHERE id=$1::uuid`, q.id)
			if err != nil {
				return nil, err
			}
		}
	case "ACTIVE", "ACCEPTED", "SCHEDULED", "IN_PROGRESS":
		if contractTerminal(q) {
			return nil, domain.ErrConflict
		}
		if q.status == "ACCEPTED" {
			if err = tx.Commit(ctx); err != nil {
				return nil, err
			}
			return r.GetSolicitationForUser(ctx, q.id, in.ActorID)
		}
		latest, e := latestPriced(ctx, tx, q.id)
		if e != nil {
			return nil, e
		}
		if latest == nil || latest.sender == in.ActorID {
			return nil, domain.ErrConflict
		}
		// The accepted price is read from the last persisted proposal. Request price is never authoritative.
		_, err = tx.Exec(ctx, `INSERT INTO orders(quote_request_id,customer_id,bikker_id,service_id,status,final_price,scheduled_date,accepted_negotiation_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4,'IN_PROGRESS',$5,$6,$7::uuid)`, q.id, q.client, q.provider, q.service, latest.amount, q.desired, latest.id)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `UPDATE quote_requests SET status='ACCEPTED',updated_at=NOW() WHERE id=$1::uuid`, q.id)
		if err != nil {
			return nil, err
		}
	case "COMPLETED":
		if q.orderStatus != nil && *q.orderStatus == "COMPLETED" {
			if err = tx.Commit(ctx); err != nil {
				return nil, err
			}
			return r.GetSolicitationForUser(ctx, q.id, in.ActorID)
		}
		if q.status != "ACCEPTED" || q.orderStatus == nil || (*q.orderStatus != "IN_PROGRESS" && *q.orderStatus != "SCHEDULED") {
			return nil, domain.ErrConflict
		}
		report := ""
		if in.TechnicalReport != nil {
			report = strings.TrimSpace(*in.TechnicalReport)
		}
		if len([]rune(report)) > 280 {
			return nil, domain.ErrInvalidInput
		}
		_, err = tx.Exec(ctx, `UPDATE orders SET status='COMPLETED',technical_report=CASE WHEN $2<>'' THEN $2 ELSE technical_report END,updated_at=NOW() WHERE quote_request_id=$1::uuid`, q.id, report)
		if err != nil {
			return nil, err
		}
	case "CANCELLED", "REJECTED":
		if q.status == "REJECTED" || q.status == "EXPIRED" {
			if err = tx.Commit(ctx); err != nil {
				return nil, err
			}
			return r.GetSolicitationForUser(ctx, q.id, in.ActorID)
		}
		if contractTerminal(q) {
			return nil, domain.ErrConflict
		}
		reason := ""
		if in.CancellationReason != nil {
			reason = strings.TrimSpace(*in.CancellationReason)
		}
		if len([]rune(reason)) > 280 {
			return nil, domain.ErrInvalidInput
		}
		target := "REJECTED"
		if in.ActorID == q.client {
			target = "EXPIRED"
		}
		_, err = tx.Exec(ctx, `UPDATE quote_requests SET status=$2::quote_status,cancellation_reason=$3,updated_at=NOW() WHERE id=$1::uuid`, q.id, target, reason)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `UPDATE orders SET status='CANCELLED',cancellation_reason=$2,updated_at=NOW() WHERE quote_request_id=$1::uuid`, q.id, reason)
		if err != nil {
			return nil, err
		}
	default:
		return nil, domain.ErrInvalidInput
	}
	target := q.provider
	if in.ActorID == target {
		target = q.client
	}
	if err = enqueueContractNotification(ctx, tx, target, q.id, "Solicitação atualizada", "Há uma atualização no seu pedido.", "solicitation_updated"); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit solicitation command: %w", err)
	}
	return r.GetSolicitationForUser(ctx, q.id, in.ActorID)
}
func (r *SolicitationPostgresRepository) SubmitProviderProposal(ctx context.Context, id, actor string, price float64, note string) (*model.SolicitationItem, error) {
	return r.UpdateSolicitationStatus(ctx, id, SolicitationStatusUpdateInput{ActorID: actor, Status: "BUDGET_RECEIVED", CounterOfferPrice: &price, CounterOfferMessage: &note})
}
func (r *SolicitationPostgresRepository) AcceptCounterOffer(ctx context.Context, id, actor string, _ float64, _ string) (*model.SolicitationItem, error) {
	return r.UpdateSolicitationStatus(ctx, id, SolicitationStatusUpdateInput{ActorID: actor, Status: "ACTIVE"})
}

func (r *SolicitationPostgresRepository) GetProviderFeed(ctx context.Context, actor string) ([]domain.OpportunityLead, error) {
	if !r.HasPool() {
		return nil, domain.ErrUnavailable
	}
	if !isValidUUID(actor) {
		return nil, domain.ErrForbidden
	}
	var active bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bikkers b JOIN users u ON u.id=b.id WHERE b.id=$1::uuid AND b.is_active AND u.deleted_at IS NULL AND u.is_bikker)`, actor).Scan(&active)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, domain.ErrForbidden
	}
	items, err := r.listSolicitations(ctx, actor, 50, 0, true)
	if err != nil {
		return nil, err
	}
	leads := []domain.OpportunityLead{}
	for _, s := range items {
		if s.ProviderID != actor || (s.Status != "PENDING_BUDGET" && s.Status != "BUDGET_RECEIVED" && s.Status != "NEGOTIATING") {
			continue
		}
		lead := domain.OpportunityLead{ID: s.ID, Title: s.ServiceName, ServiceTitle: s.ServiceName, Description: s.Description, Category: s.CategoryName, Urgency: "SCHEDULED", Photos: s.Photos, ClientPhotos: s.ClientPhotos, PriceType: "BUDGET_TO_NEGOTIATE", DistanceKm: s.DistanceKm, Neighborhood: s.Neighborhood, Address: s.Address, LocationAddress: s.LocationAddress, ClientName: s.ClientName, ContactName: s.ContactName, ClientPhone: s.AccountPhone, ContactPhone: s.ContactPhone, AccountPhone: s.AccountPhone, ClientAccountPhone: s.AccountPhone, ClientRating: s.ClientRating, CreatedAt: s.CreatedAt, ScheduledDate: s.PreferredDate, PreferredDate: s.PreferredDate}
		for _, co := range s.CounterOffers {
			lead.CounterOffers = append(lead.CounterOffers, domain.LeadCounterOffer{ID: co.ID, Round: co.Round, Author: co.Author, IsFromClient: co.IsFromClient, Price: co.Price, FormattedPrice: co.FormattedPrice, Reason: co.Reason})
		}
		lead.CounterOfferHistory = lead.CounterOffers
		lead.NegotiationHistory = lead.CounterOffers
		leads = append(leads, lead)
	}
	return leads, nil
}

func (r *SolicitationPostgresRepository) SubmitClientEvaluation(ctx context.Context, id, actor string, req domain.ClientEvaluationRequest) error {
	if !r.HasPool() {
		return domain.ErrUnavailable
	}
	if req.Stars < 1 || req.Stars > 5 || len([]rune(req.TechnicalReport)) > 280 || len([]rune(req.Comment)) > 120 || len(req.RequestKey) > 128 {
		return domain.ErrInvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q, err := lockContract(ctx, tx, id, actor)
	if err != nil {
		return err
	}
	if q.provider != actor {
		return domain.ErrForbidden
	}
	if q.orderStatus == nil || *q.orderStatus != "COMPLETED" {
		return domain.ErrOrderNotCompleted
	}
	var orderID string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM orders WHERE quote_request_id=$1::uuid`, q.id).Scan(&orderID); err != nil {
		return err
	}
	comment := strings.TrimSpace(req.Comment)
	if len(req.Tags) > 0 {
		comment = strings.TrimSpace("[" + strings.Join(req.Tags, ", ") + "] " + comment)
	}
	var fingerprint string
	if req.RequestKey != "" {
		fingerprint, err = clientEvaluationFingerprint(orderID, req, comment)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, actor+":"+req.RequestKey); err != nil {
			return err
		}
		var previousOrderID, previousFingerprint string
		err = tx.QueryRow(ctx, `SELECT order_id::text,COALESCE(request_fingerprint,'') FROM client_reviews WHERE bikker_id=$1::uuid AND request_key=$2 FOR UPDATE`, actor, req.RequestKey).
			Scan(&previousOrderID, &previousFingerprint)
		if err == nil {
			if previousOrderID != orderID || previousFingerprint != fingerprint {
				return domain.ErrConflict
			}
			return tx.Commit(ctx)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	var lockedUser string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM users WHERE id=$1::uuid FOR UPDATE`, q.client).Scan(&lockedUser); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO client_reviews(order_id,client_id,bikker_id,rating,comment,technical_report,request_key,request_fingerprint) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,NULLIF($7,''),NULLIF($8,'')) ON CONFLICT(order_id,bikker_id) DO NOTHING`, orderID, q.client, q.provider, req.Stars, comment, req.TechnicalReport, req.RequestKey, fingerprint)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAlreadyReviewed
	}
	_, err = tx.Exec(ctx, `UPDATE quote_requests SET bikker_evaluated=TRUE,technical_report=CASE WHEN $2<>'' THEN $2 ELSE technical_report END,updated_at=NOW() WHERE id=$1::uuid`, q.id, req.TechnicalReport)
	if err != nil {
		return err
	}
	if err = recomputeUserRating(ctx, tx, q.client); err != nil {
		return err
	}
	if req.TechnicalReport != "" {
		_, err = tx.Exec(ctx, `UPDATE orders SET technical_report=$2,updated_at=NOW() WHERE id=$1::uuid`, orderID, req.TechnicalReport)
		if err != nil {
			return err
		}
	}
	if err = enqueueContractNotification(ctx, tx, q.client, q.id, "Nova avaliação", "O profissional avaliou o atendimento.", "client_review_received"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func clientEvaluationFingerprint(orderID string, req domain.ClientEvaluationRequest, comment string) (string, error) {
	payload, err := json.Marshal(struct {
		OrderID         string `json:"order_id"`
		Stars           int    `json:"stars"`
		Comment         string `json:"comment"`
		TechnicalReport string `json:"technical_report"`
	}{
		OrderID:         orderID,
		Stars:           req.Stars,
		Comment:         comment,
		TechnicalReport: strings.TrimSpace(req.TechnicalReport),
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(payload)), nil
}

func enqueueContractNotification(ctx context.Context, tx pgx.Tx, userID, quoteID, title, body, event string) error {
	data, err := json.Marshal(map[string]string{"solicitation_id": quoteID, "type": event})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO notification_outbox(user_id,title,body,data) VALUES($1::uuid,$2,$3,$4::jsonb)`, userID, title, body, string(data))
	return err
}
