//go:build unix

// This test is Unix-only by design. It exercises graceful shutdown by sending
// the running process a signal that application.Start handles, which on Unix
// can be done via os.Process.Signal. Windows does not support delivering
// arbitrary signals to a process via os.Process.Signal — the only signal it
// implements is os.Kill (mapped to TerminateProcess), which is uncatchable
// and would terminate the test before the shutdown path runs. Reproducing
// the same coverage on Windows would require either a Windows console-event
// shim (Ctrl+Break via GenerateConsoleCtrlEvent) or a programmatic shutdown
// trigger on the Application type; neither is in scope here.

package main

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/servercurio/go-echo-starter/internal/env"
	asrt "github.com/stretchr/testify/assert"
)

const envPrefix = "APP"

// shutdownDelay is how long the test waits before signalling main() to exit.
// It must be long enough for Configure → Initialize → Start to reach the
// signal-wait in application.Start, but short enough to keep the test fast.
const shutdownDelay = 2 * time.Second

func TestNoPanicsInMain(t *testing.T) {
	assert := asrt.New(t)
	// Suppress the log output to avoid cluttering the test output
	assert.NoError(os.Setenv(env.AddPrefix(envPrefix, "daemon_log_enabled"), "false"), "failed to set APP_DAEMON_LOG_ENABLED")
	assert.NoError(os.Setenv(env.AddPrefix(envPrefix, "http_access_log_enabled"), "false"), "failed to set APP_HTTP_ACCESS_LOG_ENABLED")

	// main() blocks inside application.Start until one of the configured
	// shutdown signals arrives. Send SIGUSR1 to ourselves after a short delay
	// so main() returns and the test completes without an external signal.
	proc, err := os.FindProcess(os.Getpid())
	assert.NoError(err, "failed to find current process")

	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(shutdownDelay)
		_ = proc.Signal(syscall.SIGUSR1)
	}()

	assert.NotPanics(main, "main() should not panic")
	<-done
}
