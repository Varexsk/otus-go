package memorystorage

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

var base = time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)

func newEvent(userID int64, start time.Time, dur time.Duration) storage.Event {
	return storage.Event{
		ID:      uuid.New(),
		Title:   "event",
		StartAt: start,
		EndAt:   start.Add(dur),
		UserID:  userID,
	}
}

func mustCreate(t *testing.T, s *Storage, e storage.Event) {
	t.Helper()
	if err := s.CreateEvent(context.Background(), e); err != nil {
		t.Fatalf("create event: %v", err)
	}
}

func TestCreateAndList(t *testing.T) {
	s := New()
	e := newEvent(1, base, time.Hour)
	mustCreate(t, s, e)

	got, err := s.ListEvents(context.Background(), base.Add(-time.Hour), base.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != e {
		t.Fatalf("ListEvents = %+v, want [%+v]", got, e)
	}
}

func TestCreateDuplicateID(t *testing.T) {
	s := New()
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
			s := New()
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
		s := New()
		e := newEvent(1, base, time.Hour)
		mustCreate(t, s, e)

		moved := newEvent(1, base.Add(2*time.Hour), time.Hour)
		moved.ID = uuid.Nil
		if err := s.UpdateEvent(ctx, e.ID, moved); err != nil {
			t.Fatal(err)
		}

		got, _ := s.ListEvents(ctx, base.Add(-24*time.Hour), base.Add(24*time.Hour))
		if len(got) != 1 {
			t.Fatalf("got %d events, want 1", len(got))
		}
		if got[0].ID != e.ID {
			t.Errorf("ID = %v, want %v", got[0].ID, e.ID)
		}
		if !got[0].StartAt.Equal(moved.StartAt) {
			t.Errorf("StartAt = %v, want %v", got[0].StartAt, moved.StartAt)
		}
	})

	t.Run("same time is not busy with itself", func(t *testing.T) {
		s := New()
		e := newEvent(1, base, time.Hour)
		mustCreate(t, s, e)

		e.Title = "renamed"
		if err := s.UpdateEvent(ctx, e.ID, e); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	})

	t.Run("into busy slot", func(t *testing.T) {
		s := New()
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
		s := New()
		e := newEvent(1, base, time.Hour)
		if err := s.UpdateEvent(ctx, e.ID, e); !errors.Is(err, storage.ErrEventNotFound) {
			t.Fatalf("err = %v, want %v", err, storage.ErrEventNotFound)
		}
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	s := New()
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

	s := New()
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
	const (
		rounds  = 50
		workers = 100
	)
	for round := 0; round < rounds; round++ {
		createSameSlot(t, workers)
	}
}

func createSameSlot(t *testing.T, workers int) {
	t.Helper()
	s := New()

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
