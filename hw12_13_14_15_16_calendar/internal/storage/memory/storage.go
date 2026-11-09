package memorystorage

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

type Storage struct {
	events map[uuid.UUID]storage.Event
	mu     sync.RWMutex
}

func New() *Storage {
	return &Storage{
		events: make(map[uuid.UUID]storage.Event),
	}
}

func (s *Storage) CreateEvent(_ context.Context, event storage.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exist := s.events[event.ID]; exist {
		return storage.ErrEventAlreadyExist
	}

	if s.hasConflict(event, uuid.Nil) {
		return storage.ErrDateBusy
	}
	s.events[event.ID] = event
	return nil
}

func (s *Storage) DeleteEvent(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[id]; !ok {
		return storage.ErrEventNotFound
	}

	delete(s.events, id)
	return nil
}

func (s *Storage) UpdateEvent(_ context.Context, id uuid.UUID, event storage.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exist := s.events[id]; !exist {
		return storage.ErrEventNotFound
	}

	if s.hasConflict(event, id) {
		return storage.ErrDateBusy
	}

	event.ID = id
	s.events[id] = event
	return nil
}

func (s *Storage) ListEvents(_ context.Context, from, to time.Time) ([]storage.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var events []storage.Event
	for _, e := range s.events {
		if e.StartAt.Before(to) && e.EndAt.After(from) {
			events = append(events, e)
		}
	}

	slices.SortFunc(events, func(a, b storage.Event) int {
		return a.StartAt.Compare(b.StartAt)
	})

	return events, nil
}

func (s *Storage) ListEventsByUserID(_ context.Context, userID int64, from, to time.Time) ([]storage.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var events []storage.Event
	for _, e := range s.events {
		if e.UserID == userID {
			if e.StartAt.Before(to) && e.EndAt.After(from) {
				events = append(events, e)
			}
		}
	}

	slices.SortFunc(events, func(a, b storage.Event) int {
		return a.StartAt.Compare(b.StartAt)
	})

	return events, nil
}

func (s *Storage) hasConflict(event storage.Event, skipID uuid.UUID) bool {
	for id, e := range s.events {
		if id == skipID || e.UserID != event.UserID {
			continue
		}
		if overlaps(e, event) {
			return true
		}
	}
	return false
}

func overlaps(a, b storage.Event) bool {
	return a.StartAt.Before(b.EndAt) && b.StartAt.Before(a.EndAt)
}
