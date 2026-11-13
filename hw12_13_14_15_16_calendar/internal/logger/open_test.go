package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "app.log")

	logg, closeFn, err := Open("INFO", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	logg.Info("hello from test")
	if err := closeFn(); err != nil {
		t.Fatalf("close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "hello from test") {
		t.Errorf("log file does not contain message: %q", data)
	}
}

func TestOpenToStdout(t *testing.T) {
	logg, closeFn, err := Open("INFO", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if logg == nil {
		t.Fatal("logger is nil")
	}
	if err := closeFn(); err != nil {
		t.Errorf("close of stdout logger must be no-op, got %v", err)
	}
}

func TestOpenInvalidLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	if _, _, err := Open("LOUD", path); err == nil {
		t.Fatal("expected error for invalid level")
	}
}
