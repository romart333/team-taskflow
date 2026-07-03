//go:build integration

// Package test contains integration tests that exercise the MySQL
// repositories, the transaction manager and the analytics queries against a
// real MySQL instance started via testcontainers.
package test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	mysqlmodule "github.com/testcontainers/testcontainers-go/modules/mysql"

	"team-taskflow/internal/infrastructure/db"
)

var pool *sql.DB

func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		log.Printf("integration test setup failed: %v", err)
		code = 1
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container, err := mysqlmodule.Run(ctx, "mysql:8.4",
		mysqlmodule.WithDatabase("taskflow_test"),
		mysqlmodule.WithUsername("taskflow"),
		mysqlmodule.WithPassword("taskflow"),
	)
	if err != nil {
		return 1, fmt.Errorf("starting mysql container: %w", err)
	}
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			log.Printf("terminating mysql container: %v", err)
		}
	}()

	dsn, err := container.ConnectionString(ctx, "parseTime=true", "multiStatements=true")
	if err != nil {
		return 1, fmt.Errorf("building dsn: %w", err)
	}

	pool, err = db.NewMySQL(ctx, db.Config{
		DSN:             dsn,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		return 1, fmt.Errorf("connecting to mysql: %w", err)
	}
	defer func() { _ = pool.Close() }()

	if err := db.Migrate(pool, "taskflow_test"); err != nil {
		return 1, fmt.Errorf("migrating: %w", err)
	}

	return m.Run(), nil
}
