package server

import (
	"context"
	"fmt"
	"net/http"

	"team-taskflow/internal/clients/email"
	httpdelivery "team-taskflow/internal/delivery/http"
	"team-taskflow/internal/infrastructure/auth"
	"team-taskflow/internal/infrastructure/db"
	redisinfra "team-taskflow/internal/infrastructure/redis"
	"team-taskflow/internal/infrastructure/tx"
	historyrepo "team-taskflow/internal/repository/mysql/history"
	taskrepo "team-taskflow/internal/repository/mysql/task"
	teamrepo "team-taskflow/internal/repository/mysql/team"
	userrepo "team-taskflow/internal/repository/mysql/user"
	"team-taskflow/internal/repository/redis/taskcache"
	"team-taskflow/internal/usecase/auth_login"
	"team-taskflow/internal/usecase/auth_register"
	"team-taskflow/internal/usecase/task_create"
	"team-taskflow/internal/usecase/task_history_get"
	"team-taskflow/internal/usecase/task_list"
	"team-taskflow/internal/usecase/task_update"
	"team-taskflow/internal/usecase/team_create"
	"team-taskflow/internal/usecase/team_invite"
	"team-taskflow/internal/usecase/team_list"
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

	passwordHasher, err := auth.NewPasswordHasher(cfg.Auth.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("creating password hasher: %w", err)
	}
	jwtManager := auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)

	// Driven adapters.
	userRepository := userrepo.NewRepository(pool)
	teamRepository := teamrepo.NewRepository(pool)
	emailClient := email.NewClient(email.Config{
		BaseURL:        cfg.Email.BaseURL,
		RequestTimeout: cfg.Email.RequestTimeout,
		MaxRequests:    cfg.Email.BreakerMaxRequests,
		Interval:       cfg.Email.BreakerInterval,
		Timeout:        cfg.Email.BreakerTimeout,
		MaxFailures:    cfg.Email.BreakerMaxFailures,
	})

	taskRepository := taskrepo.NewRepository(pool)
	historyRepository := historyrepo.NewRepository(pool)
	taskListCache := taskcache.NewCache(redisClient, cfg.Cache.TaskListTTL)

	// Usecases.
	registerUsecase := auth_register.New(userRepository, passwordHasher)
	loginUsecase := auth_login.New(userRepository, passwordHasher, jwtManager)
	teamCreateUsecase := team_create.New(teamRepository, txManager)
	teamListUsecase := team_list.New(teamRepository)
	teamInviteUsecase := team_invite.New(teamRepository, userRepository, emailClient)
	taskCreateUsecase := task_create.New(taskRepository, teamRepository, taskListCache)
	taskListUsecase := task_list.New(taskRepository, teamRepository, taskListCache, task_list.Pagination{
		DefaultPageSize: cfg.Pagination.DefaultPageSize,
		MaxPageSize:     cfg.Pagination.MaxPageSize,
	})
	taskUpdateUsecase := task_update.New(taskRepository, teamRepository, historyRepository, txManager, taskListCache)
	taskHistoryUsecase := task_history_get.New(taskRepository, teamRepository, historyRepository)

	// Delivery.
	router := httpdelivery.NewRouter(httpdelivery.RouterDeps{
		AuthMiddleware: httpdelivery.NewAuthMiddleware(jwtManager),
		Register:       httpdelivery.NewRegisterHandler(registerUsecase).Handle,
		Login:          httpdelivery.NewLoginHandler(loginUsecase).Handle,
		TeamCreate:     httpdelivery.NewTeamCreateHandler(teamCreateUsecase).Handle,
		TeamList:       httpdelivery.NewTeamListHandler(teamListUsecase).Handle,
		TeamInvite:     httpdelivery.NewTeamInviteHandler(teamInviteUsecase).Handle,
		TaskCreate:     httpdelivery.NewTaskCreateHandler(taskCreateUsecase).Handle,
		TaskList:       httpdelivery.NewTaskListHandler(taskListUsecase).Handle,
		TaskUpdate:     httpdelivery.NewTaskUpdateHandler(taskUpdateUsecase).Handle,
		TaskHistory:    httpdelivery.NewTaskHistoryHandler(taskHistoryUsecase).Handle,
	})

	return &dependencies{
		handler: router,
		closers: []func() error{
			redisClient.Close,
			pool.Close,
		},
	}, nil
}
