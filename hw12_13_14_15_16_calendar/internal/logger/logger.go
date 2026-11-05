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

func (l Logger) Info(msg string) {
	l.log.Info(msg)
}

func (l Logger) Error(msg string) {
	l.log.Error(msg)
}

func (l Logger) Debug(msg string) {
	l.log.Debug(msg)
}

func (l Logger) Warn(msg string) {
	l.log.Warn(msg)
}
