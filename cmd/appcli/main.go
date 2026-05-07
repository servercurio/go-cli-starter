package main

import (
	"os"

	"github.com/servercurio/go-cli-starter/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		// Cobra has already printed the error. Exit non-zero so callers
		// (shells, CI, supervisors) see the failure.
		os.Exit(1)
	}
}
