# team-taskflow

REST API service for team task management with a role model, audit history and comments.

**Stack:** Go, MySQL, Redis, Docker Compose, JWT, Prometheus.

## Quick start

```bash
# Local dependencies (MySQL + Redis)
make up

# Run the service locally (creates app/configs/config.yaml from the example
# on first run; set auth.jwt_secret there or export AUTH_JWT_SECRET)
make run

# Or the whole stack in Docker
docker compose up -d --build
```

Configuration lives in `app/configs/config.yaml` (not committed; copy `app/configs/config.yaml.example`) with ENV overrides. Config path is set via `CONFIG_PATH`. The JWT secret has no default: startup fails fast unless `auth.jwt_secret` or `AUTH_JWT_SECRET` is set.

## Commands

| Command | Description |
|---|---|
| `make build` | Build the binary into `build/` |
| `make test` | Linter + unit tests |
| `make test-integration` | Integration tests (testcontainers, requires Docker) |
| `make lint` | `go vet` + `golangci-lint` |
| `make up` / `make down` | Start/stop MySQL and Redis |

## API

Base prefix: `/api/v1`. Authentication: `Authorization: Bearer <JWT>`.

| Method | Path | Description |
|---|---|---|
| POST | `/register` | Register a user |
| POST | `/login` | Log in, returns JWT |
| POST | `/teams` | Create a team (creator becomes owner) |
| GET | `/teams` | Teams the user belongs to |
| POST | `/teams/{id}/invite` | Invite a user (owner/admin only) |
| POST | `/tasks` | Create a task (team members only) |
| GET | `/tasks?team_id=&status=&assignee_id=&page=&page_size=` | Filtered list with pagination (Redis cache, 5 min TTL) |
| PUT | `/tasks/{id}` | Update a task (history written atomically) |
| GET | `/tasks/{id}/history` | Task change history |
| POST | `/tasks/{id}/comments` | Add a comment |
| GET | `/tasks/{id}/comments` | Task comments |
| GET | `/analytics/team-stats` | Team stats (multi-JOIN + aggregation) |
| GET | `/analytics/top-creators` | Top-3 task creators per team (window function) |
| GET | `/analytics/integrity/orphaned-assignees` | Audit: assignees that are not team members |
| GET | `/healthz` | Liveness |
| GET | `/metrics` | Prometheus |

## Architecture

Clean Architecture: `domain → usecase (ports, consumer-side interfaces) → delivery/http, repository/mysql, repository/redis, clients/email`. Composition root with manual DI lives in `app/internal/app/server`. Transactions go through a TxManager (context-carried `*sql.Tx`); task history rows are written in the same transaction as the task update. The team task list cache uses versioned Redis keys — invalidation bumps the team's version key instead of scanning for keys.
