package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/notification"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

var (
	now     = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	errFake = errors.New("fake failure")
)

type fakeLogger struct {
	mu     sync.Mutex
	errors []string
}

func (l *fakeLogger) Info(string, ...any) {}

func (l *fakeLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, msg)
}

type fakeStorage struct {
	mu        sync.Mutex
	events    []storage.Event
	listErr   error
	markErr   map[uuid.UUID]error
	deleteErr error

	listCalls    int
	listNow      time.Time
	marked       []uuid.UUID
	markedAt     time.Time
	deleteBefore time.Time
}

func (f *fakeStorage) ListEventsToNotify(_ context.Context, n time.Time) ([]storage.Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	f.listNow = n
	return f.events, f.listErr
}

func (f *fakeStorage) MarkNotified(_ context.Context, id uuid.UUID, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.markErr[id]; err != nil {
		return err
	}
	f.marked = append(f.marked, id)
	f.markedAt = at
	return nil
}

func (f *fakeStorage) DeleteEventsBefore(_ context.Context, before time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteBefore = before
	return 0, f.deleteErr
}

func (f *fakeStorage) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.listCalls
}

type fakePublisher struct {
	failFor map[uuid.UUID]bool
	bodies  [][]byte
}

func (p *fakePublisher) Publish(_ context.Context, body []byte) error {
	var n notification.Notification
	if err := json.Unmarshal(body, &n); err != nil {
		return err
	}
	if p.failFor[n.EventID] {
		return errFake
	}
	p.bodies = append(p.bodies, body)
	return nil
}

func newEvent(title string) storage.Event {
	return storage.Event{
		ID:           uuid.New(),
		Title:        title,
		StartAt:      now.Add(10 * time.Minute),
		EndAt:        now.Add(time.Hour),
		UserID:       42,
		NotifyBefore: 15 * time.Minute,
	}
}

func newScheduler(st Storage, pub Publisher) (*Scheduler, *fakeLogger) {
	logg := &fakeLogger{}
	return New(logg, st, pub, time.Minute, 365*24*time.Hour), logg
}

func TestTickPublishesAndMarks(t *testing.T) {
	e := newEvent("standup")
	st := &fakeStorage{events: []storage.Event{e}}
	pub := &fakePublisher{}
	s, logg := newScheduler(st, pub)

	s.tick(context.Background(), now)

	if !st.listNow.Equal(now) {
		t.Errorf("ListEventsToNotify got now=%v, want %v", st.listNow, now)
	}
	if len(pub.bodies) != 1 {
		t.Fatalf("published %d messages, want 1", len(pub.bodies))
	}

	var got notification.Notification
	if err := json.Unmarshal(pub.bodies[0], &got); err != nil {
		t.Fatalf("published body is not a notification: %v", err)
	}
	want := notification.Notification{EventID: e.ID, Title: "standup", StartAt: e.StartAt, UserID: 42}
	if got != want {
		t.Errorf("notification = %+v, want %+v", got, want)
	}

	if len(st.marked) != 1 || st.marked[0] != e.ID || !st.markedAt.Equal(now) {
		t.Errorf("marked = %v at %v, want [%s] at %v", st.marked, st.markedAt, e.ID, now)
	}
	if len(logg.errors) != 0 {
		t.Errorf("unexpected errors logged: %v", logg.errors)
	}
}

func TestTickPublishFailureDoesNotMark(t *testing.T) {
	failing, ok := newEvent("failing"), newEvent("ok")
	st := &fakeStorage{events: []storage.Event{failing, ok}}
	pub := &fakePublisher{failFor: map[uuid.UUID]bool{failing.ID: true}}
	s, logg := newScheduler(st, pub)

	s.tick(context.Background(), now)

	if len(st.marked) != 1 || st.marked[0] != ok.ID {
		t.Errorf("marked = %v, want only %s", st.marked, ok.ID)
	}
	if len(logg.errors) != 1 {
		t.Errorf("logged errors = %v, want exactly 1", logg.errors)
	}
}

func TestTickMarkFailureContinues(t *testing.T) {
	first, second := newEvent("first"), newEvent("second")
	st := &fakeStorage{
		events:  []storage.Event{first, second},
		markErr: map[uuid.UUID]error{first.ID: errFake},
	}
	pub := &fakePublisher{}
	s, logg := newScheduler(st, pub)

	s.tick(context.Background(), now)

	if len(pub.bodies) != 2 {
		t.Errorf("published %d, want 2", len(pub.bodies))
	}
	if len(st.marked) != 1 || st.marked[0] != second.ID {
		t.Errorf("marked = %v, want only %s", st.marked, second.ID)
	}
	if len(logg.errors) != 1 {
		t.Errorf("logged errors = %v, want exactly 1", logg.errors)
	}
}

func TestTickListFailureStillCleansUp(t *testing.T) {
	st := &fakeStorage{listErr: errFake}
	pub := &fakePublisher{}
	s, logg := newScheduler(st, pub)

	s.tick(context.Background(), now)

	if len(pub.bodies) != 0 {
		t.Errorf("published %d, want 0", len(pub.bodies))
	}
	if st.deleteBefore.IsZero() {
		t.Error("cleanup must run even if listing failed")
	}
	if len(logg.errors) != 1 {
		t.Errorf("logged errors = %v, want exactly 1", logg.errors)
	}
}

func TestTickCleanupBorder(t *testing.T) {
	st := &fakeStorage{}
	s, _ := newScheduler(st, &fakePublisher{})

	s.tick(context.Background(), now)

	want := now.Add(-365 * 24 * time.Hour)
	if !st.deleteBefore.Equal(want) {
		t.Errorf("DeleteEventsBefore(%v), want %v", st.deleteBefore, want)
	}
}

func TestTickCleanupFailureIsLogged(t *testing.T) {
	st := &fakeStorage{deleteErr: errFake}
	s, logg := newScheduler(st, &fakePublisher{})

	s.tick(context.Background(), now)

	if len(logg.errors) != 1 {
		t.Errorf("logged errors = %v, want exactly 1", logg.errors)
	}
}

func TestRunTicksUntilCancel(t *testing.T) {
	st := &fakeStorage{}
	s := New(&fakeLogger{}, st, &fakePublisher{}, 10*time.Millisecond, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for st.calls() < 3 {
		select {
		case <-deadline:
			t.Fatalf("only %d ticks in 2s", st.calls())
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after context cancel")
	}
}

func TestRunFirstTickIsImmediate(t *testing.T) {
	st := &fakeStorage{}
	s := New(&fakeLogger{}, st, &fakePublisher{}, time.Hour, time.Hour)
	s.now = func() time.Time { return now }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()

	deadline := time.After(time.Second)
	for st.calls() == 0 {
		select {
		case <-deadline:
			t.Fatal("first tick did not happen immediately (interval is 1h)")
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	<-done
}
