# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project Overview

Ship monitor cloud server — a Go backend that receives telemetry (metrics,
heartbeats) from ship devices and sensors, stores data, and exposes an HTTP API.

## Tech Stack

- Go 1.27
- HTTP API: generated with [ogen](https://github.com/ogen-go/ogen) from `specs/openapi.yml`
- Async messaging: NATS, client code generated from `specs/asyncapi.yml` (asyncapi-codegen)
- PostgreSQL — primary relational store (`github.com/jmoiron/sqlx` + `lib/pq`)
- ClickHouse — metrics/time-series store (`clickhouse-go/v2`)
- Valkey (Redis-compatible) — cache
- Dependency injection: `go.uber.org/fx`
- Logging: `go.uber.org/zap`
- Migrations: `pressly/goose`

## Commands

```bash
# Run the server (dependencies like NATS/Postgres/ClickHouse come from docker-compose.yml)
go run cmd/server/main.go

# Start dependencies
docker compose up -d

# Build
go build ./cmd/server/main.go

# Lint (golangci-lint v2 config in .golangci.yml)
golangci-lint run
```

## Configuration

Configuration is read from environment variables (`.env` is loaded via
godotenv in `internal/v2/config/config.go`):

| Variable         | Purpose                                                                                   |
| ---------------- | ----------------------------------------------------------------------------------------- |
| `PORT`           | HTTP server port                                                                          |
| `SECURITY_KEY`   | Security key                                                                              |
| `DEVEL`          | Set to enable development mode                                                            |
| `DB_DSN`         | Postgres DSN                                                                              |
| `CLICKHOUSE_DSN` | ClickHouse DSN (e.g. `clickhouse://user:password@localhost:9000/metrics?dial_timeout=5s`) |
| `NATS_URL`       | NATS connection URL                                                                       |
| `VALKEY_URL`     | Valkey address                                                                            |

## Project Structure

```
cmd/server/            — application entrypoint
specs/                 — API specifications (OpenAPI + AsyncAPI) and code generation files and configs
internal/v2/
  config/              — typed configuration from env vars
  di/                  — fx providers and app wiring (di.go, external.go, services.go)
  adapters/            — outbound adapters: db (Postgres), clickhousedb, server, subscriber, publisher
  handlers/            — HTTP (handler, security) and queue handlers
  repositories/        — data access per entity
  services/            — business logic per entity
  domain/              — domain types
  gen/                 — generated API code (do not edit by hand)
migrations/
  postgres/            — goose SQL migrations (Postgres)
  clickhouse/ — goose SQL migrations (ClickHouse)
pkg/                   — shared packages (connector, problem)
```

## Migrations

Both databases use goose with timestamp-prefixed filenames
(`YYYYMMDDHHMMSS_description.sql`) and `-- +goose Up` / `-- +goose Down` sections.

- Postgres migrations live in `migrations/*.sql`, embedded via `migrations/migrations.go`.
- ClickHouse migrations live in `migrations/clickhouse/*.sql`, embedded via `migrations/clickhouse/clickhouse.go`.
- Migrations run automatically on application startup (see `New` in
  `internal/v2/adapters/db/db.go` and `internal/v2/adapters/clickhousedb/clickhouse.go`).
- After adding SQL files to an embedded directory, no registration is needed
  (`embed` picks up `*.sql` automatically), but keep the `.go` embed files intact.
- Always include a `-- +goose Down` section.

## API Changes

- REST API: edit `specs/openapi.yml`, then regenerate with ogen (it is a Go
  tool dependency: `go tool ogen`). Do not edit files under `internal/v2/gen/`.
- AsyncAPI: edit `specs/asyncapi.yml`; generated code lives under
  `pkg/connector/asyncapi/` and `internal/v2/gen/`.

## Code Style

Enforced by `.golangci.yml` (default: **all** linters enabled):

- Max line length: **80 characters** (golines).
- Formatting: gofmt + gofumpt + gci import grouping; run `golangci-lint fmt`.
- Error handling: wrap errors with `fmt.Errorf("...: %w", err)` using
  lowercase, short context messages (wrapcheck is enabled — errors returned
  from other packages must be wrapped).
- Keep dependency direction: handlers → services → repositories. Do not
  import repositories from services or services from handlers (depguard rules).
- Dependency injection goes through fx providers in `internal/v2/di/` —
  register new services/repos/connectors there.
- Do not use `spf13/viper`; configuration is typed in `internal/v2/config`.
- Do not use `io/ioutil` (deprecated).

## Notes for Agents

- `specify the whole request` — when a task touches generated code, change the
  spec first and regenerate instead of editing generated files.
- Postgres and ClickHouse connections are both `*sqlx.DB`; the ClickHouse fx
  provider is annotated `name:"clickhouse"`. Inject it with
  `fx.Annotate`/`fx.ParamTags(`name:"clickhouse"`)` or a wrapper type.
- Keep new features consistent with the existing v2 layout; do not add new
  top-level packages under `internal/`.
- `ship-cloud.sqlite3` and `./.pgdata` are local data — never commit or modify them.
- Spelling matters to the team: `cspell.json` is present; keep technical terms
  (NATS, Valkey, ClickHouse, goose) spelled correctly.
