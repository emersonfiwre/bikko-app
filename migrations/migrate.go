// Package migrations embeds and applies the versioned schema before the API starts.
package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.up.sql
var schema embed.FS

// Apply serializes migrators with a transaction-scoped advisory lock. Existing local
// installations created by docker-entrypoint receive an explicit baseline adoption.
// No sample users, orders or portfolios are ever loaded by migrations.
func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("migrations begin: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(627459031)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS bikko_schema_migrations(name TEXT PRIMARY KEY, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		return err
	}
	entries, err := schema.ReadDir(".")
	if err != nil {
		return err
	}
	names := []string{}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		data, err := schema.ReadFile(name)
		if err != nil {
			return err
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(data))
		var applied string
		err = tx.QueryRow(ctx, `SELECT checksum FROM bikko_schema_migrations WHERE name=$1`, name).Scan(&applied)
		if err == nil {
			if checksum != applied {
				return fmt.Errorf("migration %s changed after application", name)
			}
			continue
		}
		if err != pgx.ErrNoRows {
			return err
		}
		baseline := false
		switch name {
		case "000001_init_schema.up.sql":
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relname='users' AND n.nspname=current_schema())`).Scan(&baseline); err != nil {
				return err
			}
		case "000007_remove_password_hash.up.sql":
			// Some installations executed the former 000004 migration before this
			// ledger existed. Adopt the renumbered migration when the column is
			// already absent, preserving the immutable checksum for databases that
			// have applied 000007 through this runner.
			if err = tx.QueryRow(ctx, `SELECT NOT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = current_schema()
				  AND table_name = 'users'
				  AND column_name = 'password_hash'
			)`).Scan(&baseline); err != nil {
				return err
			}
		}
		if !baseline {
			if _, err = tx.Exec(ctx, string(data)); err != nil {
				return fmt.Errorf("migration %s: %w", name, err)
			}
		}
		if _, err = tx.Exec(ctx, `INSERT INTO bikko_schema_migrations(name,checksum) VALUES($1,$2)`, name, checksum); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("migrations commit: %w", err)
	}
	return nil
}
