package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type outboxTestSender struct {
	fn func(context.Context, string, string, string, map[string]string) error
}

func (s outboxTestSender) SendNotification(ctx context.Context, userID, title, body string, data map[string]string) error {
	return s.fn(ctx, userID, title, body, data)
}

// Each test uses a private schema so tests can run concurrently against the
// shared disposable database without reading or changing other test events.
func outboxTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not configured: outbox integration not executed")
	}
	admin, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "push_outbox_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec(context.Background(), `CREATE SCHEMA `+pgx.Identifier{schema}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		_, _ = admin.Exec(context.Background(), `DROP SCHEMA `+pgx.Identifier{schema}.Sanitize()+` CASCADE`)
		admin.Close()
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `CREATE TABLE users (id UUID PRIMARY KEY)`); err != nil {
		pool.Close()
		_, _ = admin.Exec(context.Background(), `DROP SCHEMA `+pgx.Identifier{schema}.Sanitize()+` CASCADE`)
		admin.Close()
		t.Fatal(err)
	}
	ddl := `CREATE TABLE notification_outbox (
 id UUID PRIMARY KEY,
 user_id UUID NOT NULL REFERENCES users(id),
 title TEXT NOT NULL,
 body TEXT NOT NULL,
 data JSONB NOT NULL DEFAULT '{}',
 attempts INTEGER NOT NULL DEFAULT 0,
 available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 delivered_at TIMESTAMPTZ,
 last_error TEXT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`
	if _, err := pool.Exec(context.Background(), ddl); err != nil {
		pool.Close()
		_, _ = admin.Exec(context.Background(), `DROP SCHEMA `+pgx.Identifier{schema}.Sanitize()+` CASCADE`)
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(context.Background(), `DROP SCHEMA `+pgx.Identifier{schema}.Sanitize()+` CASCADE`); err != nil {
			t.Errorf("drop isolated outbox schema: %v", err)
		}
		admin.Close()
	})
	return pool
}

