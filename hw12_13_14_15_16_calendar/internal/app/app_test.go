package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
	memorystorage "github.com/varexsk/hw12_13_14_15_calendar/internal/storage/memory"
)

type nopLogger struct{}

func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Warn(string, ...any)  {}
func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Error(string, ...any) {}

var base = time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)

func newApp() *App {
	return New(nopLogger{}, memorystorage.New())
}

func newEvent(start time.Time, dur time.Duration) storage.Event {
	return storage.Event{
		Title:   "event",
		StartAt: start,
		EndAt:   start.Add(dur),
		UserID:  1,
	}
}

func mustCreate(t *testing.T, a *App, start time.Time, dur time.Duration) uuid.UUID {
	t.Helper()
	id, err := a.CreateEvent(context.Background(), newEvent(start, dur))
	if err != nil {
		t.Fatalf("create event at %v: %v", start, err)
	}
	return id
}

func assertIDs(t *testing.T, got []storage.Event, want ...uuid.UUID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Errorf("position %d: got event starting %v, want another one", i, got[i].StartAt)
		}
	}
}

func TestCreateInvalidEvent(t *testing.T) {
	tests := []struct {
		name  string
		event storage.Event
	}{
		{"empty title", storage.Event{StartAt: base, EndAt: base.Add(time.Hour)}},
		{"end before start", storage.Event{Title: "x", StartAt: base, EndAt: base.Add(-time.Hour)}},
		{"zero duration", storage.Event{Title: "x", StartAt: base, EndAt: base}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApp()
			if _, err := a.CreateEvent(context.Background(), tt.event); !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidEvent)
			}
			got, _ := a.ListEvents(context.Background(), base.Add(-24*time.Hour), base.Add(24*time.Hour))
			assertIDs(t, got)
		})
	}
}

func TestCreateID(t *testing.T) {
	ctx := context.Background()

	t.Run("generates id when empty", func(t *testing.T) {
		a := newApp()
		id, err := a.CreateEvent(ctx, newEvent(base, time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if id == uuid.Nil {
			t.Fatal("returned id is Nil")
		}
		got, _ := a.ListEvents(ctx, base, base.Add(time.Hour))
		assertIDs(t, got, id)
	})

	t.Run("keeps given id", func(t *testing.T) {
		a := newApp()
		e := newEvent(base, time.Hour)
		e.ID = uuid.New()
		id, err := a.CreateEvent(ctx, e)
		if err != nil {
			t.Fatal(err)
		}
		if id != e.ID {
			t.Fatalf("id = %v, want %v", id, e.ID)
		}
	})

	t.Run("storage error returns Nil id", func(t *testing.T) {
		a := newApp()
		mustCreate(t, a, base, time.Hour)

		id, err := a.CreateEvent(ctx, newEvent(base, time.Hour))
		if !errors.Is(err, storage.ErrDateBusy) {
			t.Fatalf("err = %v, want %v", err, storage.ErrDateBusy)
		}
		if id != uuid.Nil {
			t.Fatalf("id = %v, want Nil: event was not created", id)
		}
	})
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("valid", func(t *testing.T) {
		a := newApp()
		id := mustCreate(t, a, base, time.Hour)

		if err := a.UpdateEvent(ctx, id, newEvent(base.Add(2*time.Hour), time.Hour)); err != nil {
			t.Fatal(err)
		}
		got, _ := a.ListEvents(ctx, base.Add(2*time.Hour), base.Add(3*time.Hour))
		assertIDs(t, got, id)
	})

	t.Run("invalid event is rejected and nothing changes", func(t *testing.T) {
		a := newApp()
		id := mustCreate(t, a, base, time.Hour)

		bad := newEvent(base.Add(2*time.Hour), time.Hour)
		bad.Title = ""
		if err := a.UpdateEvent(ctx, id, bad); !errors.Is(err, ErrInvalidEvent) {
			t.Fatalf("err = %v, want %v", err, ErrInvalidEvent)
		}
		got, _ := a.ListEvents(ctx, base, base.Add(time.Hour))
		assertIDs(t, got, id)
	})

	t.Run("not found", func(t *testing.T) {
		a := newApp()
		err := a.UpdateEvent(ctx, uuid.New(), newEvent(base, time.Hour))
		if !errors.Is(err, storage.ErrEventNotFound) {
			t.Fatalf("err = %v, want %v", err, storage.ErrEventNotFound)
		}
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	a := newApp()
	id := mustCreate(t, a, base, time.Hour)

	if err := a.DeleteEvent(ctx, id); err != nil {
		t.Fatal(err)
	}
	got, _ := a.ListEvents(ctx, base, base.Add(time.Hour))
	assertIDs(t, got)

	if err := a.DeleteEvent(ctx, id); !errors.Is(err, storage.ErrEventNotFound) {
		t.Fatalf("second delete: err = %v, want %v", err, storage.ErrEventNotFound)
	}
}

func TestListDay(t *testing.T) {
	msk := time.FixedZone("MSK", 3*60*60)
	day := time.Date(2026, 1, 10, 0, 0, 0, 0, msk)

	a := newApp()
	earlyMorning := mustCreate(t, a, day.Add(time.Hour), time.Hour)
	lateEvening := mustCreate(t, a, day.Add(23*time.Hour), time.Hour)
	mustCreate(t, a, day.Add(-time.Hour), time.Hour)
	mustCreate(t, a, day.AddDate(0, 0, 1), time.Hour)

	got, err := a.ListDay(context.Background(), day.Add(15*time.Hour+30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, got, earlyMorning, lateEvening)
}

func TestListWeek(t *testing.T) {
	monday := time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)

	a := newApp()
	first := mustCreate(t, a, monday, time.Hour)
	last := mustCreate(t, a, monday.AddDate(0, 0, 7).Add(-time.Hour), time.Hour)
	mustCreate(t, a, monday.Add(-time.Hour), time.Hour)
	mustCreate(t, a, monday.AddDate(0, 0, 7), time.Hour)

	got, err := a.ListWeek(context.Background(), monday)
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, got, first, last)
}

func TestListMonth(t *testing.T) {
	feb := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	a := newApp()
	first := mustCreate(t, a, feb, time.Hour)
	// последние полсекунды февраля: 28 фев 23:59:59.5–00:00:00
	last := mustCreate(t, a, feb.AddDate(0, 1, 0).Add(-500*time.Millisecond), 500*time.Millisecond)
	mustCreate(t, a, feb.Add(-time.Hour), time.Hour)
	mustCreate(t, a, feb.AddDate(0, 1, 0), time.Hour)
	mustCreate(t, a, feb.AddDate(0, 1, 14), time.Hour)

	got, err := a.ListMonth(context.Background(), feb.AddDate(0, 0, 14))
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, got, first, last)
}
