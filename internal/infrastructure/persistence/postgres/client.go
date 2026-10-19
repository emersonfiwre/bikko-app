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

	maxConns := int32(cfg.DBMaxConns)
	if maxConns <= 0 {
		maxConns = 25
	}
	minConns := int32(cfg.DBMinConns)
	if minConns < 0 {
		minConns = 5
	}
	if minConns > maxConns {
		minConns = maxConns
	}

	poolConfig.MaxConns = maxConns
	poolConfig.MinConns = minConns
	poolConfig.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.DBMaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.DBHealthCheckPeriod
	poolConfig.ConnConfig.ConnectTimeout = cfg.DBConnectTimeout

	slog.Info("Configuração do pool de conexões do PostgreSQL inicializada",
		slog.Int("max_conns", int(poolConfig.MaxConns)),
		slog.Int("min_conns", int(poolConfig.MinConns)),
		slog.Duration("max_conn_lifetime", poolConfig.MaxConnLifetime),
		slog.Duration("max_conn_idle_time", poolConfig.MaxConnIdleTime),
		slog.Duration("health_check_period", poolConfig.HealthCheckPeriod),
		slog.Duration("connect_timeout", poolConfig.ConnConfig.ConnectTimeout),
	)

	var pool *pgxpool.Pool
	var pingErr error

	for attempts := 1; attempts <= 10; attempts++ {
		connectTimeout := cfg.DBConnectTimeout
		if connectTimeout <= 0 {
			connectTimeout = 5 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err == nil {
			pingErr = pool.Ping(ctx)
			cancel()
			if pingErr == nil {
				slog.Info("Pool do PostgreSQL (pgx/v5) conectado com sucesso",
					slog.Int("total_conns", int(pool.Stat().TotalConns())),
					slog.Int("idle_conns", int(pool.Stat().IdleConns())),
				)
				return pool, nil
			}
			pool.Close()
		} else {
			cancel()
			pingErr = err
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
