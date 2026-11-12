package notification

import (
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	EventID uuid.UUID `json:"eventId"`
	Title   string    `json:"title"`
	StartAt time.Time `json:"startAt"`
	UserID  int64     `json:"userId"`
}
