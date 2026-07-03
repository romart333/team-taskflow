package server

import (
	"context"
	"net/http"

	httpdelivery "team-taskflow/internal/delivery/http"
)

// dependencies holds everything App needs from the composition root.
type dependencies struct {
	handler http.Handler
	closers []func() error
}

// buildDependencies wires the dependency graph: drivers -> adapters -> usecases -> delivery.
func buildDependencies(_ context.Context, _ Config) (*dependencies, error) {
	router := httpdelivery.NewRouter(httpdelivery.RouterDeps{})

	return &dependencies{
		handler: router,
		closers: nil,
	}, nil
}
