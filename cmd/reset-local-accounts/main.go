package main

import (
	"bikko-app/config"
	"bikko-app/internal/infrastructure/persistence/postgres"
	"bikko-app/migrations"
	"context"
	"flag"
	"log/slog"
	"os"
	"time"
)

func main() {
	confirm := flag.Bool("confirm-reset", false, "clear orders and portfolios belonging to the two reserved local test accounts")
	flag.Parse()
	cfg := config.LoadConfig()
	if !*confirm || cfg.Env != "development" || !cfg.LocalTestAccounts {
		slog.Error("reset requires --confirm-reset, ENV=development and LOCAL_TEST_ACCOUNTS=true")
		os.Exit(1)
	}
	pool, err := postgres.NewPostgresPool(cfg)
	if err != nil {
		slog.Error("database unavailable", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = migrations.ResetLocalAccounts(ctx, pool); err != nil {
		slog.Error("reset failed", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("two reserved test accounts reset to empty baseline")
}
