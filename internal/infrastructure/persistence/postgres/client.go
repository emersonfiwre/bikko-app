package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"bikko-app/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dsn := cfg.GetDSN()
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao parsear DSN do Postgres: %w", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnIdleTime = 15 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar pool pgx: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("falha ao pingar Postgres: %w", err)
	}

	log.Println("Pool do PostgreSQL (pgx/v5) conectado com sucesso!")
	return pool, nil
}
