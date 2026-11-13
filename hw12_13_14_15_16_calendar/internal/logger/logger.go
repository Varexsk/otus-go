package logger

import (
	"fmt"
	"io"
	"log/slog"
)

type Logger struct {
	log *slog.Logger
}

func New(level string, w io.Writer) (*Logger, error) {
	var lvl slog.Level

	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}
	logger := slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: lvl}))
	return &Logger{
		log: logger,
	}, nil
}

func (l Logger) Info(msg string, args ...any) {
	l.log.Info(msg, args...)
}

func (l Logger) Error(msg string, args ...any) {
	l.log.Error(msg, args...)
}

func (l Logger) Debug(msg string, args ...any) {
	l.log.Debug(msg, args...)
}

func (l Logger) Warn(msg string, args ...any) {
	l.log.Warn(msg, args...)
}
