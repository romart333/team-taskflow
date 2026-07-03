// Package tx provides a transaction manager that carries *sql.Tx through context.
// Usecases control transaction boundaries via Manager.Do; repositories obtain
// the active executor with ExecutorFromContext and never begin/commit themselves.
package tx

import (
	"context"
	"database/sql"
	"fmt"
)

type txContextKey struct{}

// Executor is the query interface common to *sql.DB and *sql.Tx.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Manager implements the TxManager port defined by usecases.
type Manager struct {
	pool *sql.DB
}

func NewManager(pool *sql.DB) *Manager {
	return &Manager{pool: pool}
}

// Do runs fn inside a transaction. A nested Do reuses the transaction already
// present in ctx, so inner calls join the outer transaction.
func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if txFromContext(ctx) != nil {
		return fn(ctx)
	}

	transaction, err := m.pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txContextKey{}, transaction)
	if err := fn(txCtx); err != nil {
		if rbErr := transaction.Rollback(); rbErr != nil {
			return fmt.Errorf("rolling back after %w: %w", err, rbErr)
		}
		return err
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// ExecutorFromContext returns the active transaction from ctx, or pool when
// the call is not transactional.
func ExecutorFromContext(ctx context.Context, pool *sql.DB) Executor {
	if transaction := txFromContext(ctx); transaction != nil {
		return transaction
	}
	return pool
}

func txFromContext(ctx context.Context) *sql.Tx {
	transaction, ok := ctx.Value(txContextKey{}).(*sql.Tx)
	if !ok {
		return nil
	}
	return transaction
}
