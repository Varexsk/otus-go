package sender

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/notification"
)

var ErrInvalidNotification = errors.New("invalid notification")

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type Consumer interface {
	Consume(ctx context.Context, handle func(ctx context.Context, body []byte) error) error
}

type Sender struct {
	logg     Logger
	consumer Consumer
}

func New(logg Logger, consumer Consumer) *Sender {
	return &Sender{logg: logg, consumer: consumer}
}

func (s *Sender) Run(ctx context.Context) error {
	s.logg.Info("sender started")
	if err := s.consumer.Consume(ctx, s.handle); err != nil {
		return fmt.Errorf("consume notifications: %w", err)
	}
	s.logg.Info("sender stopped")
	return nil
}

func (s *Sender) handle(_ context.Context, body []byte) error {
	var n notification.Notification
	if err := json.Unmarshal(body, &n); err != nil {
		s.logg.Error("drop notification: invalid json", "error", err, "body", string(body))
		return fmt.Errorf("%w: %w", ErrInvalidNotification, err)
	}
	if n.EventID == uuid.Nil {
		s.logg.Error("drop notification: no event id", "body", string(body))
		return fmt.Errorf("%w: no event id", ErrInvalidNotification)
	}

	s.logg.Info("notification",
		"eventId", n.EventID,
		"title", n.Title,
		"startAt", n.StartAt.Format(time.RFC3339),
		"userId", n.UserID,
	)
	return nil
}
