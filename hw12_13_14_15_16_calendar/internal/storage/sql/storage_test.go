package sqlstorage

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

const dsnEnv = "CALENDAR_TEST_DSN"

var base = time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)

func newStorage(t *testing.T) *Storage {
	t.Helper()
	dsn := os.Getenv(dsnEnv)
	if dsn == "" {
		t.Skipf("%s is not set, skipping SQL storage tests", dsnEnv)
	}

	ctx := context.Background()
	s := New(dsn)
	if err := s.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = s.Close(ctx) })

	if _, err := s.db.ExecContext(ctx, "TRUNCATE events"); err != nil {
		t.Fatalf("truncate events: %v", err)
	}
	return s
}

func newEvent(userID int64, start time.Time, dur time.Duration) storage.Event {
	return storage.Event{
		ID:           uuid.New(),
		Title:        "event",
		StartAt:      start,
		EndAt:        start.Add(dur),
		Description:  "description",
		UserID:       userID,
		NotifyBefore: 15 * time.Minute,
	}
}

func mustCreate(t *testing.T, s *Storage, e storage.Event) {
	t.Helper()
	if err := s.CreateEvent(context.Background(), e); err != nil {
		t.Fatalf("create event: %v", err)
	}
}

func assertEvent(t *testing.T, got, want storage.Event) {
	t.Helper()
	if got.ID != want.ID || got.Title != want.Title || got.Description != want.Description ||
		got.UserID != want.UserID || got.NotifyBefore != want.NotifyBefore ||
		!got.StartAt.Equal(want.StartAt) || !got.EndAt.Equal(want.EndAt) {
		t.Fatalf("event = %+v, want %+v", got, want)
	}
}

