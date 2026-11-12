package sqlstorage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

var now = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

func TestListEventsToNotify(t *testing.T) {
	tests := []struct {
		name         string
		startIn      time.Duration
		notifyBefore time.Duration
		want         bool
	}{
		{"notify time reached", 10 * time.Minute, 15 * time.Minute, true},
		{"exactly at notify time", 15 * time.Minute, 15 * time.Minute, true},
		{"long notice in days", 48 * time.Hour, 72 * time.Hour, true},
		{"too early", time.Hour, 15 * time.Minute, false},
		{"one second too early", 15*time.Minute + time.Second, 15 * time.Minute, false},
		{"starts right now", 0, 15 * time.Minute, false},
		{"already started", -time.Minute, 15 * time.Minute, false},
		{"notification disabled", 10 * time.Minute, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStorage(t)
			e := newEvent(1, now.Add(tt.startIn), time.Hour)
			e.NotifyBefore = tt.notifyBefore
			mustCreate(t, s, e)

			got, err := s.ListEventsToNotify(context.Background(), now)
			if err != nil {
				t.Fatal(err)
			}

			if tt.want {
				if len(got) != 1 {
					t.Fatalf("got %d events, want 1", len(got))
				}
				assertEvent(t, got[0], e)
				return
			}
			if len(got) != 0 {
				t.Fatalf("got %d events, want none: %+v", len(got), got)
			}
		})
	}
}

func TestListEventsToNotifyOrder(t *testing.T) {
	s := newStorage(t)
	later := newEvent(1, now.Add(10*time.Minute), time.Hour)
	sooner := newEvent(2, now.Add(5*time.Minute), time.Hour)
	mustCreate(t, s, later)
	mustCreate(t, s, sooner)

	got, err := s.ListEventsToNotify(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != sooner.ID || got[1].ID != later.ID {
		t.Fatalf("events must be ordered by start_at, got %+v", got)
	}
}

func TestMarkNotified(t *testing.T) {
	ctx := context.Background()

	t.Run("notified event is not listed again", func(t *testing.T) {
		s := newStorage(t)
		e := newEvent(1, now.Add(10*time.Minute), time.Hour)
		other := newEvent(2, now.Add(10*time.Minute), time.Hour)
		mustCreate(t, s, e)
		mustCreate(t, s, other)

		if err := s.MarkNotified(ctx, e.ID, now); err != nil {
			t.Fatal(err)
		}

		got, err := s.ListEventsToNotify(ctx, now.Add(time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != other.ID {
			t.Fatalf("want only the other event, got %+v", got)
		}
	})

	t.Run("event itself is not changed", func(t *testing.T) {
		s := newStorage(t)
		e := newEvent(1, now.Add(10*time.Minute), time.Hour)
		mustCreate(t, s, e)

		if err := s.MarkNotified(ctx, e.ID, now); err != nil {
			t.Fatal(err)
		}

		got, err := s.ListEvents(ctx, now, now.Add(24*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d events, want 1", len(got))
		}
		assertEvent(t, got[0], e)
	})

	t.Run("not found", func(t *testing.T) {
		s := newStorage(t)
		if err := s.MarkNotified(ctx, uuid.New(), now); !errors.Is(err, storage.ErrEventNotFound) {
			t.Fatalf("err = %v, want %v", err, storage.ErrEventNotFound)
		}
	})
}

func TestDeleteEventsBefore(t *testing.T) {
	ctx := context.Background()
	yearAgo := now.AddDate(-1, 0, 0)

	t.Run("deletes only events finished before the border", func(t *testing.T) {
		s := newStorage(t)

		twoYearsAgo := newEvent(1, now.AddDate(-2, 0, 0), time.Hour)
		justOld := newEvent(2, yearAgo.Add(-2*time.Hour), time.Hour)
		endsAtBorder := newEvent(3, yearAgo.Add(-time.Hour), time.Hour)
		longRunning := newEvent(4, now.AddDate(0, -13, 0), 2*365*24*time.Hour)
		yesterday := newEvent(5, now.AddDate(0, 0, -1), time.Hour)
		future := newEvent(6, now.Add(24*time.Hour), time.Hour)
		for _, e := range []storage.Event{twoYearsAgo, justOld, endsAtBorder, longRunning, yesterday, future} {
			mustCreate(t, s, e)
		}

		deleted, err := s.DeleteEventsBefore(ctx, yearAgo)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 2 {
			t.Errorf("deleted = %d, want 2", deleted)
		}

		left, err := s.ListEvents(ctx, now.AddDate(-5, 0, 0), now.AddDate(5, 0, 0))
		if err != nil {
			t.Fatal(err)
		}
		leftIDs := make(map[uuid.UUID]bool, len(left))
		for _, e := range left {
			leftIDs[e.ID] = true
		}
		for _, gone := range []storage.Event{twoYearsAgo, justOld} {
			if leftIDs[gone.ID] {
				t.Errorf("event ending at %v must be deleted", gone.EndAt)
			}
		}
		for _, kept := range []storage.Event{endsAtBorder, longRunning, yesterday, future} {
			if !leftIDs[kept.ID] {
				t.Errorf("event %v–%v must be kept", kept.StartAt, kept.EndAt)
			}
		}
	})

	t.Run("nothing to delete is not an error", func(t *testing.T) {
		s := newStorage(t)
		mustCreate(t, s, newEvent(1, now, time.Hour))

		deleted, err := s.DeleteEventsBefore(ctx, yearAgo)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if deleted != 0 {
			t.Errorf("deleted = %d, want 0", deleted)
		}
	})
}
