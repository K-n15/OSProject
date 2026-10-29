# IT-Assist API (Go)

Normally built and started by the root `docker-compose.yml` (see the main README).

## Run it outside Docker (development)

    # from the project root: only the database and storage containers
    docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d db storage

    cd backend
    cp .env.dev.example .env.dev        # values must match the root .env
    set -a; . ./.env.dev; set +a
    go run ./cmd/api                    # listens on :8080

## Layout

| Path | Purpose |
|---|---|
| cmd/api/main.go | startup: config, wait for database and storage, routes, graceful shutdown |
| internal/api/ | HTTP handlers: login/logout (auth.go), tickets, notifications, rate limit, routes (server.go) |
| internal/store/ | all SQL: users.go (login, sessions), store.go (tickets and their rules), notifications.go |
| internal/storage/ | S3-compatible file storage (SeaweedFS in Docker) |
| internal/config/ | settings from environment variables |
| migrations/001_init.sql | schema and demo accounts, applied by the postgres container on first start |
| Dockerfile | two-stage build: compile with golang:1.27-alpine, run on distroless |
