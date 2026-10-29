package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

const userColumns = `u.id, u.username, u.name, u.role, u.created_at`

func scanUser(row pgx.Row, extra ...any) (User, error) {
	var u User
	dest := append([]any{&u.ID, &u.Username, &u.Name, &u.Role, &u.CreatedAt}, extra...)
	err := row.Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) UserForLogin(ctx context.Context, username string) (User, string, error) {
	var hash string
	u, err := scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userColumns+`, u.password_hash FROM users u WHERE lower(u.username) = lower($1)`, username), &hash)
	return u, hash, err
}

func (s *Store) CreateSession(ctx context.Context, userID int64, tokenHash string, ttl time.Duration) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, now() + make_interval(secs => $3))`, tokenHash, userID, ttl.Seconds())
	return err
}

func (s *Store) UserBySession(ctx context.Context, tokenHash string) (User, error) {
	return scanUser(s.pool.QueryRow(ctx, `
		SELECT `+userColumns+` FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`, tokenHash))
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	return err
}
