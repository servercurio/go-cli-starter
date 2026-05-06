package application

import (
	"os"

	"github.com/joomcode/errorx"
	"github.com/servercurio/go-echo-starter/internal/logging"
)

// resolveHomeDirectory looks up the current user's home directory and
// stores it on the Application for later use (config search paths, autocert
// cache fallback location). Falls back to "." with a warning when the
// lookup fails — a missing home dir shouldn't take down the daemon.
func (app *Application) resolveHomeDirectory() {
	var err error

	if app.userHomeDirectory, err = os.UserHomeDir(); err == nil {
		logging.Daemon.
			Trace().
			Str("path", app.userHomeDirectory).
			Msg("resolved user home directory")
	} else {
		wErr := errorx.ExternalError.Wrap(err, "failed to resolve user home directory")
		logging.Daemon.
			Warn().
			Err(wErr).
			Msg("failed to resolve user home directory")
		app.userHomeDirectory = "."
	}
}
