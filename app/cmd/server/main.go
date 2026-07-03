package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"team-taskflow/internal/app/server"
	"team-taskflow/internal/infrastructure/logger"
)

const (
	configPathEnv     = "CONFIG_PATH"
	defaultConfigPath = "configs/config.yaml"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.ErrorContext(ctx, "service terminated", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	configPath := os.Getenv(configPathEnv)
	if configPath == "" {
		configPath = defaultConfigPath
	}

	cfg, err := server.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := logger.Setup(cfg.Log.Level); err != nil {
		return fmt.Errorf("setting up logger: %w", err)
	}

	app, err := server.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("building app: %w", err)
	}

	if err := app.Run(ctx); err != nil {
		return fmt.Errorf("running app: %w", err)
	}
	return nil
}
