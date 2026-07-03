package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // register mysql driver
)

// Config holds MySQL connection pool settings.
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// NewMySQL opens a MySQL connection pool and verifies connectivity.
func NewMySQL(ctx context.Context, cfg Config) (*sql.DB, error) {
	pool, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("opening mysql pool: %w", err)
	}

	pool.SetMaxOpenConns(cfg.MaxOpenConns)
	pool.SetMaxIdleConns(cfg.MaxIdleConns)
	pool.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := pool.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("pinging mysql: %w", err)
	}
	return pool, nil
}
