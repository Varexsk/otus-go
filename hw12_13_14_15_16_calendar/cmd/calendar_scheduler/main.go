package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/varexsk/hw12_13_14_15_calendar/internal/config"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/logger"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/rmq"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/scheduler"
	sqlstorage "github.com/varexsk/hw12_13_14_15_calendar/internal/storage/sql"
)

const connectTimeout = 5 * time.Second

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "/etc/calendar/scheduler_config.yaml", "Path to configuration file")
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.NewScheduler(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logg, closeLog, err := logger.Open(cfg.Logger.Level, cfg.Logger.Output)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer closeLog()

	connectCtx, cancelConnect := context.WithTimeout(context.Background(), connectTimeout)
	defer cancelConnect()

	storage := sqlstorage.New(cfg.Database.DSN)
	if err := storage.Connect(connectCtx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer func() {
		if err := storage.Close(context.Background()); err != nil {
			logg.Error("close database", "error", err)
		}
	}()

	queue := rmq.New(rmq.Config(cfg.Rabbit))
	if err := queue.Connect(connectCtx); err != nil {
		return fmt.Errorf("connect to rabbitmq: %w", err)
	}
	defer func() {
		if err := queue.Close(); err != nil {
			logg.Error("close rabbitmq", "error", err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	scheduler.New(logg, storage, queue, cfg.Scheduler.Interval, cfg.Scheduler.Retention).Run(ctx)
	return nil
}
