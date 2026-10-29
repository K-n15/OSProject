package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	NotifyNewTicket = "new_ticket"
	NotifyAccepted  = "accepted"
	NotifyReply     = "reply"
	NotifyClosed    = "closed"
)

type Notification struct {
	ID           int64     `json:"id"`
	TicketID     int64     `json:"ticketId"`
	TicketTitle  string    `json:"ticketTitle"`
	TicketStatus string    `json:"ticketStatus"`
	Kind         string    `json:"kind"`
	ActorName    string    `json:"actorName"`
	CreatedAt    time.Time `json:"createdAt"`
	Read         bool      `json:"read"`
}

type NotificationList struct {
	Items  []Notification `json:"items"`
	Unread int            `json:"unread"`
}

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func notify(ctx context.Context, db execer, userID, ticketID int64, kind string, actorID int64) error {
	_, err := db.Exec(ctx,
		`INSERT INTO notifications (user_id, ticket_id, kind, actor_id) VALUES ($1, $2, $3, $4)`,
		userID, ticketID, kind, actorID)
	return err
}

func notifySupport(ctx context.Context, db execer, ticketID int64, kind string, actorID int64) error {
	_, err := db.Exec(ctx, `
		INSERT INTO notifications (user_id, ticket_id, kind, actor_id)
		SELECT id, $1, $2, $3 FROM users WHERE role = 'support'`, ticketID, kind, actorID)
	return err
}

func (s *Store) ListNotifications(ctx context.Context, userID int64, limit int) (NotificationList, error) {
	res := NotificationList{Items: []Notification{}}
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&res.Unread); err != nil {
		return res, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT n.id, n.ticket_id, t.title, t.status, n.kind, COALESCE(a.name, ''), n.created_at, n.read_at IS NOT NULL
		FROM notifications n
		JOIN tickets t ON t.id = n.ticket_id
		LEFT JOIN users a ON a.id = n.actor_id
		WHERE n.user_id = $1
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.TicketID, &n.TicketTitle, &n.TicketStatus, &n.Kind, &n.ActorName, &n.CreatedAt, &n.Read); err != nil {
			return res, err
		}
		res.Items = append(res.Items, n)
	}
	return res, rows.Err()
}

func (s *Store) MarkNotificationsRead(ctx context.Context, userID, ticketID int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE user_id = $1 AND read_at IS NULL AND ($2 = 0 OR ticket_id = $2)`, userID, ticketID)
	return err
}

func (s *Store) DeleteOldNotifications(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM notifications WHERE created_at < now() - interval '30 days'`)
	return err
}