func TestCreateAndList(t *testing.T) {
	s := newStorage(t)
	e := newEvent(1, base, time.Hour)
	mustCreate(t, s, e)

	got, err := s.ListEvents(context.Background(), base.Add(-time.Hour), base.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	assertEvent(t, got[0], e)
}

func TestCreateDuplicateID(t *testing.T) {
	s := newStorage(t)
	e := newEvent(1, base, time.Hour)
	mustCreate(t, s, e)

	dup := newEvent(2, base.Add(24*time.Hour), time.Hour)
	dup.ID = e.ID
	if err := s.CreateEvent(context.Background(), dup); !errors.Is(err, storage.ErrEventAlreadyExist) {
		t.Fatalf("err = %v, want %v", err, storage.ErrEventAlreadyExist)
	}
}

func TestCreateDateBusy(t *testing.T) {
	tests := []struct {
		name    string
		event   storage.Event
		wantErr error
	}{
		{"same time", newEvent(1, base, time.Hour), storage.ErrDateBusy},
		{"overlaps start", newEvent(1, base.Add(-30*time.Minute), time.Hour), storage.ErrDateBusy},
		{"overlaps end", newEvent(1, base.Add(30*time.Minute), time.Hour), storage.ErrDateBusy},
		{"inside", newEvent(1, base.Add(15*time.Minute), 15*time.Minute), storage.ErrDateBusy},
		{"contains", newEvent(1, base.Add(-time.Hour), 3*time.Hour), storage.ErrDateBusy},
		{"other user same time", newEvent(2, base, time.Hour), nil},
		{"ends right at start", newEvent(1, base.Add(-time.Hour), time.Hour), nil},
		{"starts right at end", newEvent(1, base.Add(time.Hour), time.Hour), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStorage(t)
			mustCreate(t, s, newEvent(1, base, time.Hour))

			err := s.CreateEvent(context.Background(), tt.event)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("moves event and keeps id", func(t *testing.T) {
		s := newStorage(t)
		e := newEvent(1, base, time.Hour)
		mustCreate(t, s, e)

		moved := newEvent(1, base.Add(2*time.Hour), time.Hour)
		moved.Title = "moved"
		if err := s.UpdateEvent(ctx, e.ID, moved); err != nil {
			t.Fatal(err)
		}

		got, err := s.ListEvents(ctx, base.Add(-24*time.Hour), base.Add(24*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d events, want 1", len(got))
		}
		moved.ID = e.ID // ID берётся из параметра, а не из события
		assertEvent(t, got[0], moved)
	})

	t.Run("same time is not busy with itself", func(t *testing.T) {
		s := newStorage(t)
		e := newEvent(1, base, time.Hour)
		mustCreate(t, s, e)

		e.Title = "renamed"
		if err := s.UpdateEvent(ctx, e.ID, e); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	})

	t.Run("into busy slot", func(t *testing.T) {
		s := newStorage(t)
		first := newEvent(1, base, time.Hour)
		second := newEvent(1, base.Add(2*time.Hour), time.Hour)
		mustCreate(t, s, first)
		mustCreate(t, s, second)

		second.StartAt, second.EndAt = first.StartAt, first.EndAt
		if err := s.UpdateEvent(ctx, second.ID, second); !errors.Is(err, storage.ErrDateBusy) {
			t.Fatalf("err = %v, want %v", err, storage.ErrDateBusy)
		}
	})

	t.Run("not found", func(t *testing.T) {
		s := newStorage(t)
		e := newEvent(1, base, time.Hour)
		if err := s.UpdateEvent(ctx, e.ID, e); !errors.Is(err, storage.ErrEventNotFound) {
			t.Fatalf("err = %v, want %v", err, storage.ErrEventNotFound)
		}
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	s := newStorage(t)
	e := newEvent(1, base, time.Hour)
	mustCreate(t, s, e)

	if err := s.DeleteEvent(ctx, e.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ListEvents(ctx, base.Add(-time.Hour), base.Add(2*time.Hour)); len(got) != 0 {
		t.Fatalf("event still listed after delete: %+v", got)
	}
	if err := s.DeleteEvent(ctx, e.ID); !errors.Is(err, storage.ErrEventNotFound) {
		t.Fatalf("second delete: err = %v, want %v", err, storage.ErrEventNotFound)
	}
}

func TestListEvents(t *testing.T) {
	ctx := context.Background()
	from := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)

	overnight := newEvent(1, from.Add(-time.Hour), 2*time.Hour)
	morning := newEvent(2, from.Add(9*time.Hour), time.Hour)
	noon := newEvent(3, from.Add(12*time.Hour), time.Hour)
	evening := newEvent(4, from.Add(20*time.Hour), time.Hour)
	lateNight := newEvent(5, to.Add(-30*time.Minute), time.Hour)
	endsAtFrom := newEvent(6, from.Add(-time.Hour), time.Hour)
	startsAtTo := newEvent(7, to, time.Hour)

	s := newStorage(t)
	for _, e := range []storage.Event{evening, startsAtTo, noon, lateNight, endsAtFrom, morning, overnight} {
		mustCreate(t, s, e)
	}

	got, err := s.ListEvents(ctx, from, to)
	if err != nil {
		t.Fatal(err)
	}

	want := []storage.Event{overnight, morning, noon, evening, lateNight}
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].ID != want[i].ID {
			t.Errorf("position %d: got event starting %v, want starting %v", i, got[i].StartAt, want[i].StartAt)
		}
	}
}

func TestConcurrentCreateSameSlot(t *testing.T) {
	const workers = 20
	s := newStorage(t)

	var (
		wg        sync.WaitGroup
		successes atomic.Int32
	)
	start := make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := s.CreateEvent(context.Background(), newEvent(1, base, time.Hour))
			switch {
			case err == nil:
				successes.Add(1)
			case !errors.Is(err, storage.ErrDateBusy):
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	close(start)
	wg.Wait()

	if got := successes.Load(); got != 1 {
		t.Fatalf("successful creates = %d, want exactly 1", got)
	}
}
