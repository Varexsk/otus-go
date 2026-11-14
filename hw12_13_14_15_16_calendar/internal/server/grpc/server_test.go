package internalgrpc

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/app"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/server/grpc/pb"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeApp struct {
	mu       sync.Mutex
	createID uuid.UUID
	events   []storage.Event
	err      error

	called   string
	gotID    uuid.UUID
	gotEvent storage.Event
	gotDate  time.Time
}

func (f *fakeApp) record(name string, id uuid.UUID, e storage.Event, d time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.called, f.gotID, f.gotEvent, f.gotDate = name, id, e, d
}

type appCall struct {
	name  string
	id    uuid.UUID
	event storage.Event
	date  time.Time
}

func (f *fakeApp) lastCall() appCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return appCall{name: f.called, id: f.gotID, event: f.gotEvent, date: f.gotDate}
}

func (f *fakeApp) CreateEvent(_ context.Context, e storage.Event) (uuid.UUID, error) {
	f.record("CreateEvent", uuid.Nil, e, time.Time{})
	return f.createID, f.err
}

func (f *fakeApp) UpdateEvent(_ context.Context, id uuid.UUID, e storage.Event) error {
	f.record("UpdateEvent", id, e, time.Time{})
	return f.err
}

func (f *fakeApp) DeleteEvent(_ context.Context, id uuid.UUID) error {
	f.record("DeleteEvent", id, storage.Event{}, time.Time{})
	return f.err
}

func (f *fakeApp) ListDay(_ context.Context, d time.Time) ([]storage.Event, error) {
	f.record("ListDay", uuid.Nil, storage.Event{}, d)
	return f.events, f.err
}

func (f *fakeApp) ListWeek(_ context.Context, d time.Time) ([]storage.Event, error) {
	f.record("ListWeek", uuid.Nil, storage.Event{}, d)
	return f.events, f.err
}

func (f *fakeApp) ListMonth(_ context.Context, d time.Time) ([]storage.Event, error) {
	f.record("ListMonth", uuid.Nil, storage.Event{}, d)
	return f.events, f.err
}

type fakeLogger struct {
	mu      sync.Mutex
	entries []map[string]any
	errors  int
}

func (l *fakeLogger) Info(_ string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := make(map[string]any)
	for i := 0; i+1 < len(args); i += 2 {
		key, _ := args[i].(string)
		entry[key] = args[i+1]
	}
	l.entries = append(l.entries, entry)
}

func (l *fakeLogger) Error(string, ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors++
}

var (
	testID    = uuid.MustParse("6b900a50-01ce-47e9-a734-582f72fa7488")
	testStart = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	errDB     = errors.New("connection refused")
)

