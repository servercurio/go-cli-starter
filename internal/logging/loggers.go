package logging

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Daemon is the package-wide structured logger for daemon (control-plane)
// events. It's swapped in by Initialize and is safe to read concurrently
// once Initialize has returned.
var Daemon zerolog.Logger

// m guards Initialize so concurrent Notify* calls during startup don't race
// while replacing Daemon.
var m sync.Mutex

// Initialize the logging system with the given configuration.
func Initialize(c *Config) {
	m.Lock()
	defer m.Unlock()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.TimestampFunc = time.Now().UTC
	zerolog.CallerMarshalFunc = formatCaller

	Daemon = newLogger(c.Daemon, os.Stdout)
}

// newLogger builds a zerolog.Logger from a LoggerConfig and writer. Returns
// a no-op logger when c.Enabled is false; otherwise honours pretty-print,
// caller info, and the parsed level.
func newLogger(c *LoggerConfig, writer io.Writer) zerolog.Logger {
	var l zerolog.Logger

	if !c.Enabled {
		return zerolog.Nop()
	}

	if c.PrettyPrint {
		l = zerolog.New(zerolog.ConsoleWriter{
			Out:          writer,
			TimeLocation: time.UTC,
			TimeFormat:   time.RFC3339Nano,
			FormatLevel:  formatLevel,
		})
	} else {
		l = zerolog.New(writer)
	}

	ctx := l.With()

	if c.IncludeCaller {
		ctx = ctx.Caller()
	}

	return ctx.Timestamp().Logger().Level(parseLevel(c.Level))
}

// formatLevel renders a zerolog level for the pretty-printing ConsoleWriter,
// applying the package-defined ANSI colour for the level when one is mapped.
func formatLevel(i interface{}) string {
	var l zerolog.Level

	switch v := i.(type) {
	case zerolog.Level:
		l = v
	case string:
		l = parseLevel(v)
	}

	ul := strings.ToUpper(l.String())
	if c, ok := zerolog.LevelColors[l]; ok {
		return colorize(ul, c, false)
	}

	return ul
}

// formatCaller shortens a full source-file path to its trailing three
// segments so log lines stay readable (e.g. "internal/foo/bar.go:42").
func formatCaller(pc uintptr, file string, line int) string {
	fileParts := strings.Split(file, string(os.PathSeparator))
	truncFile := path.Join(fileParts[len(fileParts)-3:]...)

	return fmt.Sprintf("%s:%d", truncFile, line)
}

// colorize wraps s in an ANSI colour escape sequence with code c, or
// returns the value unchanged when disabled is true. Used by formatLevel.
func colorize(s interface{}, c int, disabled bool) string {
	if disabled {
		return fmt.Sprintf("%s", s)
	}
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}
