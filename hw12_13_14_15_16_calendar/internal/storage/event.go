package storage

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID               uuid.UUID
	Title            string
	Date             time.Time
	EndDate          time.Time
	Body             string
	UserID           uint
	NotificationTime time.Duration
}
