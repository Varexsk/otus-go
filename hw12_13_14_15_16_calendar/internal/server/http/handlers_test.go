package internalhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/app"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

// fakeApp — управляемая подмена App: возвращает то, что задано в полях,
// и запоминает, с какими аргументами его вызвали.
type fakeApp struct {
	createID uuid.UUID
	events   []storage.Event
	err      error

	called   string
	gotID    uuid.UUID
	gotEvent storage.Event
	gotDate  time.Time
}

func (f *fakeApp) CreateEvent(_ context.Context, e storage.Event) (uuid.UUID, error) {
	f.called, f.gotEvent = "CreateEvent", e
	return f.createID, f.err
}

func (f *fakeApp) UpdateEvent(_ context.Context, id uuid.UUID, e storage.Event) error {
	f.called, f.gotID, f.gotEvent = "UpdateEvent", id, e
	return f.err
}

func (f *fakeApp) DeleteEvent(_ context.Context, id uuid.UUID) error {
	f.called, f.gotID = "DeleteEvent", id
	return f.err
}

func (f *fakeApp) ListDay(_ context.Context, d time.Time) ([]storage.Event, error) {
	f.called, f.gotDate = "ListDay", d
	return f.events, f.err
}

func (f *fakeApp) ListWeek(_ context.Context, d time.Time) ([]storage.Event, error) {
	f.called, f.gotDate = "ListWeek", d
	return f.events, f.err
}

func (f *fakeApp) ListMonth(_ context.Context, d time.Time) ([]storage.Event, error) {
	f.called, f.gotDate = "ListMonth", d
	return f.events, f.err
}

var (
	testID    = uuid.MustParse("6b900a50-01ce-47e9-a734-582f72fa7488")
	testStart = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	errDB     = errors.New("connection refused")
)

const validBody = `{"title":"standup","startAt":"2026-11-12T10:00:00Z",` +
	`"endAt":"2026-11-12T11:00:00Z","userId":7,"notifyBeforeSec":900}`

func doRequest(t *testing.T, fa *fakeApp, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	s := &Server{logg: &fakeLogger{}, app: fa}
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, httptest.NewRequest(method, target, strings.NewReader(body)))
	return rec
}

func TestHandleCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fa := &fakeApp{createID: testID}
		rec := doRequest(t, fa, http.MethodPost, "/events", validBody)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body)
		}
		var resp createEventResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.ID != testID.String() {
			t.Errorf("id = %q, want %q", resp.ID, testID)
		}

		want := storage.Event{
			Title:        "standup",
			StartAt:      testStart,
			EndAt:        testStart.Add(time.Hour),
			UserID:       7,
			NotifyBefore: 15 * time.Minute,
		}
		if fa.gotEvent != want {
			t.Errorf("app got event %+v, want %+v", fa.gotEvent, want)
		}
	})

	t.Run("content type is json", func(t *testing.T) {
		rec := doRequest(t, &fakeApp{}, http.MethodPost, "/events", validBody)
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
	})
}

