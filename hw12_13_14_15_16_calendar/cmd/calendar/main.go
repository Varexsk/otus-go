package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/varexsk/hw12_13_14_15_calendar/internal/app"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/config"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/logger"
	internalgrpc "github.com/varexsk/hw12_13_14_15_calendar/internal/server/grpc"
	internalhttp "github.com/varexsk/hw12_13_14_15_calendar/internal/server/http"
	memorystorage "github.com/varexsk/hw12_13_14_15_calendar/internal/storage/memory"
	sqlstorage "github.com/varexsk/hw12_13_14_15_calendar/internal/storage/sql"
	"golang.org/x/sync/errgroup"
)

const connectTimeout = 5 * time.Second

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
	cfg, err := config.New(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var out io.Writer = os.Stdout
	if path := cfg.Logger.Output; path != "" {
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
	logg, err := logger.New(cfg.Logger.Level, out)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	var storage app.Storage
	switch cfg.Storage.Type {
	case config.SQL:
		st := sqlstorage.New(cfg.Storage.DSN)

		connectCtx, cancelConnect := context.WithTimeout(context.Background(), connectTimeout)
		err := st.Connect(connectCtx)
		cancelConnect()
		if err != nil {
			return fmt.Errorf("create SQL storage: %w", err)
		}
		defer func() {
			if err := st.Close(context.Background()); err != nil {
				logg.Error("close SQL storage", "error", err)
			}
		}()

		storage = st
	case config.Memory:
		storage = memorystorage.New()
	}

	calendar := app.New(logg, storage)

	httpServer := internalhttp.New(logg, calendar, cfg.HTTPServer.Address())
	grpcServer := internalgrpc.New(logg, calendar, cfg.GRPCServer.Address())

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := httpServer.Start(ctx); err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		if err := grpcServer.Start(ctx); err != nil {
			return fmt.Errorf("grpc server: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()

		stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer stopCancel()

		return errors.Join(
			httpServer.Stop(stopCtx),
			grpcServer.Stop(stopCtx),
		)
	})

	logg.Info("calendar is running...")

	if err := g.Wait(); err != nil {
		return err
	}

	logg.Info("calendar stopped")
	return nil
}
