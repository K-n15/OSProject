package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MaxOngoing = 3

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("you are not allowed to do that")
	ErrConflict  = errors.New("the ticket is not in a state that allows this")
	ErrLimit     = fmt.Errorf("you already have %d ongoing tickets; close one before accepting another", MaxOngoing)
	ErrInvalid   = errors.New("invalid reference in request")
)

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

type Lookup struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Message struct {
	ID         int64     `json:"id"`
	AuthorID   int64     `json:"authorId"`
	AuthorRole string    `json:"from"` //
	Body       string    `json:"text"`
	CreatedAt  time.Time `json:"at"`
}

type Attachment struct {
	ID        int64  `json:"id"`
	ObjectKey string `json:"-"`
	MimeType  string `json:"mimeType"`
	SizeBytes int    `json:"sizeBytes"`
	URL       string `json:"url,omitempty"`
}

type Ticket struct {
	ID          int64        `json:"id"`
	Title       string       `json:"issue"`
	Device      string       `json:"device"`
	Description string       `json:"description"`
	Status      string       `json:"status"`
	RequesterID int64        `json:"requesterId"`
	AssigneeID  *int64       `json:"assigneeId"`
	ResolveType *string      `json:"resolveType"`
	ResolveNote *string      `json:"resolveNote"`
	CreatedAt   time.Time    `json:"reported"`
	UpdatedAt   time.Time    `json:"updated"`
	ClosedAt    *time.Time   `json:"closed"`
	Messages    []Message    `json:"messages,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type ListParams struct {
	User     User
	Status   string
	Query    string
	Page     int
	PageSize int
}

type Page struct {
	Items    []Ticket `json:"items"`
	Total    int      `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}

const ticketSelect = `
SELECT t.id, t.title, d.name, t.description, t.status, t.requester_id, t.assignee_id,
       r.name, t.resolve_note, t.created_at, t.updated_at, t.closed_at
FROM tickets t
JOIN devices d ON d.id = t.device_id
LEFT JOIN resolve_types r ON r.id = t.resolve_type_id`

func scanTicket(row pgx.Row) (Ticket, error) {
	var t Ticket
	err := row.Scan(&t.ID, &t.Title, &t.Device, &t.Description, &t.Status, &t.RequesterID, &t.AssigneeID,
		&t.ResolveType, &t.ResolveNote, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt)
	return t, err
}

func isFKViolation(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23503"
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (s *Store) Devices(ctx context.Context) ([]Lookup, error) { return s.lookup(ctx, "devices") }
func (s *Store) ResolveTypes(ctx context.Context) ([]Lookup, error) {
	return s.lookup(ctx, "resolve_types")
}

func (s *Store) lookup(ctx context.Context, table string) ([]Lookup, error) {
	rows, err := s.pool.Query(ctx, "SELECT id, name FROM "+table+" ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Lookup{}
	for rows.Next() {
		var l Lookup
		if err := rows.Scan(&l.ID, &l.Name); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) ListTickets(ctx context.Context, p ListParams) (Page, error) {
	where := []string{"TRUE"}
	args := []any{}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	isEmployee := p.User.Role == "employee"
	if isEmployee {
		add("t.requester_id = $%d", p.User.ID)
	}
	if p.Status != "" {
		add("t.status = $%d", p.Status)
	}
	if !isEmployee && p.Status == "ongoing" {
		add("t.assignee_id = $%d", p.User.ID)
	}
	if p.Query != "" {
		add(`t.title ILIKE $%d ESCAPE '\'`, "%"+escapeLike(p.Query)+"%")
	}
	cond := strings.Join(where, " AND ")

	order := "t.created_at ASC, t.id ASC"
	switch {
	case isEmployee:
		order = "t.created_at DESC, t.id DESC"
	case p.Status == "closed":
		order = "t.closed_at DESC, t.id DESC"
	}

	res := Page{Items: []Ticket{}, Page: p.Page, PageSize: p.PageSize}
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM tickets t WHERE "+cond, args...).Scan(&res.Total); err != nil {
		return res, err
	}
	q := fmt.Sprintf("%s WHERE %s ORDER BY %s LIMIT %d OFFSET %d", ticketSelect, cond, order, p.PageSize, (p.Page-1)*p.PageSize)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return res, err
		}
		res.Items = append(res.Items, t)
	}
	return res, rows.Err()
}