func newTestClient(t *testing.T, fa *fakeApp) (pb.EventServiceClient, *fakeLogger) {
	t.Helper()

	lis := bufconn.Listen(1 << 20)
	logg := &fakeLogger{}
	srv := New(logg, fa, "bufnet")
	go srv.server.Serve(lis) //nolint:errcheck
	t.Cleanup(srv.server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return pb.NewEventServiceClient(conn), logg
}

func validPBEvent() *pb.Event {
	return &pb.Event{
		Title:        "standup",
		StartAt:      timestamppb.New(testStart),
		EndAt:        timestamppb.New(testStart.Add(time.Hour)),
		UserId:       7,
		NotifyBefore: durationpb.New(15 * time.Minute),
	}
}

func TestCreateEvent(t *testing.T) {
	fa := &fakeApp{createID: testID}
	client, _ := newTestClient(t, fa)

	resp, err := client.CreateEvent(context.Background(), &pb.CreateEventRequest{Event: validPBEvent()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetId() != testID.String() {
		t.Errorf("id = %q, want %q", resp.GetId(), testID)
	}

	got := fa.lastCall().event
	want := storage.Event{
		Title:        "standup",
		StartAt:      testStart,
		EndAt:        testStart.Add(time.Hour),
		UserID:       7,
		NotifyBefore: 15 * time.Minute,
	}
	if got != want {
		t.Errorf("app got event %+v, want %+v", got, want)
	}
}

func TestInvalidArguments(t *testing.T) {
	noStart := validPBEvent()
	noStart.StartAt = nil
	noEnd := validPBEvent()
	noEnd.EndAt = nil

	tests := []struct {
		name string
		call func(c pb.EventServiceClient) error
	}{
		{"create without event", func(c pb.EventServiceClient) error {
			_, err := c.CreateEvent(context.Background(), &pb.CreateEventRequest{})
			return err
		}},
		{"create without start_at", func(c pb.EventServiceClient) error {
			_, err := c.CreateEvent(context.Background(), &pb.CreateEventRequest{Event: noStart})
			return err
		}},
		{"create without end_at", func(c pb.EventServiceClient) error {
			_, err := c.CreateEvent(context.Background(), &pb.CreateEventRequest{Event: noEnd})
			return err
		}},
		{"update with bad id", func(c pb.EventServiceClient) error {
			_, err := c.UpdateEvent(context.Background(), &pb.UpdateEventRequest{Id: "bad", Event: validPBEvent()})
			return err
		}},
		{"delete with bad id", func(c pb.EventServiceClient) error {
			_, err := c.DeleteEvent(context.Background(), &pb.DeleteEventRequest{Id: "bad"})
			return err
		}},
		{"list without date", func(c pb.EventServiceClient) error {
			_, err := c.ListDay(context.Background(), &pb.ListDayRequest{})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fa := &fakeApp{}
			client, _ := newTestClient(t, fa)

			err := tt.call(client)
			if code := status.Code(err); code != codes.InvalidArgument {
				t.Errorf("code = %s, want %s (err: %v)", code, codes.InvalidArgument, err)
			}
			if called := fa.lastCall().name; called != "" {
				t.Errorf("app must not be called, but %s was", called)
			}
		})
	}
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		appErr   error
		wantCode codes.Code
	}{
		{app.ErrInvalidEvent, codes.InvalidArgument},
		{storage.ErrEventNotFound, codes.NotFound},
		{storage.ErrEventAlreadyExist, codes.AlreadyExists},
		{storage.ErrDateBusy, codes.FailedPrecondition},
		{errDB, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.appErr.Error(), func(t *testing.T) {
			client, logg := newTestClient(t, &fakeApp{err: tt.appErr})

			_, err := client.DeleteEvent(context.Background(), &pb.DeleteEventRequest{Id: testID.String()})
			if code := status.Code(err); code != tt.wantCode {
				t.Errorf("code = %s, want %s", code, tt.wantCode)
			}

			if tt.wantCode == codes.Internal {
				if msg := status.Convert(err).Message(); msg != "internal error" {
					t.Errorf("internal details leaked to client: %q", msg)
				}
				logg.mu.Lock()
				defer logg.mu.Unlock()
				if logg.errors != 1 {
					t.Errorf("internal error must be logged once, got %d", logg.errors)
				}
			}
		})
	}
}

func TestUpdateAndDeletePassID(t *testing.T) {
	fa := &fakeApp{}
	client, _ := newTestClient(t, fa)
	ctx := context.Background()

	if _, err := client.UpdateEvent(ctx, &pb.UpdateEventRequest{Id: testID.String(), Event: validPBEvent()}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if c := fa.lastCall(); c.name != "UpdateEvent" || c.id != testID || c.event.Title != "standup" {
		t.Errorf("update: called=%s id=%s title=%q", c.name, c.id, c.event.Title)
	}

	if _, err := client.DeleteEvent(ctx, &pb.DeleteEventRequest{Id: testID.String()}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if c := fa.lastCall(); c.name != "DeleteEvent" || c.id != testID {
		t.Errorf("delete: called=%s id=%s", c.name, c.id)
	}
}

func TestListMethods(t *testing.T) {
	date := timestamppb.New(testStart)
	tests := []struct {
		wantCall string
		call     func(c pb.EventServiceClient) ([]*pb.Event, error)
	}{
		{"ListDay", func(c pb.EventServiceClient) ([]*pb.Event, error) {
			r, err := c.ListDay(context.Background(), &pb.ListDayRequest{Date: date})
			return r.GetEvents(), err
		}},
		{"ListWeek", func(c pb.EventServiceClient) ([]*pb.Event, error) {
			r, err := c.ListWeek(context.Background(), &pb.ListWeekRequest{Start: date})
			return r.GetEvents(), err
		}},
		{"ListMonth", func(c pb.EventServiceClient) ([]*pb.Event, error) {
			r, err := c.ListMonth(context.Background(), &pb.ListMonthRequest{Month: date})
			return r.GetEvents(), err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.wantCall, func(t *testing.T) {
			fa := &fakeApp{events: []storage.Event{{
				ID:           testID,
				Title:        "standup",
				StartAt:      testStart,
				EndAt:        testStart.Add(time.Hour),
				UserID:       7,
				NotifyBefore: 15 * time.Minute,
			}}}
			client, _ := newTestClient(t, fa)

			events, err := tt.call(client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c := fa.lastCall(); c.name != tt.wantCall || !c.date.Equal(testStart) {
				t.Errorf("called %s with %v, want %s with %v", c.name, c.date, tt.wantCall, testStart)
			}
			if len(events) != 1 {
				t.Fatalf("got %d events, want 1", len(events))
			}

			e := events[0]
			if e.GetId() != testID.String() || e.GetTitle() != "standup" || e.GetUserId() != 7 {
				t.Errorf("unexpected event: %v", e)
			}
			if !e.GetStartAt().AsTime().Equal(testStart) || e.GetNotifyBefore().AsDuration() != 15*time.Minute {
				t.Errorf("wrong time fields: %v", e)
			}
		})
	}
}

func TestLoggingInterceptor(t *testing.T) {
	client, logg := newTestClient(t, &fakeApp{err: storage.ErrEventNotFound})

	_, _ = client.DeleteEvent(context.Background(), &pb.DeleteEventRequest{Id: testID.String()})

	logg.mu.Lock()
	defer logg.mu.Unlock()
	if len(logg.entries) != 1 {
		t.Fatalf("got %d log entries, want 1", len(logg.entries))
	}
	entry := logg.entries[0]
	if entry["method"] != "/calendar.v1.EventService/DeleteEvent" {
		t.Errorf("method = %v", entry["method"])
	}
	if entry["status"] != codes.NotFound.String() {
		t.Errorf("status = %v, want %s", entry["status"], codes.NotFound)
	}
	if _, ok := entry["duration"]; !ok {
		t.Error("duration is not logged")
	}
}
