package notification

import (
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

type Notification struct {
	EventID uuid.UUID `json:"eventId"`
	Title   string    `json:"title"`
	StartAt time.Time `json:"startAt"`
	UserID  int64     `json:"userId"`
}

func FromEvent(e storage.Event) Notification {
	return Notification{
		EventID: e.ID,
		Title:   e.Title,
		StartAt: e.StartAt,
		UserID:  e.UserID,
	}
}
