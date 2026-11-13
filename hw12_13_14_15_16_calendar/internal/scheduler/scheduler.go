package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/notification"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type Storage interface {
	ListEventsToNotify(ctx context.Context, now time.Time) ([]storage.Event, error)
	MarkNotified(ctx context.Context, id uuid.UUID, at time.Time) error
	DeleteEventsBefore(ctx context.Context, before time.Time) (int64, error)
}

type Publisher interface {
	Publish(ctx context.Context, body []byte) error
}

type Scheduler struct {
	logg      Logger
	storage   Storage
	publisher Publisher
	interval  time.Duration
	retention time.Duration
	now       func() time.Time
}

func New(logg Logger, st Storage, pub Publisher, interval, retention time.Duration) *Scheduler {
	return &Scheduler{
		logg:      logg,
		storage:   st,
		publisher: pub,
		interval:  interval,
		retention: retention,
		now:       time.Now,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.logg.Info("scheduler started", "interval", s.interval, "retention", s.retention)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		s.tick(ctx, s.now())

		select {
		case <-ctx.Done():
			s.logg.Info("scheduler stopped")
			return
		case <-ticker.C:
		}
	}
}

func (s *Scheduler) tick(ctx context.Context, now time.Time) {
	s.notify(ctx, now)
	s.cleanup(ctx, now)
}

func (s *Scheduler) notify(ctx context.Context, now time.Time) {
	events, err := s.storage.ListEventsToNotify(ctx, now)
	if err != nil {
		s.logg.Error("list events to notify", "error", err)
		return
	}

	sent := 0
	for _, e := range events {
		// Ошибка с одним событием не должна останавливать остальные.
		if err := s.notifyEvent(ctx, e, now); err != nil {
			s.logg.Error("notify event", "eventId", e.ID, "error", err)
			continue
		}
		sent++
	}

	if len(events) > 0 {
		s.logg.Info("notifications sent", "sent", sent, "total", len(events))
	}
}

func (s *Scheduler) notifyEvent(ctx context.Context, e storage.Event, now time.Time) error {
	body, err := json.Marshal(notification.FromEvent(e))
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}
	if err := s.publisher.Publish(ctx, body); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	if err := s.storage.MarkNotified(ctx, e.ID, now); err != nil {
		return fmt.Errorf("mark notified: %w", err)
	}
	return nil
}

func (s *Scheduler) cleanup(ctx context.Context, now time.Time) {
	deleted, err := s.storage.DeleteEventsBefore(ctx, now.Add(-s.retention))
	if err != nil {
		s.logg.Error("delete old events", "error", err)
		return
	}
	if deleted > 0 {
		s.logg.Info("old events deleted", "count", deleted)
	}
}
