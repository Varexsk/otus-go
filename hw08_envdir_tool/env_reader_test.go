package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadDirTestdata(t *testing.T) {
	env, err := ReadDir("testdata/env")
	require.NoError(t, err)

	require.Equal(t, Environment{
		"BAR":   {Value: "bar"},
		"EMPTY": {Value: ""},
		"FOO":   {Value: "   foo\nwith new line"},
		"HELLO": {Value: `"hello"`},
		"UNSET": {NeedRemove: true},
	}, env)
}

func TestReadDirValues(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected EnvValue
	}{
		{
			name:     "simple value",
			content:  "value",
			expected: EnvValue{Value: "value"},
		},
		{
			name:     "only the first line is used",
			content:  "first\nsecond\nthird",
			expected: EnvValue{Value: "first"},
		},
		{
			name:     "trailing spaces and tabs are trimmed",
			content:  "value \t \t\n",
			expected: EnvValue{Value: "value"},
		},
		{
			name:     "leading spaces are kept",
			content:  "   value",
			expected: EnvValue{Value: "   value"},
		},
		{
			name:     "terminal zeroes are replaced with new lines",
			content:  "one\x00two\x00three",
			expected: EnvValue{Value: "one\ntwo\nthree"},
		},
		{
			name:     "empty file means removal",
			content:  "",
			expected: EnvValue{NeedRemove: true},
		},
		{
			name:     "empty first line is an empty value",
			content:  "\nsecond",
			expected: EnvValue{Value: ""},
		},
		{
			name:     "file of spaces is an empty value",
			content:  "  \t  \n",
			expected: EnvValue{Value: ""},
		},
		{
			name:     "windows line ending is not a part of the value",
			content:  "value\r\nsecond",
			expected: EnvValue{Value: "value"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			createEnvFile(t, dir, "VAR", tc.content)

			env, err := ReadDir(dir)
			require.NoError(t, err)
			require.Equal(t, Environment{"VAR": tc.expected}, env)
		})
	}
}

func TestReadDirSkipsUnsuitableEntries(t *testing.T) {
	dir := t.TempDir()
	createEnvFile(t, dir, "VAR", "value")
	createEnvFile(t, dir, "IN=VALID", "value")
	require.NoError(t, os.Mkdir(filepath.Join(dir, "NESTED"), 0o750))

	env, err := ReadDir(dir)
	require.NoError(t, err)
	require.Equal(t, Environment{"VAR": {Value: "value"}}, env)
}

func TestReadDirEmptyDir(t *testing.T) {
	env, err := ReadDir(t.TempDir())
	require.NoError(t, err)
	require.Empty(t, env)
}

func TestReadDirNotExistingDir(t *testing.T) {
	env, err := ReadDir(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
	require.Nil(t, env)
}

func createEnvFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
}
