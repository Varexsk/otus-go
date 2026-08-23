package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, content []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("prepare source file: %v", err)
	}

	return path
}

func tempDestination(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "destination.txt")
}

func TestCopy(t *testing.T) {
	content := []byte("0123456789abcdefghij")
	size := int64(len(content))

	tests := []struct {
		name   string
		offset int64
		limit  int64
		want   string
	}{
		{name: "whole file", want: string(content)},
		{name: "limit less than size", limit: 5, want: "01234"},
		{name: "limit equals size", limit: size, want: string(content)},
		{name: "limit greater than size", limit: size * 10, want: string(content)},
		{name: "offset without limit", offset: 10, want: "abcdefghij"},
		{name: "offset with limit", offset: 4, limit: 6, want: "456789"},
		{name: "offset with limit beyond eof", offset: 15, limit: 100, want: "fghij"},
		{name: "offset equals size", offset: size, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			destination := tempDestination(t)

			if err := Copy(writeTempFile(t, content), destination, tc.offset, tc.limit); err != nil {
				t.Fatalf("Copy() unexpected error: %v", err)
			}

			got, err := os.ReadFile(destination)
			if err != nil {
				t.Fatalf("read destination: %v", err)
			}

			if string(got) != tc.want {
				t.Errorf("copied %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCopyKeepsSourceIntact(t *testing.T) {
	content := []byte("0123456789")
	source := writeTempFile(t, content)

	if err := Copy(source, tempDestination(t), 3, 4); err != nil {
		t.Fatalf("Copy() unexpected error: %v", err)
	}

	got, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}

	if string(got) != string(content) {
		t.Errorf("source became %q, want %q", got, content)
	}
}

func TestCopyErrors(t *testing.T) {
	t.Run("source does not exist", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing.txt")

		err := Copy(missing, tempDestination(t), 0, 0)
		if !errors.Is(err, ErrNotExist) {
			t.Errorf("got %v, want %v", err, ErrNotExist)
		}
	})

	t.Run("offset exceeds file size", func(t *testing.T) {
		source := writeTempFile(t, []byte("0123456789"))

		err := Copy(source, tempDestination(t), 100, 0)
		if !errors.Is(err, ErrOffsetExceedsFileSize) {
			t.Errorf("got %v, want %v", err, ErrOffsetExceedsFileSize)
		}
	})

	t.Run("directory as source", func(t *testing.T) {
		err := Copy(t.TempDir(), tempDestination(t), 0, 0)
		if !errors.Is(err, ErrUnsupportedFile) {
			t.Errorf("got %v, want %v", err, ErrUnsupportedFile)
		}
	})
}
