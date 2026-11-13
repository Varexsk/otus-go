package config

import (
	"testing"
	"time"
)

func TestEnvOverridesCalendar(t *testing.T) {
	t.Setenv("STORAGE_TYPE", "memory")
	t.Setenv("STORAGE_DSN", "postgres://from-env")
	t.Setenv("HTTP_PORT", "9999")

	cfg, err := NewCalendar(writeConfig(t, validConfig))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Storage.Type != Memory {
		t.Errorf("storage type = %q, want %q", cfg.Storage.Type, Memory)
	}
	if cfg.Storage.DSN != "postgres://from-env" {
		t.Errorf("dsn = %q, want value from env", cfg.Storage.DSN)
	}
	if cfg.HTTPServer.Port != 9999 {
		t.Errorf("http port = %d, want 9999", cfg.HTTPServer.Port)
	}
	if cfg.GRPCServer.Port != 50051 {
		t.Errorf("grpc port = %d, must stay from file", cfg.GRPCServer.Port)
	}
}

func TestEnvOverridesScheduler(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://db:5432/calendar")
	t.Setenv("RABBIT_URL", "amqp://rabbitmq:5672/")
	t.Setenv("RABBIT_ROUTING_KEY", "from-env")
	t.Setenv("SCHEDULER_INTERVAL", "2s")

	cfg, err := NewScheduler(writeConfig(t, validScheduler))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Database.DSN != "postgres://db:5432/calendar" {
		t.Errorf("dsn = %q, want value from env", cfg.Database.DSN)
	}
	if cfg.Rabbit.URL != "amqp://rabbitmq:5672/" || cfg.Rabbit.RoutingKey != "from-env" {
		t.Errorf("rabbit = %+v, want url and routing key from env", cfg.Rabbit)
	}
	if cfg.Scheduler.Interval != 2*time.Second {
		t.Errorf("interval = %v, want 2s", cfg.Scheduler.Interval)
	}
	if cfg.Rabbit.Queue != "notifications" {
		t.Errorf("queue = %q, must stay from file", cfg.Rabbit.Queue)
	}
}

func TestEnvInvalidValueFailsValidation(t *testing.T) {
	t.Setenv("STORAGE_TYPE", "redis")

	if _, err := NewCalendar(writeConfig(t, validConfig)); err == nil {
		t.Fatal("invalid value from env must fail validation")
	}
}
