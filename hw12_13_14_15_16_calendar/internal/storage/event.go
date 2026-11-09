package storage

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID           uuid.UUID
	Title        string
	StartAt      time.Time
	EndAt        time.Time
	Body         string
	UserID       int64
	NotifyBefore time.Duration
}
