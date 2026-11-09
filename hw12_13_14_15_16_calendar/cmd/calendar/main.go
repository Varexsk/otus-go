package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/varexsk/hw12_13_14_15_calendar/internal/app"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/logger"
	internalhttp "github.com/varexsk/hw12_13_14_15_calendar/internal/server/http"
	memorystorage "github.com/varexsk/hw12_13_14_15_calendar/internal/storage/memory"
	sqlstorage "github.com/varexsk/hw12_13_14_15_calendar/internal/storage/sql"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "/etc/calendar/config.yaml", "Path to configuration file")
}

func main() {
	flag.Parse()

	if flag.Arg(0) == "version" {
		printVersion()
		return
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	config, err := NewConfig(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var out io.Writer = os.Stdout
	if path := config.Logger.Output; path != "" {
		err := os.MkdirAll(filepath.Dir(path), 0o755)
		if err != nil {
			return fmt.Errorf("create log folder: %w", err)
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		defer f.Close()
		out = f
	}
	logg, err := logger.New(config.Logger.Level, out)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	var storage app.Storage
	switch config.Storage.Type {
	case SQL:
		storage, err = sqlstorage.New(config.Storage.DSN)
		if err != nil {
			return fmt.Errorf("create SQL storage: %w", err)
		}
	case Memory:
		storage = memorystorage.New()
	}

	calendar := app.New(logg, storage)

	server := internalhttp.NewServer(logg, calendar, config.ServerAddress())

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			logg.Error("stop http server: " + err.Error())
		}
	}()

	logg.Info("calendar is running...")

	if err := server.Start(ctx); err != nil {
		cancel()
		return fmt.Errorf("start http server: %w", err)
	}

	<-stopped
	logg.Info("calendar stopped")

	return nil
}
