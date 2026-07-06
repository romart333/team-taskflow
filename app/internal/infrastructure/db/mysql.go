package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
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
	dsnCfg, err := mysql.ParseDSN(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parsing mysql dsn: %w", err)
	}
	// Make RowsAffected report matched rows instead of changed rows so
	// repositories can treat 0 on UPDATE as "row not found" even when the
	// update is a no-op.
	dsnCfg.ClientFoundRows = true

	pool, err := sql.Open("mysql", dsnCfg.FormatDSN())
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
