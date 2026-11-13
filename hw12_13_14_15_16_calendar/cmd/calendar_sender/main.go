package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/varexsk/hw12_13_14_15_calendar/internal/config"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/logger"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/rmq"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/sender"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "/etc/calendar/sender_config.yaml", "Path to configuration file")
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.NewSender(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logg, closeLog, err := logger.Open(cfg.Logger.Level, cfg.Logger.Output)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer closeLog()

	queue := rmq.New(rmq.Config(cfg.Rabbit))
	if err := queue.Connect(context.Background()); err != nil {
		return fmt.Errorf("connect to rabbitmq: %w", err)
	}
	defer func() {
		if err := queue.Close(); err != nil {
			logg.Error("close rabbitmq", "error", err)
		}
	}()

	status := rmq.New(rmq.Config(cfg.Status))
	if err := status.Connect(context.Background()); err != nil {
		return fmt.Errorf("connect to rabbitmq (status): %w", err)
	}
	defer func() {
		if err := status.Close(); err != nil {
			logg.Error("close rabbitmq (status)", "error", err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	return sender.New(logg, queue, status).Run(ctx)
}
