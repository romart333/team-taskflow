package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// App owns the composed dependency graph and the HTTP server lifecycle.
type App struct {
	cfg        Config
	httpServer *http.Server
	closers    []func() error
}

// New builds the full dependency graph for the service.
func New(ctx context.Context, cfg Config) (*App, error) {
	deps, err := buildDependencies(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("building dependencies: %w", err)
	}

	httpServer := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      deps.handler,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	return &App{
		cfg:        cfg,
		httpServer: httpServer,
		closers:    deps.closers,
	}, nil
}

// Run starts the HTTP server and blocks until ctx is cancelled or the server fails.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		slog.InfoContext(ctx, "http server starting", "addr", a.cfg.HTTP.Addr)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		a.close(ctx)
		return err
	case <-ctx.Done():
		slog.InfoContext(ctx, "shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		a.close(ctx)
		return fmt.Errorf("shutting down http server: %w", err)
	}
	a.close(ctx)
	slog.InfoContext(ctx, "http server stopped gracefully")
	return nil
}

func (a *App) close(ctx context.Context) {
	for _, closeFn := range a.closers {
		if err := closeFn(); err != nil {
			slog.ErrorContext(ctx, "closing dependency", "error", err)
		}
	}
}
