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
	sqltorage "github.com/varexsk/hw12_13_14_15_calendar/internal/storage/sql"
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

	config, err := NewConfig(configFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config: "+err.Error())
		os.Exit(2)
	}

	var out io.Writer = os.Stdout
	if path := config.Logger.Output; path != "" {
		os.MkdirAll(filepath.Dir(path), 0o755)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to open log file: "+err.Error())
			os.Exit(2)
		}
		defer f.Close()
		out = f
	}
	logg, err := logger.New(config.Logger.Level, out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create logger: "+err.Error())
		os.Exit(2)
	}

	var storage app.Storage
	switch config.Storage.Type {
	case Sql:
		storage, err = sqltorage.New(config.Storage.DSN)
		if err != nil {
			logg.Error("failed to create SQL storage: " + err.Error())
			os.Exit(2)
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
			logg.Error("failed to stop http server: " + err.Error())
		}
	}()

	logg.Info("calendar is running...")

	if err := server.Start(ctx); err != nil {
		logg.Error("failed to start http server: " + err.Error())
		cancel()
		os.Exit(1) //nolint:gocritic
	}

	<-stopped
	logg.Info("calendar stopped")
}
