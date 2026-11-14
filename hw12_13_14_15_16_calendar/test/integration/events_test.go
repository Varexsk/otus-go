//go:build integration

package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

var eventStart = time.Date(2031, 1, 15, 10, 0, 0, 0, time.UTC)

func TestCreateEvent(t *testing.T) {
	want := event{
		Title:           "integration: create",
		StartAt:         eventStart,
		EndAt:           eventStart.Add(time.Hour),
		Description:     "created by integration test",
		UserID:          newUserID(),
		NotifyBeforeSec: 900,
	}

	id := createEvent(t, want)

	got, ok := listEvents(t, "day", "2031-01-15")[id]
	if !ok {
		t.Fatalf("created event %s is not listed", id)
	}
	if got.Title != want.Title || got.Description != want.Description || got.UserID != want.UserID ||
		!got.StartAt.Equal(want.StartAt) || !got.EndAt.Equal(want.EndAt) ||
		got.NotifyBeforeSec != want.NotifyBeforeSec {
		t.Errorf("stored event = %+v, want %+v", got, want)
	}
}

func TestCreateEventBusinessErrors(t *testing.T) {
	user := newUserID()
	createEvent(t, event{Title: "occupied", StartAt: eventStart, EndAt: eventStart.Add(time.Hour), UserID: user})

	tests := []struct {
		name       string
		body       any
		wantStatus int
	}{
		{
			name:       "empty title",
			body:       event{StartAt: eventStart, EndAt: eventStart.Add(time.Hour), UserID: newUserID()},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ends before start",
			body:       event{Title: "x", StartAt: eventStart, EndAt: eventStart.Add(-time.Hour), UserID: newUserID()},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no start time",
			body:       `{"title":"x","endAt":"2031-01-15T11:00:00Z","userId":1}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "broken json",
			body:       `{"title":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "time is busy for the same user",
			body: event{
				Title: "overlap", StartAt: eventStart.Add(30 * time.Minute), EndAt: eventStart.Add(2 * time.Hour), UserID: user,
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "same time for another user is fine",
			body: event{
				Title: "other user", StartAt: eventStart, EndAt: eventStart.Add(time.Hour), UserID: newUserID(),
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := call(t, http.MethodPost, "/events", tt.body)
			if resp.status != tt.wantStatus {
				t.Errorf("status = %d, want %d, body %s", resp.status, tt.wantStatus, resp.body)
			}
		})
	}
}

func TestUpdateAndDeleteEvent(t *testing.T) {
	user := newUserID()
	start := eventStart.AddDate(0, 0, 1)
	id := createEvent(t, event{Title: "before update", StartAt: start, EndAt: start.Add(time.Hour), UserID: user})

	updated := event{Title: "after update", StartAt: start, EndAt: start.Add(2 * time.Hour), UserID: user}
	if resp := call(t, http.MethodPut, "/events/"+id, updated); resp.status != http.StatusNoContent {
		t.Fatalf("update: status %d, body %s", resp.status, resp.body)
	}
	got := listEvents(t, "day", "2031-01-16")[id]
	if got.Title != "after update" || !got.EndAt.Equal(updated.EndAt) {
		t.Errorf("event after update = %+v", got)
	}

	missing := uuid.NewString()
	checks := []struct {
		name       string
		method     string
		path       string
		body       any
		wantStatus int
	}{
		{"update unknown event", http.MethodPut, "/events/" + missing, updated, http.StatusNotFound},
		{"update with bad id", http.MethodPut, "/events/not-a-uuid", updated, http.StatusBadRequest},
		{"delete", http.MethodDelete, "/events/" + id, nil, http.StatusNoContent},
		{"delete again", http.MethodDelete, "/events/" + id, nil, http.StatusNotFound},
		{"delete with bad id", http.MethodDelete, "/events/123", nil, http.StatusBadRequest},
	}
	for _, c := range checks {
		if resp := call(t, c.method, c.path, c.body); resp.status != c.wantStatus {
			t.Errorf("%s: status %d, want %d, body %s", c.name, resp.status, c.wantStatus, resp.body)
		}
	}

	if _, ok := listEvents(t, "day", "2031-01-16")[id]; ok {
		t.Error("deleted event is still listed")
	}
}
