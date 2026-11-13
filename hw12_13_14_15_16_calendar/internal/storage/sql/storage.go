package sqlstorage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	// Registration driver "pgx" for database/sql.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

const (
	codeUniqueViolation    = "23505"
	codeExclusionViolation = "23P01"
)

type Storage struct {
	dsn string
	db  *sqlx.DB
}

type eventRow struct {
	ID           uuid.UUID `db:"id"`
	Title        string    `db:"title"`
	StartAt      time.Time `db:"start_at"`
	EndAt        time.Time `db:"end_at"`
	Description  string    `db:"description"`
	UserID       int64     `db:"user_id"`
	NotifyBefore int64     `db:"notify_before"`
}

func (r eventRow) toEvent() storage.Event {
	return storage.Event{
		ID:           r.ID,
		Title:        r.Title,
		StartAt:      r.StartAt,
		EndAt:        r.EndAt,
		Description:  r.Description,
		UserID:       r.UserID,
		NotifyBefore: time.Duration(r.NotifyBefore),
	}
}

func New(dsn string) *Storage {
	return &Storage{dsn: dsn}
}

func (s *Storage) Connect(ctx context.Context) error {
	db, err := sqlx.ConnectContext(ctx, "pgx", s.dsn)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	s.db = db
	return nil
}

func (s *Storage) Close(_ context.Context) error {
	if s.db == nil {
		return nil
	}
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}
	return nil
}

func (s *Storage) CreateEvent(ctx context.Context, event storage.Event) error {
	const query = `
		INSERT INTO events (id, title, start_at, end_at, description, user_id, notify_before)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := s.db.ExecContext(ctx, query,
		event.ID, event.Title, event.StartAt, event.EndAt,
		event.Description, event.UserID, int64(event.NotifyBefore))
	if err != nil {
		return fmt.Errorf("insert event: %w", mapError(err))
	}
	return nil
}

func (s *Storage) UpdateEvent(ctx context.Context, id uuid.UUID, event storage.Event) error {
	const query = `
		UPDATE events
		SET title = $2, start_at = $3, end_at = $4, description = $5, user_id = $6, notify_before = $7
		WHERE id = $1`

	res, err := s.db.ExecContext(ctx, query,
		id, event.Title, event.StartAt, event.EndAt,
		event.Description, event.UserID, int64(event.NotifyBefore))
	if err != nil {
		return fmt.Errorf("update event: %w", mapError(err))
	}
	return checkAffected(res.RowsAffected())
}

func (s *Storage) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	return checkAffected(res.RowsAffected())
}

func (s *Storage) ListEvents(ctx context.Context, from, to time.Time) ([]storage.Event, error) {
	const query = `
		SELECT id, title, start_at, end_at, description, user_id, notify_before
		FROM events
		WHERE start_at < $2 AND end_at > $1
		ORDER BY start_at`

	var rows []eventRow
	if err := s.db.SelectContext(ctx, &rows, query, from, to); err != nil {
		return nil, fmt.Errorf("select events: %w", err)
	}

	events := make([]storage.Event, 0, len(rows))
	for _, r := range rows {
		events = append(events, r.toEvent())
	}
	return events, nil
}

func (s *Storage) ListEventsToNotify(ctx context.Context, now time.Time) ([]storage.Event, error) {
	const query = `
		SELECT id, title, start_at, end_at, description, user_id, notify_before
		FROM events
		WHERE notify_before > 0 
		AND notified_at IS NULL 
		AND start_at > $1
		AND start_at - make_interval(secs => notify_before/1e9) <= $1
		ORDER by start_at ASC
	`
	var rows []eventRow
	if err := s.db.SelectContext(ctx, &rows, query, now); err != nil {
		return nil, fmt.Errorf("select events: %w", err)
	}

	events := make([]storage.Event, 0, len(rows))
	for _, r := range rows {
		events = append(events, r.toEvent())
	}
	return events, nil
}

func (s *Storage) MarkNotified(ctx context.Context, id uuid.UUID, at time.Time) error {
	const query = `
		UPDATE events
		SET notified_at = $2
		WHERE id = $1`

	res, err := s.db.ExecContext(ctx, query, id, at)
	if err != nil {
		return fmt.Errorf("mark notified event: %w", mapError(err))
	}
	return checkAffected(res.RowsAffected())
}

func (s *Storage) DeleteEventsBefore(ctx context.Context, before time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE end_at < $1`, before)
	if err != nil {
		return 0, fmt.Errorf("delete event: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return n, nil
}

func checkAffected(n int64, err error) error {
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return storage.ErrEventNotFound
	}
	return nil
}

func mapError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case codeExclusionViolation:
		return storage.ErrDateBusy
	case codeUniqueViolation:
		return storage.ErrEventAlreadyExist
	default:
		return err
	}
}
