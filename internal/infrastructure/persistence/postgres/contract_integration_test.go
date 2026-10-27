package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"bikko-app/internal/domain"
	"bikko-app/internal/model"
	"bikko-app/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TEST_DATABASE_URL must name a disposable database; these tests only mutate their own fixtures.
func contractTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not configured: PostgreSQL/PostGIS integration not executed")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err = migrations.Apply(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	return pool
}
func contractUsers(t *testing.T, pool *pgxpool.Pool) (string, string, string, string) {
	t.Helper()
	ctx := context.Background()
	ids := []string{uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()}
	for i, id := range ids {
		provider := i == 1 || i == 3
		_, err := pool.Exec(ctx, `INSERT INTO users(id,full_name,email,phone,is_bikker) VALUES($1::uuid,$2,$3,$4,$5)`, id, "Integration User", id+"@test.invalid", nil, provider)
		if err != nil {
			t.Fatal(err)
		}
		if provider {
			_, err = pool.Exec(ctx, `INSERT INTO bikkers(id,profession,location_name,location_geom,is_active) VALUES($1::uuid,'Electrician','Pinheiros',ST_SetSRID(ST_MakePoint(-46.6914,-23.5614),4326)::geography,TRUE)`, id)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	return ids[0], ids[1], ids[2], ids[3]
}
func contractQuote(t *testing.T, pool *pgxpool.Pool, client, provider string) *model.SolicitationItem {
	t.Helper()
	lat, lon := -23.5641, -46.6823
	item, err := NewSolicitationPostgresRepository(pool).CreateSolicitation(context.Background(), model.SolicitationItem{ClientID: client, ProviderID: provider, ServiceName: "Electric repair", Description: "Repair lights", PreferredDate: "Próximos 7 dias", LocationAddress: "Pinheiros SP", Latitude: &lat, Longitude: &lon, Photos: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	return item
}
func commandQuote(t *testing.T, repo *SolicitationPostgresRepository, id, actor, status string, price *float64, key string) *model.SolicitationItem {
	t.Helper()
	note := "Offer " + key
	item, err := repo.UpdateSolicitationStatus(context.Background(), id, model.SolicitationStatusUpdateInput{ActorID: actor, Status: status, CounterOfferPrice: price, CounterOfferMessage: &note, RequestKey: key})
	if err != nil {
		t.Fatal(err)
	}
	return item
}
func TestContractLifecycleIsolationAndFiveCounters(t *testing.T) {
	pool := contractTestPool(t)
	ctx := context.Background()
	client, provider, otherClient, otherProvider := contractUsers(t, pool)
	repo := NewSolicitationPostgresRepository(pool)
	quote := contractQuote(t, pool, client, provider)
	if quote.DistanceKm < 0.5 || quote.DistanceKm > 2 {
		t.Fatalf("coordinates were replaced: %.3fkm", quote.DistanceKm)
	}
	// Going offline removes the provider from new work, but must not strand an
	// already assigned contract or prevent it from being completed.
	if _, err := pool.Exec(ctx, `UPDATE bikkers SET is_active=FALSE WHERE id=$1::uuid`, provider); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetProviderFeed(ctx, provider); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("offline provider received new-work feed: %v", err)
	}
	lat, lon := -23.5641, -46.6823
	if _, err := repo.CreateSolicitation(ctx, model.SolicitationItem{ClientID: otherClient, ProviderID: provider, ServiceName: "New request", Description: "Do not route while offline", LocationAddress: "Pinheiros SP", Latitude: &lat, Longitude: &lon}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("new request was routed to offline provider: %v", err)
	}
	for _, outsider := range []string{otherClient, otherProvider} {
		if _, err := repo.GetSolicitationForUser(ctx, quote.ID, outsider); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("outsider read: %v", err)
		}
		if _, err := repo.UpdateSolicitationStatus(ctx, quote.ID, model.SolicitationStatusUpdateInput{ActorID: outsider, Status: "COMPLETED"}); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("outsider mutation: %v", err)
		}
	}
	amount := 200.0
	item := commandQuote(t, repo, quote.ID, provider, "BUDGET_RECEIVED", &amount, "initial")
	if item.RenegotiationCount != 0 || item.Status != "BUDGET_RECEIVED" {
		t.Fatalf("initial counted as counter: %+v", item)
	}
	for n := 1; n <= 5; n++ {
		actor := client
		if n%2 == 0 {
			actor = provider
		}
		amount = 200 - float64(n)
		item = commandQuote(t, repo, quote.ID, actor, "NEGOTIATING", &amount, "round-"+string(rune('0'+n)))
		if item.RenegotiationCount != n {
			t.Fatalf("round %d counted %d", n, item.RenegotiationCount)
		}
	}
	amount = 150
	if _, err := repo.UpdateSolicitationStatus(ctx, quote.ID, model.SolicitationStatusUpdateInput{ActorID: provider, Status: "NEGOTIATING", CounterOfferPrice: &amount}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("sixth counter allowed: %v", err)
	}
	spoofed := 99999.0
	item = commandQuote(t, repo, quote.ID, provider, "ACTIVE", &spoofed, "")
	if item.Status != "ACTIVE" || item.Price != 195 {
		t.Fatalf("accepted spoofed price: status=%s price=%v", item.Status, item.Price)
	}
	if _, err := repo.UpdateSolicitationStatus(ctx, quote.ID, model.SolicitationStatusUpdateInput{ActorID: client, Status: "NEGOTIATING", CounterOfferPrice: &amount}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("reopened accepted contract: %v", err)
	}
	reviews := NewReviewRepository(pool)
	review := &domain.Review{OrderID: quote.ID, ReviewerID: client, Rating: 5}
	if err := reviews.CreateReviewInTx(ctx, review); !errors.Is(err, domain.ErrOrderNotCompleted) {
		t.Fatalf("review completed order implicitly: %v", err)
	}
	commandQuote(t, repo, quote.ID, provider, "COMPLETED", nil, "")
	if err := reviews.CreateReviewInTx(ctx, review); err != nil {
		t.Fatal(err)
	}
	if err := reviews.CreateReviewInTx(ctx, review); !errors.Is(err, domain.ErrAlreadyReviewed) {
		t.Fatalf("duplicate review: %v", err)
	}
	if err := repo.SubmitClientEvaluation(ctx, quote.ID, otherProvider, domain.ClientEvaluationRequest{Stars: 4}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign provider rated: %v", err)
	}
	if err := repo.SubmitClientEvaluation(ctx, quote.ID, provider, domain.ClientEvaluationRequest{Stars: 4, TechnicalReport: "Lights fixed"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SubmitClientEvaluation(ctx, quote.ID, provider, domain.ClientEvaluationRequest{Stars: 1}); !errors.Is(err, domain.ErrAlreadyReviewed) {
		t.Fatalf("duplicate client rating: %v", err)
	}
	var finalPrice float64
	var proposalID *string
	if err := pool.QueryRow(ctx, `SELECT final_price,accepted_negotiation_id::text FROM orders WHERE quote_request_id=$1::uuid`, quote.ID).Scan(&finalPrice, &proposalID); err != nil || finalPrice != 195 || proposalID == nil {
		t.Fatalf("agreement not persisted: %v %v %v", finalPrice, proposalID, err)
	}
}
func TestConcurrentRetryAndInformationDoesNotConsumeRound(t *testing.T) {
	pool := contractTestPool(t)
	ctx := context.Background()
	client, provider, _, _ := contractUsers(t, pool)
	quote := contractQuote(t, pool, client, provider)
	repo := NewSolicitationPostgresRepository(pool)
	amount := 200.0
	commandQuote(t, repo, quote.ID, provider, "BUDGET_RECEIVED", &amount, "initial")
	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			price := 180.0
			note := "same offer"
			_, err := repo.UpdateSolicitationStatus(ctx, quote.ID, model.SolicitationStatusUpdateInput{ActorID: client, Status: "NEGOTIATING", CounterOfferPrice: &price, CounterOfferMessage: &note, RequestKey: "repeat-once"})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	note := "Please send another photo"
	item, err := repo.UpdateSolicitationStatus(ctx, quote.ID, model.SolicitationStatusUpdateInput{ActorID: provider, Status: "NEGOTIATING", CounterOfferMessage: &note, RequestKey: "information"})
	if err != nil || item.RenegotiationCount != 1 {
		t.Fatalf("information consumed round: %+v %v", item, err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM quote_negotiations WHERE quote_request_id=$1::uuid AND is_counter_offer`, quote.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("retry duplicated counter: %d %v", count, err)
	}
}
func TestPortfolioOwnershipSoftDeleteAndRollback(t *testing.T) {
	pool := contractTestPool(t)
	ctx := context.Background()
	_, provider, _, outsider := contractUsers(t, pool)
	repo := NewServiceRepository(pool).(domain.ServiceMutationRepository)
	var category string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM categories LIMIT 1`).Scan(&category); err != nil {
		t.Fatal(err)
	}
	s, err := repo.CreateService(ctx, &domain.Service{BikkerID: provider, CategoryID: category, Name: "Panel repair", Latitude: -23.5614, Longitude: -46.6914, IsActive: true, Photos: []string{"https://media.test.invalid/photo.jpg"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE bikkers SET is_active=FALSE WHERE id=$1::uuid`, provider); err != nil {
		t.Fatal(err)
	}
	// Offline professionals may edit and remove portfolio items, but may not
	// publish a new active listing until they reactivate their profile.
	if _, err = repo.CreateService(ctx, &domain.Service{BikkerID: provider, CategoryID: category, Name: "Offline listing", Latitude: -23.5614, Longitude: -46.6914, IsActive: true}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("offline provider published service: %v", err)
	}
	s.Description = "Updated while offline"
	if _, err = repo.UpdateService(ctx, s); err != nil {
		t.Fatalf("offline provider could not manage portfolio: %v", err)
	}
	if err = repo.DeleteService(ctx, s.ID, outsider); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign delete allowed: %v", err)
	}
	var photos int
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM service_photos WHERE service_id=$1::uuid`, s.ID).Scan(&photos); err != nil || photos != 1 {
		t.Fatalf("foreign deletion changed photos: %d %v", photos, err)
	}
	if err = repo.DeleteService(ctx, s.ID, provider); err != nil {
		t.Fatal(err)
	}
	if _, err = NewServiceRepository(pool).GetByID(ctx, s.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted service remains public: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM service_photos WHERE service_id=$1::uuid`, s.ID).Scan(&photos); err != nil || photos != 1 {
		t.Fatalf("history photos lost: %d %v", photos, err)
	}
}

func TestCommandRollsBackWhenOutboxCannotBePersisted(t *testing.T) {
	pool := contractTestPool(t)
	ctx := context.Background()
	client, provider, _, _ := contractUsers(t, pool)
	quote := contractQuote(t, pool, client, provider)
	repo := NewSolicitationPostgresRepository(pool)
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	function := "fail_outbox_" + suffix
	trigger := "fail_outbox_trigger_" + suffix
	sql := `CREATE FUNCTION ` + function + `() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.data->>'solicitation_id' = '` + quote.ID + `' THEN RAISE EXCEPTION 'injected outbox failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER ` + trigger + ` BEFORE INSERT ON notification_outbox FOR EACH ROW EXECUTE FUNCTION ` + function + `() `
	if _, err := pool.Exec(ctx, sql); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DROP TRIGGER `+trigger+` ON notification_outbox; DROP FUNCTION `+function+`() `); err != nil {
			t.Error(err)
		}
	})
	price := 200.0
	if _, err := repo.UpdateSolicitationStatus(ctx, quote.ID, model.SolicitationStatusUpdateInput{ActorID: provider, Status: "BUDGET_RECEIVED", CounterOfferPrice: &price}); err == nil {
		t.Fatal("injected persistence failure returned success")
	}
	var status string
	var count int
	if err := pool.QueryRow(ctx, `SELECT status::text,(SELECT COUNT(*) FROM quote_negotiations WHERE quote_request_id=$1::uuid) FROM quote_requests WHERE id=$1::uuid`, quote.ID).Scan(&status, &count); err != nil || status != "PENDING" || count != 0 {
		t.Fatalf("partial commit: status=%s negotiations=%d err=%v", status, count, err)
	}
}

func TestCreateIdempotencyKeyRejectsDifferentPayload(t *testing.T) {
	pool := contractTestPool(t)
	ctx := context.Background()
	client, provider, _, _ := contractUsers(t, pool)
	repo := NewSolicitationPostgresRepository(pool)
	lat, lon := -23.5641, -46.6823
	req := model.SolicitationItem{ClientID: client, ProviderID: provider, ServiceName: "Repair", Description: "Repair lights", LocationAddress: "Pinheiros", Latitude: &lat, Longitude: &lon, Photos: []string{}, RequestKey: "create-once"}
	first, err := repo.CreateSolicitation(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.CreateSolicitation(ctx, req)
	if err != nil || first.ID != second.ID {
		t.Fatalf("retry duplicated request: %v %v", second, err)
	}
	req.Description = "Different work"
	if _, err = repo.CreateSolicitation(ctx, req); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("reused key changed operation: %v", err)
	}
}
