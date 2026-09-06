package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunCmd runs a command + arguments (cmd) with environment variables from env.
// Standard input/output/error of the current process are passed to the command,
// the returned code is the exit code of the command.
func RunCmd(cmd []string, env Environment) (returnCode int) {
	if len(cmd) == 0 {
		fmt.Fprintln(os.Stderr, "envdir: no command to run")
		return failureExitCode
	}

	command := exec.Command(cmd[0], cmd[1:]...) //nolint:gosec // running a user given command is the whole point.
	command.Env = buildEnv(env)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "envdir: %v\n", err)
		return failureExitCode
	}

	return 0
}

// buildEnv applies env to the environment of the current process.
func buildEnv(env Environment) []string {
	current := os.Environ()

	result := make([]string, 0, len(current)+len(env))
	for _, variable := range current {
		name, _, _ := strings.Cut(variable, "=")
		if _, overridden := env[name]; overridden {
			continue
		}
		result = append(result, variable)
	}

	for name, value := range env {
		if value.NeedRemove {
			continue
		}
		result = append(result, name+"="+value.Value)
	}

	return result
}
