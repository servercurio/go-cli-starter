package cli

import (
	"github.com/spf13/cobra"

	"github.com/servercurio/go-cli-starter/internal/application"
	"github.com/servercurio/go-cli-starter/internal/version"
)

// rootFlags holds the values bound to the root command's persistent flags.
// Subcommands read this struct in their PreRunE to overlay flag values onto
// the loaded *application.Config — the final stage in the
// defaults → file → env → flag precedence chain.
type rootFlags struct {
	configFile string
	logLevel   string
	logPretty  bool
	dbDSN      string
	poolSize   int
}

// rootContext bundles the shared state passed from the root command to each
// subcommand: the parsed flags, the cumulative *application.Config (mutated
// by PersistentPreRunE), and the constructed *application.Application
// (populated by PersistentPreRunE so subcommand RunEs reach for one
// instance, not three).
type rootContext struct {
	flags *rootFlags
	cfg   *application.Config
	app   *application.Application
}

// NewRootCommand returns the top-level Cobra command tree. cmd/daemon's
// main.go calls Execute on the result; everything else flows from here.
func NewRootCommand() *cobra.Command {
	flags := &rootFlags{}
	rc := &rootContext{flags: flags}

	root := &cobra.Command{
		Use:   "appcli",
		Short: "Composable starter for one-shot CLI tools and CLI daemons",
		Long: `appcli is a starter template for building Go command-line tools and
daemons. It bundles structured logging, layered configuration
(defaults → file → env → flags), an optional PostgreSQL/Bun ORM, and a
shared goroutine pool for fan-out work.

Replace the example serve and copy subcommands with your own to ship a
new CLI.`,
		Version:       version.Tag(),
		SilenceUsage:  true,
		SilenceErrors: false,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return prepare(cmd, rc)
		},
	}

	root.PersistentFlags().StringVar(&flags.configFile, "config", "", "explicit path to a config file (overrides search paths)")
	root.PersistentFlags().StringVar(&flags.logLevel, "log-level", "", "logging level (trace, debug, info, warn, error)")
	root.PersistentFlags().BoolVar(&flags.logPretty, "log-pretty", false, "force pretty-printed log output (overrides config)")
	root.PersistentFlags().StringVar(&flags.dbDSN, "db-dsn", "", "database connection string (empty disables the database subsystem)")
	root.PersistentFlags().IntVar(&flags.poolSize, "workers", 0, "goroutine pool size (overrides APP_POOL_SIZE / config)")

	root.AddCommand(
		newServeCommand(rc),
		newCopyCommand(rc),
		newVersionCommand(),
	)

	return root
}

// prepare runs Application.Configure (which loads files + env), overlays any
// explicitly-set Cobra flag values, and stashes the constructed Application
// on rc so subcommand RunEs can reach for it.
//
// Skipped for the version and help subcommands — they print embedded
// metadata and need neither config nor logging spin-up.
func prepare(cmd *cobra.Command, rc *rootContext) error {
	switch cmd.Name() {
	case "version", "help", "completion":
		return nil
	}

	cfg := application.DefaultConfig()
	app := application.NewApplication(cfg)
	if rc.flags.configFile != "" {
		app.ConfigFileName = rc.flags.configFile
	}
	if err := app.Configure(); err != nil {
		return err
	}

	// Flag overlay (highest precedence). Only apply when the user
	// actually set the flag — otherwise zero-valued flag defaults would
	// clobber values loaded from file/env.
	flags := rc.flags
	if cmd.Flags().Changed("log-level") {
		cfg.Logging.Daemon.Level = flags.logLevel
	}
	if cmd.Flags().Changed("log-pretty") {
		cfg.Logging.Daemon.PrettyPrint = flags.logPretty
	}
	if cmd.Flags().Changed("db-dsn") {
		cfg.Database.DSN = flags.dbDSN
	}
	if cmd.Flags().Changed("workers") {
		cfg.Pool.Size = flags.poolSize
	}

	rc.cfg = cfg
	rc.app = app
	return nil
}
