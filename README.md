# IT-Assist

A small IT help-desk application deployed as a multi-tier system with Docker.
Employees report IT problems as tickets; IT support staff accept, answer and close them.

## Architecture

```
  Browser
     │ http://localhost:8000
     ▼
┌──────────────── Docker network: it-assist_default ────────────────┐
│                                                                   │
│   web (nginx + Vue) ── /api ──▶ api (Go) ──▶ db (PostgreSQL 17)   │
│                                    │                              │
│                                    └──────▶ storage (SeaweedFS)   │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
   volumes: pgdata (database files), objects (uploaded images)
```

| Container | Image | Role | Reachable from the host? |
|---|---|---|---|
| web | built from `frontend/Dockerfile` (nginx) | serves the app, forwards `/api` to the API | yes, port 8000 |
| api | built from `backend/Dockerfile` (Go) | login, ticket rules, notifications, file uploads | no |
| db | `postgres:17` | accounts, sessions, tickets, messages, notifications | no |
| storage | `chrislusf/seaweedfs` | uploaded ticket images (S3-compatible) | no |

Containers find each other by service name (`api`, `db`, `storage`) through Docker's internal DNS.
Data lives in two named volumes (`pgdata`, `objects`), so it survives container restarts.

## Requirements

- Docker Engine 24+ (or Docker Desktop) with Docker Compose v2 (`docker compose version`)
- Free host port 8000 (change `WEB_PORT` in `.env` if needed)

Nothing else: Go, Node.js and PostgreSQL are only used inside the containers.

## Install, build and run

```bash
# 1. Configuration
cp .env.example .env          # then change the passwords in .env

# 2. Build the two images (web, api) and start all four containers
docker compose up -d --build

# 3. Check that everything is running (db should become "healthy")
docker compose ps
```

Open **http://localhost:8000** and log in with one of the demo accounts.

## Demo accounts

Created by `backend/migrations/001_init.sql` when the database starts for the first time.

| Username | Password | Role |
|---|---|---|
| ivan | itassist-ivan | IT support |
| ines | itassist-ines | IT support |
| emma | itassist-emma | Employee |
| noah | itassist-noah | Employee |

These are demo passwords. Change them in `001_init.sql` (then `docker compose down -v`) before any real use.

## Trying it out

Use two separate logins side by side: a normal and a private window, or `http://localhost:8000`
and `http://127.0.0.1:8000` (each has its own login cookie).

1. Log in as **emma** and create a ticket (optionally with an image).
2. Log in as **ivan** in the other window: the bell shows **1**. Click it, then the notification:
   the ticket opens. Accept it.
3. Emma's bell now shows that ivan accepted it. Keep the ticket open in both windows and reply
   back and forth: new messages appear within a few seconds without reloading.
4. As ivan, close the ticket with a resolve type; emma is notified and sees how it was resolved.
5. Each IT member can hold at most **3 ongoing tickets**: accept three as ivan and the fourth is refused,
   while **ines** can still accept.

## Everyday commands

| Task | Command |
|---|---|
| Follow the API's log | `docker compose logs -f api` |
| Stop (keep data) | `docker compose down` |
| Start again | `docker compose up -d` |
| Rebuild after code changes | `docker compose up -d --build` |
| Delete all data and start fresh | `docker compose down -v` |
| Open a database shell | `docker compose exec db psql -U itassist -d itassist` |
| See resource use per container | `docker stats` |

## Configuration (`.env`)

| Variable | Purpose |
|---|---|
| `DB_PASSWORD` | PostgreSQL password (letters and digits; it is placed inside a URL) |
| `S3_ACCESS_KEY`, `S3_SECRET_KEY` | credentials for the storage container, shared with the API |
| `WEB_PORT` | host port for the app (default 8000) |

The database password is applied only when the `pgdata` volume is first created.
After changing it, run `docker compose down -v` (this deletes the data).

## Project structure

```
it-assist/
├── docker-compose.yml        # the four services, network and volumes
├── docker-compose.dev.yml    # development: publishes the db and storage ports
├── .env.example              # configuration template
├── API.md                    # every API endpoint, the ticket rules and notifications
├── frontend/                 # Vue 3 app
│   ├── Dockerfile            # build with Node, serve with nginx
│   ├── nginx.conf            # static files + /api reverse proxy
│   └── src/
└── backend/                  # Go API
    ├── Dockerfile            # build with Go, run on distroless
    ├── migrations/001_init.sql   # schema + demo accounts
    ├── cmd/api/main.go
    └── internal/             # api (HTTP), store (SQL), storage (files), config
```

## Development without rebuilding images

See `backend/README.md` and `frontend/README.md`: start only `db` and `storage` with the dev
override, then run the API with `go run` and the frontend with `npm run dev`.

## Troubleshooting

| Problem | Cause and fix |
|---|---|
| `port is already allocated` | Another program or Compose project uses the port. `docker ps` shows it; stop it, or change `WEB_PORT` in `.env`. |
| Page says "IT-Assist is not available" | The API isn't answering yet or has stopped. `docker compose logs api` shows why. The page retries every 5 seconds. |
| API log: `waiting for database` / `waiting for storage` | Normal during the first seconds; it retries for up to a minute. |
| `password authentication failed` | `DB_PASSWORD` changed after the volume was created. `docker compose down -v`, then up again. |
| Demo accounts don't work | The database was created by an older version of the project. `docker compose down -v`, then up again. |

## Security notes

Passwords are stored as bcrypt hashes. Logging in creates a random session token, sent as an
HttpOnly, SameSite cookie; the database keeps only its SHA-256 hash. Login attempts are
rate-limited. Only the web container is published, and only on `127.0.0.1`. For a real
deployment: HTTPS in front of `web`, `COOKIE_SECURE=1`, real passwords, and backups of both volumes.
