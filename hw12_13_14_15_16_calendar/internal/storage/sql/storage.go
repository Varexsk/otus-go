package sqlstorage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

type Storage struct { // TODO
}

func New(dsn string) (*Storage, error) {
	// TODO: Initialize the SQL storage with the provided DSN
	return &Storage{}, nil
}

func (s *Storage) Connect(ctx context.Context) error {
	// TODO
	return nil
}

func (s *Storage) Close(ctx context.Context) error {
	// TODO
	return nil
}

func (s *Storage) CreateEvent(ctx context.Context, event storage.Event) error {
	// TODO
	return nil
}

func (s *Storage) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	// TODO
	return nil
}

func (s *Storage) UpdateEvent(ctx context.Context, id uuid.UUID, event storage.Event) error {
	// TODO
	return nil
}

func (s *Storage) ListEvents(ctx context.Context, from, to time.Time) ([]storage.Event, error) {
	// TODO
	return []storage.Event{}, nil
}
