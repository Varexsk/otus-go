// Package config загружает и проверяет конфигурацию всех сервисов календаря.
package config

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	ErrLogLevelRequired   = errors.New("log level required")
	ErrInvalidStorageType = errors.New("invalid storage type")
	ErrDSNRequired        = errors.New("dsn required")
	ErrHostRequired       = errors.New("host required")
	ErrInvalidPort        = errors.New("invalid port")
	ErrReadConfig         = errors.New("read config")
	ErrRabbitRequired     = errors.New("rabbit setting required")
	ErrInvalidDuration    = errors.New("duration must be positive")
)

type StorageType string

const (
	SQL    StorageType = "sql"
	Memory StorageType = "memory"
)

type Calendar struct {
	Logger     LoggerConf
	HTTPServer ServerConf `mapstructure:"http"`
	GRPCServer ServerConf `mapstructure:"grpc"`
	Storage    StorageConf
}

type Scheduler struct {
	Logger    LoggerConf
	Database  DatabaseConf
	Rabbit    RabbitConf
	Scheduler SchedulerConf
}

type Sender struct {
	Logger LoggerConf
	Rabbit RabbitConf
}

type LoggerConf struct {
	Level  string
	Output string
}

type ServerConf struct {
	Host string
	Port int
}

type StorageConf struct {
	Type StorageType
	DSN  string
}

type DatabaseConf struct {
	DSN string
}

type RabbitConf struct {
	URL        string
	Exchange   string
	Queue      string
	RoutingKey string `mapstructure:"routing_key"`
}

type SchedulerConf struct {
	Interval  time.Duration
	Retention time.Duration
}

func NewCalendar(path string) (Calendar, error) {
	return load[Calendar](path)
}

func NewScheduler(path string) (Scheduler, error) {
	return load[Scheduler](path)
}

func NewSender(path string) (Sender, error) {
	return load[Sender](path)
}

func load[T interface{ validate() error }](path string) (T, error) {
	var zero T

	v := viper.New()
	v.SetConfigFile(path)

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return zero, fmt.Errorf("%w: %w", ErrReadConfig, err)
	}

	var cfg T
	if err := v.Unmarshal(&cfg); err != nil {
		return zero, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return zero, err
	}
	return cfg, nil
}

func (c Calendar) validate() error {
	if err := c.Logger.validate(); err != nil {
		return err
	}
	if err := c.GRPCServer.validate("grpc"); err != nil {
		return err
	}
	if err := c.HTTPServer.validate("http"); err != nil {
		return err
	}
	if !slices.Contains([]StorageType{SQL, Memory}, c.Storage.Type) {
		return fmt.Errorf("%w: %q", ErrInvalidStorageType, c.Storage.Type)
	}
	if c.Storage.Type == SQL && c.Storage.DSN == "" {
		return ErrDSNRequired
	}
	return nil
}

func (c Scheduler) validate() error {
	if err := c.Logger.validate(); err != nil {
		return err
	}
	if c.Database.DSN == "" {
		return ErrDSNRequired
	}
	if err := c.Rabbit.validate(); err != nil {
		return err
	}
	if c.Scheduler.Interval <= 0 {
		return fmt.Errorf("scheduler interval: %w", ErrInvalidDuration)
	}
	if c.Scheduler.Retention <= 0 {
		return fmt.Errorf("scheduler retention: %w", ErrInvalidDuration)
	}
	return nil
}

func (c Sender) validate() error {
	if err := c.Logger.validate(); err != nil {
		return err
	}
	return c.Rabbit.validate()
}

func (c LoggerConf) validate() error {
	if c.Level == "" {
		return ErrLogLevelRequired
	}
	return nil
}

func (c RabbitConf) validate() error {
	fields := []struct{ name, value string }{
		{"url", c.URL},
		{"exchange", c.Exchange},
		{"queue", c.Queue},
		{"routing_key", c.RoutingKey},
	}
	for _, f := range fields {
		if f.value == "" {
			return fmt.Errorf("%w: %s", ErrRabbitRequired, f.name)
		}
	}
	return nil
}

func (c ServerConf) validate(name string) error {
	if c.Host == "" {
		return fmt.Errorf("%s server: %w", name, ErrHostRequired)
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("%s server: %w", name, ErrInvalidPort)
	}
	return nil
}

func (c ServerConf) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}
