package sender

import (
	"context"
	"errors"
	"testing"
)

type logEntry struct {
	level string
	msg   string
	args  map[string]any
}

type fakeLogger struct {
	entries []logEntry
}

func (l *fakeLogger) add(level, msg string, args []any) {
	m := make(map[string]any)
	for i := 0; i+1 < len(args); i += 2 {
		key, _ := args[i].(string)
		m[key] = args[i+1]
	}
	l.entries = append(l.entries, logEntry{level: level, msg: msg, args: m})
}

func (l *fakeLogger) Info(msg string, args ...any)  { l.add("info", msg, args) }
func (l *fakeLogger) Error(msg string, args ...any) { l.add("error", msg, args) }

func (l *fakeLogger) find(msg string) (logEntry, bool) {
	for _, e := range l.entries {
		if e.msg == msg {
			return e, true
		}
	}
	return logEntry{}, false
}

type fakeConsumer struct {
	messages []string
	err      error
	results  []error
}

func (c *fakeConsumer) Consume(ctx context.Context, handle func(context.Context, []byte) error) error {
	for _, m := range c.messages {
		c.results = append(c.results, handle(ctx, []byte(m)))
	}
	return c.err
}

const validJSON = `{"eventId":"6b900a50-01ce-47e9-a734-582f72fa7488","title":"standup",` +
	`"startAt":"2026-11-13T12:10:00Z","userId":42}`

func TestSendsValidNotification(t *testing.T) {
	consumer := &fakeConsumer{messages: []string{validJSON}}
	logg := &fakeLogger{}

	if err := New(logg, consumer).Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}

	if consumer.results[0] != nil {
		t.Errorf("valid message must be acked, handler returned %v", consumer.results[0])
	}
	entry, ok := logg.find("notification")
	if !ok {
		t.Fatalf("notification was not logged, entries: %+v", logg.entries)
	}
	if entry.args["title"] != "standup" || entry.args["userId"] != int64(42) ||
		entry.args["startAt"] != "2026-11-13T12:10:00Z" {
		t.Errorf("wrong notification fields: %+v", entry.args)
	}
}

func TestDropsInvalidNotification(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"not json", "hello"},
		{"wrong field type", `{"eventId":"6b900a50-01ce-47e9-a734-582f72fa7488","userId":"x"}`},
		{"no event id", `{"title":"standup"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer := &fakeConsumer{messages: []string{tt.body, validJSON}}
			logg := &fakeLogger{}

			if err := New(logg, consumer).Run(context.Background()); err != nil {
				t.Fatalf("run: %v", err)
			}

			if !errors.Is(consumer.results[0], ErrInvalidNotification) {
				t.Errorf("invalid message: handler returned %v, want %v", consumer.results[0], ErrInvalidNotification)
			}
			if consumer.results[1] != nil {
				t.Errorf("valid message after invalid one must still be handled, got %v", consumer.results[1])
			}
			if e, ok := logg.find("notification"); !ok || e.level != "info" {
				t.Error("valid notification after invalid one was not logged")
			}
		})
	}
}

func TestRunReturnsConsumerError(t *testing.T) {
	errLost := errors.New("connection lost")
	err := New(&fakeLogger{}, &fakeConsumer{err: errLost}).Run(context.Background())
	if !errors.Is(err, errLost) {
		t.Fatalf("err = %v, want %v", err, errLost)
	}
}
