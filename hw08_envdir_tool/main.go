package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// failureExitCode is returned when the utility itself fails, as the original envdir does.
const failureExitCode = 111

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <env-dir> <command> [args...]\n", filepath.Base(os.Args[0]))
		os.Exit(failureExitCode)
	}

	env, err := ReadDir(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "envdir: %v\n", err)
		os.Exit(failureExitCode)
	}

	os.Exit(RunCmd(os.Args[2:], env))
}
