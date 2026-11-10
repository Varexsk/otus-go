package sqlstorage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует драйвер "pgx" в database/sql
	"github.com/jmoiron/sqlx"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

var errNotImplemented = errors.New("not implemented")

type Storage struct {
	dsn string
	db  *sqlx.DB
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

func (s *Storage) CreateEvent(_ context.Context, _ storage.Event) error {
	return errNotImplemented
}

func (s *Storage) DeleteEvent(_ context.Context, _ uuid.UUID) error {
	return errNotImplemented
}

func (s *Storage) UpdateEvent(_ context.Context, _ uuid.UUID, _ storage.Event) error {
	return errNotImplemented
}

func (s *Storage) ListEvents(_ context.Context, _, _ time.Time) ([]storage.Event, error) {
	return nil, errNotImplemented
}
