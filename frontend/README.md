# IT-Assist frontend (Vue 3 + Vite)

Normally built into the `web` container (Dockerfile + nginx.conf) by the root `docker-compose.yml`.

## Development

    npm install
    npm run dev            # http://localhost:5173, forwards /api to http://localhost:8080

Run the Go API with `go run` first (backend/README.md), or preview the screens without it:

    npm run mock           # in-memory stand-in API with the same demo accounts

## Layout

| Path | Purpose |
|---|---|
| src/api.js | every request to the backend |
| src/session.js | who is signed in, lookups |
| src/router.js | screens and who may open them |
| src/views/auth/Login.vue | login screen |
| src/views/ | tickets (employee), pending/ongoing/closed queues (IT), solutions, FAQ |
| src/components/TicketDetail.vue | ticket thread (refreshes every 5 s), reply, accept, close |
| src/components/NotificationBell.vue | bell with unread count and dropdown of ticket updates |
| nginx.conf | serves the built app and forwards /api to the api container |
| Dockerfile | two-stage build: npm build on node:22-alpine, serve with nginx:1.27-alpine |
| dev/mock-api.mjs | in-memory stand-in API for previewing (never deploy) |
