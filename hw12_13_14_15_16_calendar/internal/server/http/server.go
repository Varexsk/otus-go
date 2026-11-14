package internalhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

const (
	readHeaderTimeout = 5 * time.Second
)

type Server struct {
	server *http.Server
	logg   Logger
	app    Application
}

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type Application interface {
	CreateEvent(ctx context.Context, event storage.Event) (uuid.UUID, error)
	UpdateEvent(ctx context.Context, id uuid.UUID, event storage.Event) error
	DeleteEvent(ctx context.Context, id uuid.UUID) error
	ListDay(ctx context.Context, date time.Time) ([]storage.Event, error)
	ListWeek(ctx context.Context, start time.Time) ([]storage.Event, error)
	ListMonth(ctx context.Context, month time.Time) ([]storage.Event, error)
}

func New(logger Logger, app Application, address string) *Server {
	s := &Server{logg: logger, app: app}
	s.server = &http.Server{
		Addr:              address,
		Handler:           s.routes(),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	return s
}

func (s *Server) Start(_ context.Context) error {
	s.logg.Info("starting server", "addr", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logg.Info("stopping server")
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	s.logg.Info("server stopped")
	return nil
}
