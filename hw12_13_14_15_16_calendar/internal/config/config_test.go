package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

const validConfig = `
logger:
  level: "INFO"
http:
  host: "localhost"
  port: 8080
grpc:
  host: "127.0.0.1"
  port: 50051
storage:
  type: "sql"
  dsn: "postgres://user:pass@localhost:5432/db"
`

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestNew(t *testing.T) {
	cfg, err := New(writeConfig(t, validConfig))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := Config{
		Logger:     LoggerConf{Level: "INFO"},
		HTTPServer: ServerConf{Host: "localhost", Port: 8080},
		GRPCServer: ServerConf{Host: "127.0.0.1", Port: 50051},
		Storage: StorageConf{
			Type: SQL,
			DSN:  "postgres://user:pass@localhost:5432/db",
		},
	}
	if cfg != want {
		t.Errorf("config mismatch:\n got: %+v\nwant: %+v", cfg, want)
	}
}

func TestNewErrors(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.yaml")

		_, err := New(path)
		if !errors.Is(err, ErrReadConfig) {
			t.Errorf("want ErrReadConfig, got %v", err)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("want fs.ErrNotExist in chain, got %v", err)
		}
	})

	t.Run("broken yaml", func(t *testing.T) {
		_, err := New(writeConfig(t, "http: [unclosed"))
		if !errors.Is(err, ErrReadConfig) {
			t.Errorf("want ErrReadConfig, got %v", err)
		}
	})

	t.Run("wrong field type", func(t *testing.T) {
		_, err := New(writeConfig(t, "http:\n  port: \"not-a-number\"\n"))
		if err == nil {
			t.Fatal("want parse error, got nil")
		}
		if errors.Is(err, ErrReadConfig) {
			t.Errorf("file was read fine, error must come from parsing: %v", err)
		}
	})

	t.Run("invalid values return zero config", func(t *testing.T) {
		cfg, err := New(writeConfig(t, "logger:\n  level: \"INFO\"\n"))
		if err == nil {
			t.Fatal("want validation error, got nil")
		}
		if cfg != (Config{}) {
			t.Errorf("want zero config on error, got %+v", cfg)
		}
	})
}

func TestValidate(t *testing.T) {
	valid := func() Config {
		return Config{
			Logger:     LoggerConf{Level: "INFO"},
			HTTPServer: ServerConf{Host: "localhost", Port: 8080},
			GRPCServer: ServerConf{Host: "localhost", Port: 50051},
			Storage:    StorageConf{Type: Memory},
		}
	}

	tests := []struct {
		name    string
		modify  func(c *Config)
		wantErr error
	}{
		{
			name:   "valid memory",
			modify: func(_ *Config) {},
		},
		{
			name:   "valid sql with dsn",
			modify: func(c *Config) { c.Storage = StorageConf{Type: SQL, DSN: "postgres://x"} },
		},
		{
			name:    "empty log level",
			modify:  func(c *Config) { c.Logger.Level = "" },
			wantErr: ErrLogLevelRequired,
		},
		{
			name:    "empty http host",
			modify:  func(c *Config) { c.HTTPServer.Host = "" },
			wantErr: ErrHostRequired,
		},
		{
			name:    "empty grpc host",
			modify:  func(c *Config) { c.GRPCServer.Host = "" },
			wantErr: ErrHostRequired,
		},
		{
			name:    "zero port",
			modify:  func(c *Config) { c.HTTPServer.Port = 0 },
			wantErr: ErrInvalidPort,
		},
		{
			name:    "negative port",
			modify:  func(c *Config) { c.GRPCServer.Port = -1 },
			wantErr: ErrInvalidPort,
		},
		{
			name:    "port too big",
			modify:  func(c *Config) { c.HTTPServer.Port = 65536 },
			wantErr: ErrInvalidPort,
		},
		{
			name:   "max port",
			modify: func(c *Config) { c.HTTPServer.Port = 65535 },
		},
		{
			name:    "unknown storage type",
			modify:  func(c *Config) { c.Storage.Type = "redis" },
			wantErr: ErrInvalidStorageType,
		},
		{
			name:    "sql without dsn",
			modify:  func(c *Config) { c.Storage = StorageConf{Type: SQL} },
			wantErr: ErrDSNRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid()
			tt.modify(&cfg)

			err := cfg.validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestServerConfAddress(t *testing.T) {
	tests := []struct {
		conf ServerConf
		want string
	}{
		{ServerConf{Host: "localhost", Port: 8080}, "localhost:8080"},
		{ServerConf{Host: "::1", Port: 50051}, "[::1]:50051"},
	}
	for _, tt := range tests {
		if got := tt.conf.Address(); got != tt.want {
			t.Errorf("Address() = %q, want %q", got, tt.want)
		}
	}
}