// Ошибки, после которых запрос не должен дойти до App.
func TestHandleCreateBadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"broken json", `{oops`},
		{"no startAt", `{"title":"x","endAt":"2026-11-12T11:00:00Z"}`},
		{"no endAt", `{"title":"x","startAt":"2026-11-12T10:00:00Z"}`},
		{"wrong field type", `{"title":123}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fa := &fakeApp{}
			rec := doRequest(t, fa, http.MethodPost, "/events", tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			if fa.called != "" {
				t.Errorf("app must not be called, but %s was", fa.called)
			}
		})
	}
}

// Перевод ошибок App в HTTP-статусы.
func TestWriteErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		appErr   error
		wantCode int
		wantMsg  string
	}{
		{"invalid event", app.ErrInvalidEvent, http.StatusBadRequest, app.ErrInvalidEvent.Error()},
		{"wrapped invalid event", errors.Join(errDB, app.ErrInvalidEvent), http.StatusBadRequest, ""},
		{"not found", storage.ErrEventNotFound, http.StatusNotFound, storage.ErrEventNotFound.Error()},
		{"already exist", storage.ErrEventAlreadyExist, http.StatusConflict, storage.ErrEventAlreadyExist.Error()},
		{"date busy", storage.ErrDateBusy, http.StatusConflict, storage.ErrDateBusy.Error()},
		{"unknown error is hidden", errDB, http.StatusInternalServerError, "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, &fakeApp{err: tt.appErr}, http.MethodPost, "/events", validBody)

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			var resp errorResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("error body is not json: %v", err)
			}
			if tt.wantMsg != "" && resp.Error != tt.wantMsg {
				t.Errorf("error = %q, want %q", resp.Error, tt.wantMsg)
			}
			if strings.Contains(resp.Error, "connection refused") && tt.wantCode == http.StatusInternalServerError {
				t.Errorf("internal error details leaked to client: %q", resp.Error)
			}
		})
	}
}

func TestHandleUpdate(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		body     string
		appErr   error
		wantCode int
		wantCall bool
	}{
		{"success", "/events/" + testID.String(), validBody, nil, http.StatusNoContent, true},
		{"invalid id", "/events/not-a-uuid", validBody, nil, http.StatusBadRequest, false},
		{"invalid body", "/events/" + testID.String(), `{oops`, nil, http.StatusBadRequest, false},
		{"not found", "/events/" + testID.String(), validBody, storage.ErrEventNotFound, http.StatusNotFound, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fa := &fakeApp{err: tt.appErr}
			rec := doRequest(t, fa, http.MethodPut, tt.path, tt.body)

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d, body: %s", rec.Code, tt.wantCode, rec.Body)
			}
			if called := fa.called == "UpdateEvent"; called != tt.wantCall {
				t.Fatalf("UpdateEvent called = %v, want %v", called, tt.wantCall)
			}
			if tt.wantCall && fa.gotID != testID {
				t.Errorf("app got id %s, want %s", fa.gotID, testID)
			}
		})
	}
}

func TestHandleDelete(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		appErr   error
		wantCode int
	}{
		{"success", "/events/" + testID.String(), nil, http.StatusNoContent},
		{"invalid id", "/events/123", nil, http.StatusBadRequest},
		{"not found", "/events/" + testID.String(), storage.ErrEventNotFound, http.StatusNotFound},
		{"storage failure", "/events/" + testID.String(), errDB, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fa := &fakeApp{err: tt.appErr}
			rec := doRequest(t, fa, http.MethodDelete, tt.path, "")

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if rec.Code == http.StatusNoContent && rec.Body.Len() != 0 {
				t.Errorf("204 must have empty body, got %q", rec.Body)
			}
		})
	}
}

// Каждый маршрут должен вызывать СВОЙ метод App — защита от copy-paste ошибок.
func TestHandleListRouting(t *testing.T) {
	tests := []struct {
		path     string
		wantCall string
	}{
		{"/events/day?date=2026-11-12", "ListDay"},
		{"/events/week?date=2026-11-12", "ListWeek"},
		{"/events/month?date=2026-11-12", "ListMonth"},
	}

	for _, tt := range tests {
		t.Run(tt.wantCall, func(t *testing.T) {
			fa := &fakeApp{}
			rec := doRequest(t, fa, http.MethodGet, tt.path, "")

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if fa.called != tt.wantCall {
				t.Errorf("called %q, want %q", fa.called, tt.wantCall)
			}
			wantDate := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
			if !fa.gotDate.Equal(wantDate) {
				t.Errorf("app got date %v, want %v", fa.gotDate, wantDate)
			}
		})
	}
}

func TestHandleListResponse(t *testing.T) {
	t.Run("events are converted to json", func(t *testing.T) {
		fa := &fakeApp{events: []storage.Event{{
			ID:           testID,
			Title:        "standup",
			StartAt:      testStart,
			EndAt:        testStart.Add(time.Hour),
			UserID:       7,
			NotifyBefore: 15 * time.Minute,
		}}}
		rec := doRequest(t, fa, http.MethodGet, "/events/day?date=2026-11-12", "")

		var resp listEventsResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(resp.Events) != 1 {
			t.Fatalf("got %d events, want 1", len(resp.Events))
		}
		got := resp.Events[0]
		if got.ID != testID.String() || got.Title != "standup" || got.UserID != 7 {
			t.Errorf("unexpected event: %+v", got)
		}
		if !got.StartAt.Equal(testStart) || got.NotifyBeforeSec != 900 {
			t.Errorf("wrong time fields: startAt=%v notifyBeforeSec=%d", got.StartAt, got.NotifyBeforeSec)
		}
	})

	t.Run("empty list is [] not null", func(t *testing.T) {
		rec := doRequest(t, &fakeApp{}, http.MethodGet, "/events/day?date=2026-11-12", "")
		if body := strings.TrimSpace(rec.Body.String()); body != `{"events":[]}` {
			t.Errorf("body = %s, want {\"events\":[]}", body)
		}
	})
}

func TestHandleListBadDate(t *testing.T) {
	for _, path := range []string{
		"/events/day",                    // нет параметра
		"/events/day?date=12.11.2026",    // чужой формат
		"/events/week?date=2026-13-01",   // несуществующий месяц
		"/events/month?date=2026-11-12x", // мусор в конце
	} {
		t.Run(path, func(t *testing.T) {
			fa := &fakeApp{}
			rec := doRequest(t, fa, http.MethodGet, path, "")

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			if fa.called != "" {
				t.Errorf("app must not be called, but %s was", fa.called)
			}
		})
	}
}
