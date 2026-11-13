//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/notification"
)

const (
	statusTimeout = 30 * time.Second
	quietPeriod   = 4 * time.Second
)

func TestNotificationIsSentOnce(t *testing.T) {
	statuses := subscribeStatuses(t)
	now := time.Now().UTC().Truncate(time.Second)

	dueID := createEvent(t, event{
		Title: "integration: due", StartAt: now.Add(5 * time.Minute), EndAt: now.Add(35 * time.Minute),
		UserID: newUserID(), NotifyBeforeSec: 600,
	})
	laterID := createEvent(t, event{
		Title: "integration: later", StartAt: now.Add(2 * time.Hour), EndAt: now.Add(3 * time.Hour),
		UserID: newUserID(), NotifyBeforeSec: 600,
	})
	silentID := createEvent(t, event{
		Title: "integration: silent", StartAt: now.Add(time.Minute), EndAt: now.Add(time.Hour),
		UserID: newUserID(),
	})

	received := waitForStatus(t, statuses, dueID, statusTimeout)
	if received.Status != notification.StatusSent {
		t.Errorf("status = %q, want %q", received.Status, notification.StatusSent)
	}
	if received.SentAt.Before(now) {
		t.Errorf("sentAt = %v is before the event was created (%v)", received.SentAt, now)
	}

	counts := collectStatuses(statuses, quietPeriod)
	if counts[dueID] != 0 {
		t.Errorf("notification for due event was sent again %d time(s)", counts[dueID])
	}
	if counts[laterID] != 0 {
		t.Error("notification sent before its time")
	}
	if counts[silentID] != 0 {
		t.Error("notification sent for event without notify_before")
	}
}

func TestOldEventsAreDeleted(t *testing.T) {
	start := time.Now().UTC().AddDate(-2, 0, 0).Truncate(time.Hour)
	day := start.Format("2006-01-02")
	id := createEvent(t, event{
		Title: "integration: old", StartAt: start, EndAt: start.Add(time.Hour), UserID: newUserID(),
	})

	deadline := time.Now().Add(statusTimeout)
	for time.Now().Before(deadline) {
		if _, ok := listEvents(t, "day", day)[id]; !ok {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("event that ended 2 years ago was not deleted within %s", statusTimeout)
}

func waitForStatus(
	t *testing.T, statuses <-chan notification.Status, id string, timeout time.Duration,
) notification.Status {
	t.Helper()

	want := uuid.MustParse(id)
	deadline := time.After(timeout)
	for {
		select {
		case s := <-statuses:
			if s.EventID == want {
				return s
			}
		case <-deadline:
			t.Fatalf("no notification status for event %s within %s", id, timeout)
		}
	}
}

func collectStatuses(statuses <-chan notification.Status, period time.Duration) map[string]int {
	counts := make(map[string]int)
	deadline := time.After(period)
	for {
		select {
		case s := <-statuses:
			counts[s.EventID.String()]++
		case <-deadline:
			return counts
		}
	}
}
