package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Environment map[string]EnvValue

// EnvValue helps to distinguish between empty files and files with the first empty line.
type EnvValue struct {
	Value      string
	NeedRemove bool
}

// ReadDir reads a specified directory and returns map of env variables.
// Variables represented as files where filename is name of variable, file first line is a value.
// Nested directories and files whose name contains "=" are skipped: they can not name a variable.
func ReadDir(dir string) (Environment, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read env dir: %w", err)
	}

	env := make(Environment, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.Contains(name, "=") {
			continue
		}

		value, err := readEnvValue(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read env file %q: %w", name, err)
		}
		env[name] = value
	}

	return env, nil
}

// readEnvValue builds a variable value from the first line of the file.
// An empty file means that the variable has to be removed from the environment.
func readEnvValue(path string) (EnvValue, error) {
	file, err := os.Open(path)
	if err != nil {
		return EnvValue{}, err
	}
	defer file.Close()

	line, err := bufio.NewReader(file).ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return EnvValue{}, err
	}
	if len(line) == 0 {
		return EnvValue{NeedRemove: true}, nil
	}

	line = bytes.TrimSuffix(line, []byte("\n"))
	line = bytes.TrimSuffix(line, []byte("\r"))
	line = bytes.ReplaceAll(line, []byte{0x00}, []byte("\n"))

	return EnvValue{Value: strings.TrimRight(string(line), " \t")}, nil
}
