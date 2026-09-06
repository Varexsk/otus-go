package main

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// helperMarker prefixes the line printed by the helper process, so the output
// can be found among the messages of the testing framework.
const helperMarker = "HELPER:"

func TestRunCmd(t *testing.T) {
	t.Run("environment is passed to the command", func(t *testing.T) {
		prepareHelperEnv(t)

		out, code := runAndCaptureStdout(t, helperCmd(), Environment{
			"FOO":   {Value: "123"},
			"UNSET": {NeedRemove: true},
		})

		require.Equal(t, 0, code)
		require.Contains(t, out, helperMarker+"FOO=[123] BAR=[from original env] UNSET=[]")
	})

	t.Run("exit code of the command is returned", func(t *testing.T) {
		prepareHelperEnv(t)

		_, code := runAndCaptureStdout(t, helperCmd(), Environment{"EXIT_CODE": {Value: "42"}})

		require.Equal(t, 42, code)
	})

	t.Run("unknown command", func(t *testing.T) {
		code := RunCmd([]string{"definitely-not-an-existing-command"}, Environment{})

		require.Equal(t, failureExitCode, code)
	})

	t.Run("empty command", func(t *testing.T) {
		code := RunCmd(nil, Environment{})

		require.Equal(t, failureExitCode, code)
	})
}

// TestHelperProcess is not a real test: it is executed as a child process by TestRunCmd
// and reports the environment it has been started with.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		t.Skip("not a helper process run")
	}

	fmt.Printf("%sFOO=[%s] BAR=[%s] UNSET=[%s]\n",
		helperMarker, os.Getenv("FOO"), os.Getenv("BAR"), os.Getenv("UNSET"))

	code, err := strconv.Atoi(os.Getenv("EXIT_CODE"))
	if err != nil {
		code = 0
	}
	os.Exit(code)
}

// prepareHelperEnv fills the environment of the current process to check that
// RunCmd passes it to the command, replacing and removing the requested variables.
func prepareHelperEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Setenv("FOO", "should be replaced")
	t.Setenv("BAR", "from original env")
	t.Setenv("UNSET", "should be removed")
}

// helperCmd is a command that runs TestHelperProcess in a child process.
func helperCmd() []string {
	return []string{os.Args[0], "-test.run=^TestHelperProcess$"}
}

// runAndCaptureStdout redirects the standard output of the current process to a file
// to make sure that the command really writes to it.
func runAndCaptureStdout(t *testing.T, cmd []string, env Environment) (output string, returnCode int) {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	defer file.Close()

	original := os.Stdout
	os.Stdout = file
	returnCode = RunCmd(cmd, env)
	os.Stdout = original

	captured, err := os.ReadFile(file.Name())
	require.NoError(t, err)

	return string(captured), returnCode
}