func insertOutboxFixture(t *testing.T, pool *pgxpool.Pool) (string, string) {
	t.Helper()
	ctx := context.Background()
	userID, eventID := uuid.NewString(), uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, userID); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(map[string]string{"type": "test_event"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO notification_outbox(id,user_id,title,body,data) VALUES($1,$2,'Title','Body',$3::jsonb)`, eventID, userID, string(data)); err != nil {
		t.Fatal(err)
	}
	return userID, eventID
}

func TestOutboxDeliverOneMarksSuccessfulDelivery(t *testing.T) {
	pool := outboxTestPool(t)
	userID, eventID := insertOutboxFixture(t, pool)
	var calls atomic.Int32
	sender := outboxTestSender{fn: func(_ context.Context, user, title, body string, data map[string]string) error {
		calls.Add(1)
		if user != userID || title != "Title" || body != "Body" || data["type"] != "test_event" || data["notification_id"] != eventID {
			t.Errorf("unexpected notification: user=%q title=%q body=%q data=%v", user, title, body, data)
		}
		return nil
	}}
	found, err := deliverOne(context.Background(), pool, sender)
	if err != nil || !found {
		t.Fatalf("deliverOne=(%v,%v), want (true,nil)", found, err)
	}
	var delivered *time.Time
	var attempts int
	var lastError *string
	if err := pool.QueryRow(context.Background(), `SELECT delivered_at,attempts,last_error FROM notification_outbox WHERE id=$1`, eventID).Scan(&delivered, &attempts, &lastError); err != nil {
		t.Fatal(err)
	}
	if delivered == nil || attempts != 1 || lastError != nil || calls.Load() != 1 {
		t.Fatalf("delivery state: delivered=%v attempts=%d last_error=%v calls=%d", delivered, attempts, lastError, calls.Load())
	}
}

func TestOutboxFailureKeepsEventPendingWithBackoffAndNoProviderSecret(t *testing.T) {
	pool := outboxTestPool(t)
	userID, eventID := insertOutboxFixture(t, pool)
	if _, err := pool.Exec(context.Background(), `UPDATE notification_outbox SET attempts=5 WHERE id=$1`, eventID); err != nil {
		t.Fatal(err)
	}
	secret := "FCM registration token: sensitive-token-value"
	var started time.Time
	if err := pool.QueryRow(context.Background(), `SELECT clock_timestamp()`).Scan(&started); err != nil {
		t.Fatal(err)
	}
	found, err := deliverOne(context.Background(), pool, outboxTestSender{fn: func(context.Context, string, string, string, map[string]string) error {
		return errors.New(secret)
	}})
	if err != nil || !found {
		t.Fatalf("deliverOne=(%v,%v), want (true,nil) after recording retry", found, err)
	}
	var delivered *time.Time
	var available time.Time
	var attempts int
	var lastError *string
	var storedData []byte
	if err := pool.QueryRow(context.Background(), `SELECT delivered_at,available_at,attempts,last_error,data FROM notification_outbox WHERE id=$1 AND user_id=$2`, eventID, userID).Scan(&delivered, &available, &attempts, &lastError, &storedData); err != nil {
		t.Fatal(err)
	}
	if delivered != nil || attempts != 6 || lastError == nil || *lastError != "delivery_failed" || strings.Contains(*lastError, secret) || strings.Contains(string(storedData), secret) {
		t.Fatalf("unsafe retry state: delivered=%v attempts=%d last_error=%v data=%s", delivered, attempts, lastError, storedData)
	}
	backoff := available.Sub(started)
	if backoff < 30*time.Second || backoff > 60*time.Second {
		t.Fatalf("sixth retry should schedule about 32 seconds from database start, backoff=%s", backoff)
	}
	found, err = deliverOne(context.Background(), pool, outboxTestSender{fn: func(context.Context, string, string, string, map[string]string) error {
		t.Fatal("backoff event should not be delivered again yet")
		return nil
	}})
	if err != nil || found {
		t.Fatalf("backoff poll=(%v,%v), want (false,nil)", found, err)
	}
}

func TestOutboxCancellationAfterSendRollsBackForAtLeastOnceRetry(t *testing.T) {
	pool := outboxTestPool(t)
	_, eventID := insertOutboxFixture(t, pool)
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	found, err := deliverOne(ctx, pool, outboxTestSender{fn: func(context.Context, string, string, string, map[string]string) error {
		calls.Add(1)
		cancel() // Simulate cancellation immediately after the external send succeeds.
		return nil
	}})
	if err == nil || !found {
		t.Fatalf("cancelled delivery=(%v,%v), want (true,error) while persisting marker", found, err)
	}
	var delivered *time.Time
	var attempts int
	if err := pool.QueryRow(context.Background(), `SELECT delivered_at,attempts FROM notification_outbox WHERE id=$1`, eventID).Scan(&delivered, &attempts); err != nil {
		t.Fatal(err)
	}
	if delivered != nil || attempts != 0 || calls.Load() != 1 {
		t.Fatalf("cancelled send should leave event retryable: delivered=%v attempts=%d sends=%d", delivered, attempts, calls.Load())
	}
	found, err = deliverOne(context.Background(), pool, outboxTestSender{fn: func(context.Context, string, string, string, map[string]string) error {
		calls.Add(1)
		return nil
	}})
	if err != nil || !found {
		t.Fatalf("retry delivery=(%v,%v), want (true,nil)", found, err)
	}
	if calls.Load() != 2 {
		t.Fatalf("sender called %d times, want duplicate external send to demonstrate at-least-once", calls.Load())
	}
	if err := pool.QueryRow(context.Background(), `SELECT delivered_at,attempts FROM notification_outbox WHERE id=$1`, eventID).Scan(&delivered, &attempts); err != nil {
		t.Fatal(err)
	}
	if delivered == nil || attempts != 1 {
		t.Fatalf("retried event not marked delivered: delivered=%v attempts=%d", delivered, attempts)
	}
}

func TestConcurrentOutboxWorkersDoNotSendSameRow(t *testing.T) {
	pool := outboxTestPool(t)
	_, eventID := insertOutboxFixture(t, pool)
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	sender := outboxTestSender{fn: func(context.Context, string, string, string, map[string]string) error {
		if calls.Add(1) == 1 {
			close(entered)
			<-release
		}
		return nil
	}}
	type result struct {
		found bool
		err   error
	}
	first := make(chan result, 1)
	go func() {
		found, err := deliverOne(context.Background(), pool, sender)
		first <- result{found, err}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first worker did not acquire and send event")
	}
	secondFound, secondErr := deliverOne(context.Background(), pool, sender)
	if secondErr != nil || secondFound {
		t.Fatalf("second worker=(%v,%v), want (false,nil) while first owns row lock", secondFound, secondErr)
	}
	close(release)
	select {
	case got := <-first:
		if got.err != nil || !got.found {
			t.Fatalf("first worker=(%v,%v), want (true,nil)", got.found, got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first worker did not finish")
	}
	if calls.Load() != 1 {
		t.Fatalf("sender called %d times, want 1", calls.Load())
	}
	var delivered *time.Time
	if err := pool.QueryRow(context.Background(), `SELECT delivered_at FROM notification_outbox WHERE id=$1`, eventID).Scan(&delivered); err != nil || delivered == nil {
		t.Fatalf("event not marked delivered: %v %v", delivered, err)
	}
}

func TestOutboxUpdateFailureRollsBackWithoutLosingEvent(t *testing.T) {
	pool := outboxTestPool(t)
	_, eventID := insertOutboxFixture(t, pool)
	functionName := "reject_delivery_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	triggerName := "reject_delivery_trigger_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	functionDDL := fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.id='%s'::uuid THEN RAISE EXCEPTION 'injected delivery marker failure'; END IF; RETURN NEW; END $$`, pgx.Identifier{functionName}.Sanitize(), eventID)
	if _, err := pool.Exec(context.Background(), functionDDL); err != nil {
		t.Fatal(err)
	}
	triggerDDL := `CREATE TRIGGER ` + pgx.Identifier{triggerName}.Sanitize() + ` BEFORE UPDATE ON notification_outbox FOR EACH ROW EXECUTE FUNCTION ` + pgx.Identifier{functionName}.Sanitize() + `()`
	if _, err := pool.Exec(context.Background(), triggerDDL); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS `+pgx.Identifier{triggerName}.Sanitize()+` ON notification_outbox`); err != nil {
			t.Errorf("drop private trigger: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `DROP FUNCTION IF EXISTS `+pgx.Identifier{functionName}.Sanitize()+`() `); err != nil {
			t.Errorf("drop private trigger function: %v", err)
		}
	})
	var calls atomic.Int32
	found, err := deliverOne(context.Background(), pool, outboxTestSender{fn: func(context.Context, string, string, string, map[string]string) error {
		calls.Add(1)
		return nil
	}})
	if err == nil || !found {
		t.Fatalf("deliverOne=(%v,%v), want (true,error) on update failure", found, err)
	}
	var delivered *time.Time
	var attempts int
	var lastError *string
	if err := pool.QueryRow(context.Background(), `SELECT delivered_at,attempts,last_error FROM notification_outbox WHERE id=$1`, eventID).Scan(&delivered, &attempts, &lastError); err != nil {
		t.Fatal(err)
	}
	if delivered != nil || attempts != 0 || lastError != nil || calls.Load() != 1 {
		t.Fatalf("failed transaction lost or changed event: delivered=%v attempts=%d last_error=%v calls=%d", delivered, attempts, lastError, calls.Load())
	}
	if _, err := pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS `+pgx.Identifier{triggerName}.Sanitize()+` ON notification_outbox`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `DROP FUNCTION IF EXISTS `+pgx.Identifier{functionName}.Sanitize()+`() `); err != nil {
		t.Fatal(err)
	}
}
