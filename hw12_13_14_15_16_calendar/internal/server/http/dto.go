package internalhttp

import (
	"errors"
	"fmt"
	"time"

	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

var errBadRequest = errors.New("bad request")

type eventDTO struct {
	ID              string    `json:"id,omitempty"`
	Title           string    `json:"title"`
	StartAt         time.Time `json:"startAt"`
	EndAt           time.Time `json:"endAt"`
	Description     string    `json:"description,omitempty"`
	UserID          int64     `json:"userId"`
	NotifyBeforeSec int64     `json:"notifyBeforeSec,omitempty"`
}

type createEventResponse struct {
	ID string `json:"id"`
}

type listEventsResponse struct {
	Events []eventDTO `json:"events"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (d eventDTO) toEvent() (storage.Event, error) {
	if d.StartAt.IsZero() {
		return storage.Event{}, fmt.Errorf("%w: startAt is required", errBadRequest)
	}
	if d.EndAt.IsZero() {
		return storage.Event{}, fmt.Errorf("%w: endAt is required", errBadRequest)
	}
	return storage.Event{
		Title:        d.Title,
		StartAt:      d.StartAt,
		EndAt:        d.EndAt,
		Description:  d.Description,
		UserID:       d.UserID,
		NotifyBefore: time.Duration(d.NotifyBeforeSec) * time.Second,
	}, nil
}

func toDTO(e storage.Event) eventDTO {
	return eventDTO{
		ID:              e.ID.String(),
		Title:           e.Title,
		StartAt:         e.StartAt,
		EndAt:           e.EndAt,
		Description:     e.Description,
		UserID:          e.UserID,
		NotifyBeforeSec: int64(e.NotifyBefore / time.Second),
	}
}

func toDTOs(events []storage.Event) []eventDTO {
	dtos := make([]eventDTO, len(events))
	for i, e := range events {
		dtos[i] = toDTO(e)
	}
	return dtos
}
