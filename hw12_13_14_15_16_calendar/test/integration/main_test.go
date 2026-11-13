//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/varexsk/hw12_13_14_15_calendar/internal/notification"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/rmq"
)

var (
	apiURL    = envOr("CALENDAR_API_URL", "http://calendar:8080")
	rabbitURL = envOr("RABBIT_URL", "amqp://rabbit:password@rabbitmq:5672/")
	client    = &http.Client{Timeout: 5 * time.Second}
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func TestMain(m *testing.M) {
	if err := waitForAPI(30 * time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func waitForAPI(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, apiURL+"/hello", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("calendar API at %s is not ready after %s", apiURL, timeout)
}

func newUserID() int64 {
	return rand.Int64N(1_000_000_000) + 1
}

type event struct {
	ID              string    `json:"id,omitempty"`
	Title           string    `json:"title"`
	StartAt         time.Time `json:"startAt"`
	EndAt           time.Time `json:"endAt"`
	Description     string    `json:"description,omitempty"`
	UserID          int64     `json:"userId"`
	NotifyBeforeSec int64     `json:"notifyBeforeSec,omitempty"`
}

type apiResponse struct {
	status int
	body   []byte
}

func call(t *testing.T, method, path string, body any) apiResponse {
	t.Helper()

	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = bytes.NewBufferString(b)
	default:
		data, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, apiURL+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return apiResponse{status: resp.StatusCode, body: data}
}

func createEvent(t *testing.T, e event) string {
	t.Helper()

	resp := call(t, http.MethodPost, "/events", e)
	if resp.status != http.StatusCreated {
		t.Fatalf("create event: status %d, body %s", resp.status, resp.body)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.body, &created); err != nil || created.ID == "" {
		t.Fatalf("create event: bad response %s: %v", resp.body, err)
	}
	return created.ID
}

func listEvents(t *testing.T, period, date string) map[string]event {
	t.Helper()

	resp := call(t, http.MethodGet, "/events/"+period+"?date="+date, nil)
	if resp.status != http.StatusOK {
		t.Fatalf("list %s %s: status %d, body %s", period, date, resp.status, resp.body)
	}
	var list struct {
		Events []event `json:"events"`
	}
	if err := json.Unmarshal(resp.body, &list); err != nil {
		t.Fatalf("list %s: bad response %s: %v", period, resp.body, err)
	}

	byID := make(map[string]event, len(list.Events))
	for _, e := range list.Events {
		byID[e.ID] = e
	}
	return byID
}

func subscribeStatuses(t *testing.T) <-chan notification.Status {
	t.Helper()

	c := rmq.New(rmq.Config{
		URL:        rabbitURL,
		Exchange:   "calendar",
		Queue:      "notification_status",
		RoutingKey: "notification_status",
	})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("connect to rabbitmq: %v", err)
	}

	statuses := make(chan notification.Status, 100)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = c.Consume(ctx, func(_ context.Context, body []byte) error {
			var s notification.Status
			if err := json.Unmarshal(body, &s); err != nil {
				return err
			}
			statuses <- s
			return nil
		})
	}()

	t.Cleanup(func() {
		cancel()
		<-done
		_ = c.Close()
	})
	return statuses
}
