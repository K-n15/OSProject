# IT-Assist API

All requests go through nginx at `/api/...`. Bodies are JSON unless noted.
Errors are always `{"error": "message"}` with a matching HTTP status.

## Session

Logging in sets a cookie, `itassist_session` (random 32 bytes, HttpOnly, SameSite=Lax),
valid for 7 days. The browser sends it automatically; the database stores only its SHA-256 hash.
Any endpoint below except login/logout returns **401** without a valid session.

| Method | Path | Body | Success | Errors |
|---|---|---|---|---|
| POST | `/api/login` | `{username, password}` | `200` user + cookie | `401` wrong username or password, `429` too many attempts |
| POST | `/api/logout` | – | `204`, cookie cleared | – |
| GET | `/api/me` | – | `200` user | `401` |

User object: `{"id": 3, "username": "emma", "name": "Emma Employee", "role": "employee", "createdAt": "..."}`.
`role` is `employee` or `support`.

## Lookups

| Method | Path | Returns |
|---|---|---|
| GET | `/api/devices` | `[{id, name}]` for the New ticket form |
| GET | `/api/resolve-types` | `[{id, name}]` for closing a ticket |

## Tickets

| Method | Path | Who | Body | Success |
|---|---|---|---|---|
| GET | `/api/tickets?status=&q=&page=&pageSize=` | all | – | `{items, total, page, pageSize}` |
| POST | `/api/tickets` | employee | multipart: `title`, `device_id`, `description`, optional `image` | `201` ticket |
| GET | `/api/tickets/{id}` | all | – | ticket with `messages` and `attachments` |
| GET | `/api/tickets/{id}/attachments/{aid}` | all | – | the image file |
| POST | `/api/tickets/{id}/messages` | all | `{body}` | `201` |
| POST | `/api/tickets/{id}/accept` | IT | – | `204` |
| POST | `/api/tickets/{id}/close` | IT | `{resolveTypeId, note}` | `204` |

Rules enforced by the API (the database transactions are in `backend/internal/store/store.go`):

- Employees see only their own tickets; another employee's ticket answers `404`.
- IT sees all pending and closed tickets, and only their **own** ongoing tickets.
- Order: pending and ongoing oldest first, closed by most recently closed, employees newest first.
- An IT member can hold at most **3 ongoing tickets**; accepting a 4th answers `409`.
  Two simultaneous accepts can't exceed the limit (per-member advisory lock).
- A ticket can be accepted only while pending (`409` if someone else got it first).
- Employees can reply until the ticket is closed; IT can reply only on tickets they accepted.
- Closing requires a resolve type.
- Images: PNG, JPEG, GIF or WebP (checked from the file's bytes), at most 2 MB.

## Notifications

Created by the API in the same database transaction as the ticket change they describe.

| Event | Who is notified | `kind` |
|---|---|---|
| Employee creates a ticket | every IT member | `new_ticket` |
| IT accepts a ticket | the employee | `accepted` |
| IT replies | the employee | `reply` |
| Employee replies | the IT member handling the ticket (nobody while pending) | `reply` |
| IT closes a ticket | the employee | `closed` |

| Method | Path | Body | Success |
|---|---|---|---|
| GET | `/api/notifications` | – | `{items: [...], unread: 3}`, newest 20 |
| POST | `/api/notifications/read` | `{ticketId: 5}` for one ticket, `{}` for all | `204` |

Item: `{"id": 7, "ticketId": 5, "ticketTitle": "Printer offline", "ticketStatus": "ongoing",
"kind": "reply", "actorName": "Ivan (IT)", "createdAt": "...", "read": false}`.
Opening a ticket marks its notifications read. Notifications older than 30 days are deleted hourly.

## Keeping screens up to date

The frontend polls: the open ticket every 5 seconds, ticket lists and the bell every 15 seconds,
and only while the browser tab is visible. No extra server connection is needed.
