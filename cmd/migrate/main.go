package main

import (
	"bikko-app/config"
	"bikko-app/internal/infrastructure/persistence/postgres"
	"bikko-app/migrations"
	"context"
	"log/slog"
	"os"
	"time"
)

func main() {
	cfg := config.LoadConfig()
	pool, err := postgres.NewPostgresPool(cfg)
	if err != nil {
		slog.Error("database unavailable", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err = migrations.Apply(ctx, pool); err != nil {
		slog.Error("migration failed", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("migrations applied")
}
