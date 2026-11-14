package storage

import "errors"

var (
	ErrEventNotFound     = errors.New("event not found")
	ErrDateBusy          = errors.New("time slot is already booked")
	ErrEventAlreadyExist = errors.New("event already exist")
)
