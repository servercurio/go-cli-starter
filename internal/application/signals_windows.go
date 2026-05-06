//go:build windows

package application

import (
	"os"
	"syscall"
)

// shutdownSignals lists the OS signals that cause Application.Start to return.
// On Windows only os.Interrupt (Ctrl+C / Ctrl+Break) is delivered to user code;
// SIGTERM is included for symmetry with the Unix list but the kernel never
// delivers it. SIGUSR1 has no Windows equivalent and is omitted.
var shutdownSignals = []os.Signal{
	os.Interrupt,
	os.Kill,
	syscall.SIGTERM,
}
