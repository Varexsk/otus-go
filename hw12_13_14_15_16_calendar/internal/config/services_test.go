package config

import (
	"errors"
	"testing"
	"time"
)

const validScheduler = `
logger:
  level: "DEBUG"
database:
  dsn: "postgres://user:pass@localhost:5432/db"
rabbit:
  url: "amqp://user:pass@localhost:5672/"
  exchange: "calendar"
  queue: "notifications"
  routing_key: "notification"
scheduler:
  interval: "1m"
  retention: "8760h"
`

const validSender = `
logger:
  level: "INFO"
rabbit:
  url: "amqp://user:pass@localhost:5672/"
  exchange: "calendar"
  queue: "notifications"
  routing_key: "notification"
`

var validRabbit = RabbitConf{
	URL:        "amqp://user:pass@localhost:5672/",
	Exchange:   "calendar",
	Queue:      "notifications",
	RoutingKey: "notification",
}

func TestNewScheduler(t *testing.T) {
	cfg, err := NewScheduler(writeConfig(t, validScheduler))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := Scheduler{
		Logger:   LoggerConf{Level: "DEBUG"},
		Database: DatabaseConf{DSN: "postgres://user:pass@localhost:5432/db"},
		Rabbit:   validRabbit,
		Scheduler: SchedulerConf{
			Interval:  time.Minute,
			Retention: 365 * 24 * time.Hour,
		},
	}
	if cfg != want {
		t.Errorf("config mismatch:\n got: %+v\nwant: %+v", cfg, want)
	}
}

func TestNewSender(t *testing.T) {
	cfg, err := NewSender(writeConfig(t, validSender))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := Sender{Logger: LoggerConf{Level: "INFO"}, Rabbit: validRabbit}
	if cfg != want {
		t.Errorf("config mismatch:\n got: %+v\nwant: %+v", cfg, want)
	}
}

func TestValidateScheduler(t *testing.T) {
	valid := func() Scheduler {
		return Scheduler{
			Logger:    LoggerConf{Level: "INFO"},
			Database:  DatabaseConf{DSN: "postgres://x"},
			Rabbit:    validRabbit,
			Scheduler: SchedulerConf{Interval: time.Minute, Retention: time.Hour},
		}
	}

	tests := []struct {
		name    string
		modify  func(c *Scheduler)
		wantErr error
	}{
		{"valid", func(_ *Scheduler) {}, nil},
		{"empty log level", func(c *Scheduler) { c.Logger.Level = "" }, ErrLogLevelRequired},
		{"empty dsn", func(c *Scheduler) { c.Database.DSN = "" }, ErrDSNRequired},
		{"empty rabbit url", func(c *Scheduler) { c.Rabbit.URL = "" }, ErrRabbitRequired},
		{"empty routing key", func(c *Scheduler) { c.Rabbit.RoutingKey = "" }, ErrRabbitRequired},
		{"zero interval", func(c *Scheduler) { c.Scheduler.Interval = 0 }, ErrInvalidDuration},
		{"negative retention", func(c *Scheduler) { c.Scheduler.Retention = -time.Hour }, ErrInvalidDuration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid()
			tt.modify(&cfg)
			if err := cfg.validate(); !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSender(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Sender
		wantErr error
	}{
		{"valid", Sender{Logger: LoggerConf{Level: "INFO"}, Rabbit: validRabbit}, nil},
		{"empty log level", Sender{Rabbit: validRabbit}, ErrLogLevelRequired},
		{"no rabbit", Sender{Logger: LoggerConf{Level: "INFO"}}, ErrRabbitRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.validate(); !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
