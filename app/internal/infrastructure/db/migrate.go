package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"team-taskflow/migrations"
)

// Migrate applies all pending SQL migrations embedded in the migrations package.
func Migrate(ctx context.Context, pool *sql.DB, database string) (err error) {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("loading embedded migrations: %w", err)
	}

	conn, err := pool.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquiring migration connection: %w", err)
	}

	// WithConnection instead of WithInstance: the driver then does not own
	// the pool, so closing the migrator only returns conn to the pool.
	driver, err := migratemysql.WithConnection(ctx, conn, &migratemysql.Config{DatabaseName: database})
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("creating migrate driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, database, driver)
	if err != nil {
		_ = driver.Close()
		return fmt.Errorf("creating migrator: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if closeErr := errors.Join(srcErr, dbErr); closeErr != nil && err == nil {
			err = fmt.Errorf("closing migrator: %w", closeErr)
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("applying migrations: %w", err)
	}
	return nil
}
