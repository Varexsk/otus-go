package main

import (
	"fmt"
	"net"
	"slices"
	"strconv"

	"github.com/spf13/viper"
)

type StorageType string

const (
	Sql    StorageType = "sql"
	Memory StorageType = "memory"
)

type Config struct {
	Logger  LoggerConf
	Server  ServerConf
	Storage StorageConf
}

type LoggerConf struct {
	Level string
}

type ServerConf struct {
	Host string
	Port int
}

type StorageConf struct {
	Type StorageType
	DSN  string
}

func NewConfig(path string) (Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, cfg.validate()
}

func (c Config) validate() error {
	if c.Logger.Level == "" {
		return fmt.Errorf("logger level is required")
	}
	if c.Server.Host == "" {
		return fmt.Errorf("server host is required")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be a valid TCP port")
	}
	if !slices.Contains([]StorageType{Sql, Memory}, c.Storage.Type) {
		return fmt.Errorf("invalid storage type: %s", c.Storage.Type)
	}
	if c.Storage.Type == Sql && c.Storage.DSN == "" {
		return fmt.Errorf("storage DSN is required for sql")
	}
	return nil
}

func (c Config) ServerAddress() string {
	return net.JoinHostPort(c.Server.Host, strconv.Itoa(c.Server.Port))
}
