package rmq

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

const urlEnv = "CALENDAR_TEST_RMQ_URL"

const waitTimeout = 5 * time.Second

func testConfig(t *testing.T) Config {
	t.Helper()
	url := os.Getenv(urlEnv)
	if url == "" {
		t.Skipf("%s is not set, skipping RabbitMQ tests", urlEnv)
	}

	suffix := uuid.NewString()
	cfg := Config{
		URL:        url,
		Exchange:   "test-exchange-" + suffix,
		Queue:      "test-queue-" + suffix,
		RoutingKey: "test",
	}

	t.Cleanup(func() {
		c := New(cfg)
		if err := c.Connect(context.Background()); err != nil {
			return
		}
		defer c.Close()
		_, _ = c.ch.QueueDelete(cfg.Queue, false, false, false)
		_ = c.ch.ExchangeDelete(cfg.Exchange, false, false)
	})
	return cfg
}

func connect(t *testing.T, cfg Config) *Client {
	t.Helper()
	c := New(cfg)
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func readyMessages(t *testing.T, c *Client, queue string) int {
	t.Helper()
	q, err := c.ch.QueueDeclarePassive(queue, true, false, false, false, nil)
	if err != nil {
		t.Fatalf("inspect queue: %v", err)
	}
	return q.Messages
}

func consumeAsync(c *Client, handle HandlerFunc) (stop func() error) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Consume(ctx, handle) }()

	return func() error {
		cancel()
		select {
		case err := <-done:
			return err
		case <-time.After(waitTimeout):
			return errors.New("consume did not stop after context cancel")
		}
	}
}

func TestNotConnected(t *testing.T) {
	c := New(Config{})

	if err := c.Publish(context.Background(), []byte("x")); !errors.Is(err, ErrNotConnected) {
		t.Errorf("Publish: err = %v, want %v", err, ErrNotConnected)
	}
	err := c.Consume(context.Background(), func(context.Context, []byte) error { return nil })
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("Consume: err = %v, want %v", err, ErrNotConnected)
	}
	if err := c.Close(); err != nil {
		t.Errorf("Close of never connected client: %v", err)
	}
}

func TestConnectUnreachable(t *testing.T) {
	c := New(Config{URL: "amqp://guest:guest@127.0.0.1:1/"})
	if err := c.Connect(context.Background()); err == nil {
		_ = c.Close()
		t.Fatal("expected error for unreachable broker")
	}
}

func TestConnectIsIdempotent(t *testing.T) {
	cfg := testConfig(t)
	connect(t, cfg)
	connect(t, cfg)
}

func TestPublishConsume(t *testing.T) {
	cfg := testConfig(t)
	publisher := connect(t, cfg)
	consumer := connect(t, cfg)

	want := []byte(`{"title":"standup"}`)
	if err := publisher.Publish(context.Background(), want); err != nil {
		t.Fatalf("publish: %v", err)
	}

	got := make(chan []byte, 1)
	stop := consumeAsync(consumer, func(_ context.Context, body []byte) error {
		got <- body
		return nil
	})

	select {
	case body := <-got:
		if string(body) != string(want) {
			t.Errorf("body = %s, want %s", body, want)
		}
	case <-time.After(waitTimeout):
		t.Fatal("message was not delivered")
	}

	if err := stop(); err != nil {
		t.Fatalf("consume returned error: %v", err)
	}

	if n := readyMessages(t, publisher, cfg.Queue); n != 0 {
		t.Errorf("queue has %d messages after ack, want 0", n)
	}
}

func TestMessageSurvivesWithoutConsumer(t *testing.T) {
	cfg := testConfig(t)
	c := connect(t, cfg)

	for range 3 {
		if err := c.Publish(context.Background(), []byte("x")); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}
	if n := readyMessages(t, c, cfg.Queue); n != 3 {
		t.Errorf("queue has %d messages, want 3", n)
	}
}

func TestFailedMessageIsNotRequeued(t *testing.T) {
	cfg := testConfig(t)
	publisher := connect(t, cfg)
	consumer := New(cfg)
	if err := consumer.Connect(context.Background()); err != nil {
		t.Fatalf("connect: %v", err)
	}

	if err := publisher.Publish(context.Background(), []byte("broken")); err != nil {
		t.Fatalf("publish: %v", err)
	}

	var calls atomic.Int32
	handled := make(chan struct{}, 1)
	stop := consumeAsync(consumer, func(context.Context, []byte) error {
		calls.Add(1)
		select {
		case handled <- struct{}{}:
		default:
		}
		return errors.New("cannot handle")
	})

	select {
	case <-handled:
	case <-time.After(waitTimeout):
		t.Fatal("message was not delivered")
	}
	time.Sleep(200 * time.Millisecond)

	if err := stop(); err != nil {
		t.Fatalf("consume returned error: %v", err)
	}
	
	if err := consumer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if n := calls.Load(); n != 1 {
		t.Errorf("handle called %d times, want exactly 1 (no redelivery)", n)
	}
	if n := readyMessages(t, publisher, cfg.Queue); n != 0 {
		t.Errorf("failed message returned to queue: %d messages, want 0", n)
	}
}

func TestConsumeStopsOnContextCancel(t *testing.T) {
	cfg := testConfig(t)
	c := connect(t, cfg)

	stop := consumeAsync(c, func(context.Context, []byte) error { return nil })
	if err := stop(); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}