func (s *Store) GetTicket(ctx context.Context, u User, id int64) (Ticket, error) {
	t, err := scanTicket(s.pool.QueryRow(ctx, ticketSelect+" WHERE t.id=$1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, err
	}
	if u.Role == "employee" && t.RequesterID != u.ID {
		return Ticket{}, ErrNotFound
	}

	mrows, err := s.pool.Query(ctx, `
		SELECT m.id, m.author_id, CASE WHEN u.role = 'employee' THEN 'employee' ELSE 'support' END, m.body, m.created_at
		FROM messages m JOIN users u ON u.id = m.author_id
		WHERE m.ticket_id=$1 ORDER BY m.created_at, m.id`, id)
	if err != nil {
		return t, err
	}
	defer mrows.Close()
	for mrows.Next() {
		var m Message
		if err := mrows.Scan(&m.ID, &m.AuthorID, &m.AuthorRole, &m.Body, &m.CreatedAt); err != nil {
			return t, err
		}
		t.Messages = append(t.Messages, m)
	}
	if err := mrows.Err(); err != nil {
		return t, err
	}

	arows, err := s.pool.Query(ctx, `SELECT id, object_key, mime_type, size_bytes FROM attachments WHERE ticket_id=$1 ORDER BY id`, id)
	if err != nil {
		return t, err
	}
	defer arows.Close()
	for arows.Next() {
		var a Attachment
		if err := arows.Scan(&a.ID, &a.ObjectKey, &a.MimeType, &a.SizeBytes); err != nil {
			return t, err
		}
		t.Attachments = append(t.Attachments, a)
	}
	return t, arows.Err()
}

func (s *Store) CreateTicket(ctx context.Context, requesterID int64, title string, deviceID int, description string) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO tickets (requester_id, device_id, title, description) VALUES ($1,$2,$3,$4) RETURNING id`,
		requesterID, deviceID, title, description).Scan(&id)
	if isFKViolation(err) {
		return 0, ErrInvalid
	}
	if err != nil {
		return 0, err
	}
	if err := notifySupport(ctx, tx, id, NotifyNewTicket, requesterID); err != nil {
		return 0, err
	}
	return id, tx.Commit(ctx)
}

func (s *Store) DeleteTicket(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM tickets WHERE id=$1`, id)
	return err
}

func (s *Store) AddAttachment(ctx context.Context, ticketID int64, key, mime string, size int) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO attachments (ticket_id, object_key, mime_type, size_bytes) VALUES ($1,$2,$3,$4)`,
		ticketID, key, mime, size)
	return err
}

func (s *Store) AddMessage(ctx context.Context, u User, id int64, body string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	var requester int64
	var assignee *int64
	err = tx.QueryRow(ctx, `SELECT status, requester_id, assignee_id FROM tickets WHERE id=$1 FOR UPDATE`, id).
		Scan(&status, &requester, &assignee)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if u.Role == "employee" {
		if requester != u.ID {
			return ErrNotFound
		}
		if status == "closed" {
			return ErrConflict
		}
	} else {
		if assignee == nil || *assignee != u.ID {
			return ErrForbidden
		}
		if status != "ongoing" {
			return ErrConflict
		}
	}

	if _, err := tx.Exec(ctx, `INSERT INTO messages (ticket_id, author_id, body) VALUES ($1,$2,$3)`, id, u.ID, body); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE tickets SET updated_at=now() WHERE id=$1`, id); err != nil {
		return err
	}
	if u.Role == "employee" {
		if assignee != nil {
			if err := notify(ctx, tx, *assignee, id, NotifyReply, u.ID); err != nil {
				return err
			}
		}
	} else if err := notify(ctx, tx, requester, id, NotifyReply, u.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Accept(ctx context.Context, u User, id int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, u.ID); err != nil {
		return err
	}
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM tickets WHERE assignee_id=$1 AND status='ongoing'`, u.ID).Scan(&n); err != nil {
		return err
	}
	if n >= MaxOngoing {
		return ErrLimit
	}
	ct, err := tx.Exec(ctx,
		`UPDATE tickets SET status='ongoing', assignee_id=$1, updated_at=now() WHERE id=$2 AND status='pending'`, u.ID, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		var status string
		if err := tx.QueryRow(ctx, `SELECT status FROM tickets WHERE id=$1`, id).Scan(&status); errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return ErrConflict
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, ticket_id, kind, actor_id)
		SELECT requester_id, id, $2, $3 FROM tickets WHERE id = $1`, id, NotifyAccepted, u.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Close(ctx context.Context, u User, id int64, resolveTypeID int, note string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var requester int64
	err = tx.QueryRow(ctx, `
		UPDATE tickets SET status='closed', resolve_type_id=$1, resolve_note=$2, closed_at=now(), updated_at=now()
		WHERE id=$3 AND status='ongoing' AND assignee_id=$4
		RETURNING requester_id`, resolveTypeID, note, id, u.ID).Scan(&requester)
	if isFKViolation(err) {
		return ErrInvalid
	}
	if err == nil {
		if err := notify(ctx, tx, requester, id, NotifyClosed, u.ID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	var status string
	var assignee *int64
	err = tx.QueryRow(ctx, `SELECT status, assignee_id FROM tickets WHERE id=$1`, id).Scan(&status, &assignee)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status == "ongoing" && (assignee == nil || *assignee != u.ID) {
		return ErrForbidden
	}
	return ErrConflict
}

func (s *Store) GetAttachment(ctx context.Context, u User, ticketID, attID int64) (Attachment, error) {
	var a Attachment
	var requester int64
	err := s.pool.QueryRow(ctx, `
		SELECT a.id, a.object_key, a.mime_type, a.size_bytes, t.requester_id
		FROM attachments a JOIN tickets t ON t.id = a.ticket_id
		WHERE a.id = $1 AND a.ticket_id = $2`, attID, ticketID).
		Scan(&a.ID, &a.ObjectKey, &a.MimeType, &a.SizeBytes, &requester)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	if err != nil {
		return a, err
	}
	if u.Role == "employee" && requester != u.ID {
		return Attachment{}, ErrNotFound
	}
	return a, nil
}
