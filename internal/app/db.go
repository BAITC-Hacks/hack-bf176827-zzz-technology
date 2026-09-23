package app

import (
	"context"
	"fmt"

	"hackaton/internal/config"
	"hackaton/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func ModuleDB() fx.Option {
	return fx.Provide(newPool)
}

func newPool(lc fx.Lifecycle, cfg *config.Config, log *zap.Logger) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), cfg.Postgres.DSN())
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := pool.Ping(ctx); err != nil {
				return fmt.Errorf("ping postgres (%s:%d): %w", cfg.Postgres.Host, cfg.Postgres.Port, err)
			}
			if cfg.Postgres.AutoMigrate {
				return migrations.Up(ctx, pool, log)
			}
			return nil
		},
		OnStop: func(context.Context) error {
			pool.Close()
			return nil
		},
	})

	return pool, nil
}
