package migrations

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed local_test_accounts.sql
var localAccounts string

// SeedLocalAccounts is called only by explicitly enabled development composition.
func SeedLocalAccounts(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, localAccounts); err != nil {
		return fmt.Errorf("local account fixtures: %w", err)
	}
	return tx.Commit(ctx)
}

// ResetLocalAccounts restores only the two reserved development accounts to the
// empty baseline requested for UI tests. It is never called during API startup.
func ResetLocalAccounts(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var ids []string
	rows, err := tx.Query(ctx, `SELECT id::text FROM users WHERE phone IN ('11999990000','11988880000') AND deleted_at IS NULL FOR UPDATE`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if len(ids) != 2 {
		return fmt.Errorf("expected exactly the two reserved local accounts, found %d", len(ids))
	}
	statements := []string{
		`DELETE FROM client_reviews WHERE order_id IN(SELECT id FROM orders WHERE customer_id=ANY($1::uuid[]) OR bikker_id=ANY($1::uuid[]))`,
		`DELETE FROM reviews WHERE order_id IN(SELECT id FROM orders WHERE customer_id=ANY($1::uuid[]) OR bikker_id=ANY($1::uuid[]))`,
		`DELETE FROM orders WHERE customer_id=ANY($1::uuid[]) OR bikker_id=ANY($1::uuid[])`,
		`DELETE FROM quote_negotiations WHERE quote_request_id IN(SELECT id FROM quote_requests WHERE customer_id=ANY($1::uuid[]) OR bikker_id=ANY($1::uuid[]))`,
		`DELETE FROM quote_requests WHERE customer_id=ANY($1::uuid[]) OR bikker_id=ANY($1::uuid[])`,
		`DELETE FROM services WHERE bikker_id=ANY($1::uuid[])`,
		`DELETE FROM notification_outbox WHERE user_id=ANY($1::uuid[])`,
		`DELETE FROM user_devices WHERE user_id=ANY($1::uuid[])`,
		`UPDATE users SET rating=0,total_ratings=0,device_token=NULL WHERE id=ANY($1::uuid[])`,
		`UPDATE bikkers SET rating=0,total_reviews=0,is_active=TRUE WHERE id=ANY($1::uuid[])`,
	}
	for _, statement := range statements {
		if _, err = tx.Exec(ctx, statement, ids); err != nil {
			return fmt.Errorf("reset reserved accounts: %w", err)
		}
	}
	return tx.Commit(ctx)
}
