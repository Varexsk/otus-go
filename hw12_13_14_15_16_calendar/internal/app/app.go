package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

var ErrInvalidEvent = errors.New("invalid event")

type App struct {
	logg    Logger
	storage Storage
}

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
}

type Storage interface {
	CreateEvent(ctx context.Context, event storage.Event) error
	UpdateEvent(ctx context.Context, id uuid.UUID, event storage.Event) error
	DeleteEvent(ctx context.Context, id uuid.UUID) error
	ListEvents(ctx context.Context, from, to time.Time) ([]storage.Event, error)
}

func New(logger Logger, storage Storage) *App {
	return &App{
		logg:    logger,
		storage: storage,
	}
}

func (a *App) CreateEvent(ctx context.Context, event storage.Event) (uuid.UUID, error) {
	if err := validate(event); err != nil {
		return uuid.Nil, err
	}
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if err := a.storage.CreateEvent(ctx, event); err != nil {
		return uuid.Nil, err
	}
	return event.ID, nil
}

func (a *App) UpdateEvent(ctx context.Context, id uuid.UUID, event storage.Event) error {
	if err := validate(event); err != nil {
		return err
	}
	return a.storage.UpdateEvent(ctx, id, event)
}

func (a *App) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	return a.storage.DeleteEvent(ctx, id)
}

func (a *App) ListEvents(ctx context.Context, from, to time.Time) ([]storage.Event, error) {
	return a.storage.ListEvents(ctx, from, to)
}

func (a *App) ListDay(ctx context.Context, date time.Time) ([]storage.Event, error) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	return a.storage.ListEvents(ctx, start, start.AddDate(0, 0, 1))
}

func (a *App) ListWeek(ctx context.Context, start time.Time) ([]storage.Event, error) {
	return a.storage.ListEvents(ctx, start, start.AddDate(0, 0, 7))
}

func (a *App) ListMonth(ctx context.Context, month time.Time) ([]storage.Event, error) {
	startMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	endMonth := startMonth.AddDate(0, 1, 0)
	return a.storage.ListEvents(ctx, startMonth, endMonth)
}

func validate(e storage.Event) error {
	if e.Title == "" {
		return fmt.Errorf("%w: empty title", ErrInvalidEvent)
	}
	if !e.EndAt.After(e.StartAt) {
		return fmt.Errorf("%w: end must be after start", ErrInvalidEvent)
	}
	return nil
}
