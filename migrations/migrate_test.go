package migrations

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"testing"
)

func TestFreshAndLegacyMigrationsAndEmptyLocalFixtures(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not configured: migration integration not executed")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err = admin.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public; CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public`); err != nil {
		t.Fatal(err)
	}
	for _, legacyState := range []string{"fresh", "legacy_with_password", "legacy_password_removed"} {
		schemaName := "audit_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
			t.Fatal(err)
		}
		defer admin.Exec(ctx, "DROP SCHEMA "+schemaName+" CASCADE")
		cfg, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			t.Fatal(err)
		}
		cfg.ConnConfig.RuntimeParams["search_path"] = schemaName + ",public"
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		// PostGIS is shared in public; migration creates application objects in the isolated schema.
		if legacyState != "fresh" {
			initial, err := schema.ReadFile("000001_init_schema.up.sql")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = pool.Exec(ctx, string(initial)); err != nil {
				t.Fatal(err)
			}
			if legacyState == "legacy_password_removed" {
				if _, err = pool.Exec(ctx, `ALTER TABLE users DROP COLUMN password_hash`); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err = Apply(ctx, pool); err != nil {
			t.Fatal(err)
		}
		if err = Apply(ctx, pool); err != nil {
			t.Fatalf("repeat migration: %v", err)
		}
		if err = SeedLocalAccounts(ctx, pool); err != nil {
			t.Fatal(err)
		}
		if err = SeedLocalAccounts(ctx, pool); err != nil {
			t.Fatalf("repeat fixtures: %v", err)
		}
		var users, quotes, orders, portfolio int
		err = pool.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM users),(SELECT COUNT(*) FROM quote_requests),(SELECT COUNT(*) FROM orders),(SELECT COUNT(*) FROM services)`).Scan(&users, &quotes, &orders, &portfolio)
		if err != nil {
			t.Fatal(err)
		}
		if users != 2 || quotes != 0 || orders != 0 || portfolio != 0 {
			t.Fatalf("legacy_state=%s users=%d quotes=%d orders=%d portfolio=%d", legacyState, users, quotes, orders, portfolio)
		}
		var lat, lon float64
		if err = pool.QueryRow(ctx, `SELECT ST_Y(location_geom::geometry),ST_X(location_geom::geometry) FROM users WHERE phone='11999990000'`).Scan(&lat, &lon); err != nil {
			t.Fatal(err)
		}
		if lat != -23.5641 || lon != -46.6823 {
			t.Fatalf("customer location changed: %v,%v", lat, lon)
		}
		pool.Close()
	}
}
