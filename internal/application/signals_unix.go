//go:build unix

package application

import (
	"os"
	"syscall"
)

// shutdownSignals lists the OS signals that cause Application.Start to return.
// On Unix the daemon honours SIGINT, SIGKILL (uncatchable; included for symmetry),
// SIGTERM (the canonical shutdown signal from kill(1), Docker, and Kubernetes),
// and SIGUSR1 (an application-defined signal repurposed here for clean shutdown).
var shutdownSignals = []os.Signal{
	os.Interrupt,
	os.Kill,
	syscall.SIGTERM,
	syscall.SIGUSR1,
}
