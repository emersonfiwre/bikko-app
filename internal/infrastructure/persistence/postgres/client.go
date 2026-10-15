package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"bikko-app/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := cfg.GetDSN()
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao parsear DSN do Postgres: %w", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnIdleTime = 15 * time.Minute

	var pool *pgxpool.Pool
	var pingErr error

	for attempts := 1; attempts <= 10; attempts++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err == nil {
			pingErr = pool.Ping(ctx)
			cancel()
			if pingErr == nil {
				slog.Info("Pool do PostgreSQL (pgx/v5) conectado com sucesso")
				return pool, nil
			}
			pool.Close()
		} else {
			cancel()
		}

		slog.Warn("tentativa de conexão com o PostgreSQL falhou, tentando novamente",
			slog.Int("attempt", attempts),
			slog.Int("max_attempts", 10),
			slog.Any("error", pingErr),
		)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("falha ao conectar e pingar Postgres após várias tentativas: %w", pingErr)
}
