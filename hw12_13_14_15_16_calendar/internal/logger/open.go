package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Open(level, output string) (*Logger, func() error, error) {
	var (
		out     io.Writer = os.Stdout
		closeFn           = func() error { return nil }
	)

	if output != "" {
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return nil, nil, fmt.Errorf("create log folder: %w", err)
		}
		f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file: %w", err)
		}
		out, closeFn = f, f.Close
	}

	logg, err := New(level, out)
	if err != nil {
		_ = closeFn()
		return nil, nil, err
	}
	return logg, closeFn, nil
}
