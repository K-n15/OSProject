-- pgcrypto provides crypt()/gen_salt(), used below to bcrypt-hash the demo passwords.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------- accounts ----------
CREATE TABLE users (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  username      TEXT NOT NULL CHECK (username ~ '^[A-Za-z0-9._-]{3,40}$'),
  name          TEXT NOT NULL,
  password_hash TEXT NOT NULL,                -- bcrypt, never the password itself
  role          TEXT NOT NULL CHECK (role IN ('employee','support')),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Usernames are unique regardless of case ("Emma" and "emma" are the same person).
CREATE UNIQUE INDEX users_username_key ON users (lower(username));

-- One row per signed-in browser. Only a SHA-256 hash of the cookie value is
-- stored, so a leaked database backup cannot be used to sign in.
CREATE TABLE sessions (
  token_hash TEXT PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX ON sessions (user_id);

-- ---------- tickets ----------
CREATE TABLE devices       (id SMALLINT PRIMARY KEY, name TEXT UNIQUE NOT NULL);
CREATE TABLE resolve_types (id SMALLINT PRIMARY KEY, name TEXT UNIQUE NOT NULL);

CREATE TABLE tickets (
  id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  requester_id    BIGINT NOT NULL REFERENCES users(id),
  assignee_id     BIGINT REFERENCES users(id),
  device_id       SMALLINT NOT NULL REFERENCES devices(id),
  title           TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 80),
  description     TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','ongoing','closed')),
  resolve_type_id SMALLINT REFERENCES resolve_types(id),
  resolve_note    TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  closed_at       TIMESTAMPTZ,
  CHECK (status <> 'closed' OR (resolve_type_id IS NOT NULL AND closed_at IS NOT NULL)),
  CHECK (status = 'pending' OR assignee_id IS NOT NULL)
);

CREATE TABLE messages (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  ticket_id  BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  author_id  BIGINT NOT NULL REFERENCES users(id),
  body       TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE attachments (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  ticket_id   BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  message_id  BIGINT REFERENCES messages(id),
  object_key  TEXT NOT NULL UNIQUE,
  mime_type   TEXT NOT NULL,
  size_bytes  INT  NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- What to tell whom: one row per recipient. read_at stays empty until it is read.
CREATE TABLE notifications (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  ticket_id  BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  kind       TEXT NOT NULL CHECK (kind IN ('new_ticket','accepted','reply','closed')),
  actor_id   BIGINT REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  read_at    TIMESTAMPTZ
);
CREATE INDEX ON notifications (user_id, created_at DESC);
CREATE INDEX ON notifications (user_id) WHERE read_at IS NULL;

CREATE INDEX ON tickets (status, created_at);
CREATE INDEX ON tickets (requester_id, created_at DESC);
CREATE INDEX ON tickets (assignee_id) WHERE status = 'ongoing';
CREATE INDEX ON messages (ticket_id, created_at);

-- ---------- reference data ----------
INSERT INTO devices (id, name) VALUES
  (1,'Laptop'),(2,'Desktop'),(3,'Printer'),(4,'Phone or tablet'),(5,'Network or Wi-Fi'),(6,'Other');
INSERT INTO resolve_types (id, name) VALUES
  (1,'Fixed'),(2,'Workaround provided'),(3,'Guidance given'),(4,'Cannot reproduce'),(5,'Duplicate'),(6,'Escalated to vendor');

-- ---------- demo accounts ----------
-- The application has no sign-up screen: accounts are created here.
INSERT INTO users (username, name, role, password_hash) VALUES
  ('ivan', 'Ivan (IT)',       'support',  crypt('itassist-ivan', gen_salt('bf', 10))),
  ('ines', 'Ines (IT)',       'support',  crypt('itassist-ines', gen_salt('bf', 10))),
  ('emma', 'Emma Employee',   'employee', crypt('itassist-emma', gen_salt('bf', 10))),
  ('noah', 'Noah Employee',   'employee', crypt('itassist-noah', gen_salt('bf', 10)));
