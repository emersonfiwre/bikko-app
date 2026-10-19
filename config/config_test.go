package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_PostgresPoolDefaults(t *testing.T) {
	// Ensure no env overrides
	os.Unsetenv("DB_MAX_CONNS")
	os.Unsetenv("DB_MIN_CONNS")
	os.Unsetenv("DB_MAX_CONN_LIFETIME")
	os.Unsetenv("DB_MAX_CONN_IDLE_TIME")
	os.Unsetenv("DB_HEALTH_CHECK_PERIOD")
	os.Unsetenv("DB_CONNECT_TIMEOUT")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("DB_SSLMODE")

	cfg := LoadConfig()

	if cfg.DBMaxConns != 25 {
		t.Errorf("expected DBMaxConns 25, got %d", cfg.DBMaxConns)
	}
	if cfg.DBMinConns != 5 {
		t.Errorf("expected DBMinConns 5, got %d", cfg.DBMinConns)
	}
	if cfg.DBMaxConnLifetime != 30*time.Minute {
		t.Errorf("expected DBMaxConnLifetime 30m, got %v", cfg.DBMaxConnLifetime)
	}
	if cfg.DBMaxConnIdleTime != 5*time.Minute {
		t.Errorf("expected DBMaxConnIdleTime 5m, got %v", cfg.DBMaxConnIdleTime)
	}
	if cfg.DBHealthCheckPeriod != 1*time.Minute {
		t.Errorf("expected DBHealthCheckPeriod 1m, got %v", cfg.DBHealthCheckPeriod)
	}
	if cfg.DBConnectTimeout != 5*time.Second {
		t.Errorf("expected DBConnectTimeout 5s, got %v", cfg.DBConnectTimeout)
	}
	expectedDSN := "postgres://bikko:bikkopass@localhost:5432/bikkodb?sslmode=disable"
	if cfg.GetDSN() != expectedDSN {
		t.Errorf("expected DSN %q, got %q", expectedDSN, cfg.GetDSN())
	}
}

func TestLoadConfig_PostgresPoolCustomEnv(t *testing.T) {
	os.Setenv("DB_MAX_CONNS", "50")
	os.Setenv("DB_MIN_CONNS", "10")
	os.Setenv("DB_MAX_CONN_LIFETIME", "1h")
	os.Setenv("DB_MAX_CONN_IDLE_TIME", "10m")
	os.Setenv("DB_HEALTH_CHECK_PERIOD", "30s")
	os.Setenv("DB_CONNECT_TIMEOUT", "8s")
	os.Setenv("DATABASE_URL", "postgres://custom:pass@cloud-db.internal:5432/proddb?sslmode=require")
	defer func() {
		os.Unsetenv("DB_MAX_CONNS")
		os.Unsetenv("DB_MIN_CONNS")
		os.Unsetenv("DB_MAX_CONN_LIFETIME")
		os.Unsetenv("DB_MAX_CONN_IDLE_TIME")
		os.Unsetenv("DB_HEALTH_CHECK_PERIOD")
		os.Unsetenv("DB_CONNECT_TIMEOUT")
		os.Unsetenv("DATABASE_URL")
	}()

	cfg := LoadConfig()

	if cfg.DBMaxConns != 50 {
		t.Errorf("expected DBMaxConns 50, got %d", cfg.DBMaxConns)
	}
	if cfg.DBMinConns != 10 {
		t.Errorf("expected DBMinConns 10, got %d", cfg.DBMinConns)
	}
	if cfg.DBMaxConnLifetime != time.Hour {
		t.Errorf("expected DBMaxConnLifetime 1h, got %v", cfg.DBMaxConnLifetime)
	}
	if cfg.DBMaxConnIdleTime != 10*time.Minute {
		t.Errorf("expected DBMaxConnIdleTime 10m, got %v", cfg.DBMaxConnIdleTime)
	}
	if cfg.DBHealthCheckPeriod != 30*time.Second {
		t.Errorf("expected DBHealthCheckPeriod 30s, got %v", cfg.DBHealthCheckPeriod)
	}
	if cfg.DBConnectTimeout != 8*time.Second {
		t.Errorf("expected DBConnectTimeout 8s, got %v", cfg.DBConnectTimeout)
	}
	expectedDSN := "postgres://custom:pass@cloud-db.internal:5432/proddb?sslmode=require"
	if cfg.GetDSN() != expectedDSN {
		t.Errorf("expected DSN %q, got %q", expectedDSN, cfg.GetDSN())
	}
}
