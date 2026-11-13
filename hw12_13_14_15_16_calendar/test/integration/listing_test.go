//go:build integration

package integration

import (
	"net/http"
	"testing"
	"time"
)

const listingYear = 2031

func at(month time.Month, day, hour int) time.Time {
	return time.Date(listingYear, month, day, hour, 0, 0, 0, time.UTC)
}

func TestListDayWeekMonth(t *testing.T) {
	user := newUserID()
	create := func(title string, start time.Time) string {
		return createEvent(t, event{Title: title, StartAt: start, EndAt: start.Add(time.Hour), UserID: user})
	}

	prevMonth := create("feb 28", at(time.February, 28, 10))
	sunday := create("sun mar 2", at(time.March, 2, 10))
	monday := create("mon mar 3", at(time.March, 3, 10))
	mondayEvening := create("mon mar 3 evening", at(time.March, 3, 20))
	thursday := create("thu mar 6", at(time.March, 6, 10))
	nextMonday := create("mon mar 10", at(time.March, 10, 10))
	endOfMonth := create("fri mar 28", at(time.March, 28, 10))
	nextMonth := create("wed apr 2", at(time.April, 2, 10))
	all := []string{prevMonth, sunday, monday, mondayEvening, thursday, nextMonday, endOfMonth, nextMonth}

	tests := []struct {
		name   string
		period string
		date   string
		want   []string
	}{
		{"day", "day", "2031-03-03", []string{monday, mondayEvening}},
		{"week from monday", "week", "2031-03-03", []string{monday, mondayEvening, thursday}},
		{
			"month by any day", "month", "2031-03-17",
			[]string{sunday, monday, mondayEvening, thursday, nextMonday, endOfMonth},
		},
		{"empty day", "day", "2031-03-04", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listed := listEvents(t, tt.period, tt.date)

			want := make(map[string]bool, len(tt.want))
			for _, id := range tt.want {
				want[id] = true
			}
			for _, id := range all {
				_, got := listed[id]
				if got != want[id] {
					t.Errorf("event %q: listed = %v, want %v", eventTitle(listed, id), got, want[id])
				}
			}
		})
	}
}

func eventTitle(listed map[string]event, id string) string {
	if e, ok := listed[id]; ok {
		return e.Title
	}
	return id
}

func TestListBadDate(t *testing.T) {
	for _, path := range []string{"/events/day", "/events/week?date=03.03.2031", "/events/month?date=2031-13-01"} {
		if resp := call(t, http.MethodGet, path, nil); resp.status != http.StatusBadRequest {
			t.Errorf("%s: status %d, want %d", path, resp.status, http.StatusBadRequest)
		}
	}
}
