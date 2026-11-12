package logger

import (
	"bytes"
	"testing"
)

func TestLevelFiltering(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		call    func(*Logger, string, ...any)
		visible bool
	}{
		{"debug shown at debug", "debug", (*Logger).Debug, true},
		{"debug hidden at info", "info", (*Logger).Debug, false},
		{"info shown at info", "info", (*Logger).Info, true},
		{"info hidden at warn", "warn", (*Logger).Info, false},
		{"warn shown at warn", "warn", (*Logger).Warn, true},
		{"warn hidden at error", "error", (*Logger).Warn, false},
		{"error shown at error", "error", (*Logger).Error, true},
		{"debug shown at DEBUG", "DEBUG", (*Logger).Debug, true},
		{"debug hidden at INFO", "INFO", (*Logger).Debug, false},
		{"info shown at INFO", "INFO", (*Logger).Info, true},
		{"info hidden at WARN", "WARN", (*Logger).Info, false},
		{"warn shown at WARN", "WARN", (*Logger).Warn, true},
		{"warn hidden at ERROR", "ERROR", (*Logger).Warn, false},
		{"error shown at ERROR", "ERROR", (*Logger).Error, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l, err := New(tt.level, &buf)
			if err != nil {
				t.Fatal(err)
			}

			tt.call(l, "message")

			if got := buf.Len() > 0; got != tt.visible {
				t.Errorf("visible = %v, want %v, output: %q", got, tt.visible, buf.String())
			}
		})
	}
}

func TestNewUnknownLevel(t *testing.T) {
	if _, err := New("unknown", &bytes.Buffer{}); err == nil {
		t.Fatal("expected error for unknown log level, got nil")
	}
}
