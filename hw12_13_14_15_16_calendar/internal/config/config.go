package config

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"

	"github.com/spf13/viper"
)

var (
	ErrLogLevelRequired   = errors.New("log level required")
	ErrInvalidStorageType = errors.New("invalid storage type")
	ErrDSNRequired        = errors.New("dsn required")
	ErrHostRequired       = errors.New("host required")
	ErrInvalidPort        = errors.New("invalid port")
	ErrReadConfig         = errors.New("read config")
)

type StorageType string

const (
	SQL    StorageType = "sql"
	Memory StorageType = "memory"
)

type Config struct {
	Logger     LoggerConf
	HTTPServer ServerConf `mapstructure:"http"`
	GRPCServer ServerConf `mapstructure:"grpc"`
	Storage    StorageConf
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

func New(path string) (Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("%w: %w", ErrReadConfig, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.Logger.Level == "" {
		return ErrLogLevelRequired
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
