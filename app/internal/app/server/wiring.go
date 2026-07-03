package server

import (
	"context"
	"fmt"
	"net/http"

	httpdelivery "team-taskflow/internal/delivery/http"
	"team-taskflow/internal/infrastructure/db"
	redisinfra "team-taskflow/internal/infrastructure/redis"
	"team-taskflow/internal/infrastructure/tx"
)

// dependencies holds everything App needs from the composition root.
type dependencies struct {
	handler http.Handler
	closers []func() error
}

// buildDependencies wires the dependency graph: drivers -> adapters -> usecases -> delivery.
func buildDependencies(ctx context.Context, cfg Config) (*dependencies, error) {
	// Drivers.
	pool, err := db.NewMySQL(ctx, db.Config{
		DSN:             cfg.MySQL.DSN(),
		MaxOpenConns:    cfg.MySQL.MaxOpenConns,
		MaxIdleConns:    cfg.MySQL.MaxIdleConns,
		ConnMaxLifetime: cfg.MySQL.ConnMaxLifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to mysql: %w", err)
	}

	if err := db.Migrate(pool, cfg.MySQL.Database); err != nil {
		return nil, fmt.Errorf("migrating database: %w", err)
	}

	redisClient, err := redisinfra.NewClient(ctx, redisinfra.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	txManager := tx.NewManager(pool)
	_ = txManager // wired into usecases in later batches

	// Delivery.
	router := httpdelivery.NewRouter(httpdelivery.RouterDeps{})

	return &dependencies{
		handler: router,
		closers: []func() error{
			redisClient.Close,
			pool.Close,
		},
	}, nil
}
