package cli

import (
	"github.com/spf13/cobra"
)

// copyFlags holds the per-invocation flags for the copy subcommand.
type copyFlags struct {
	recursive bool
}

// newCopyCommand returns the one-shot example. The actual copy work lives
// on Application.Copy; this file is a thin Cobra shell that translates
// args/flags into a single method call.
func newCopyCommand(rc *rootContext) *cobra.Command {
	flags := &copyFlags{}

	cmd := &cobra.Command{
		Use:   "copy SRC DST",
		Short: "Copy a file or directory (example one-shot)",
		Long: `copy is a one-shot example demonstrating positional args, optional
flags, structured logging, and pool-backed parallelism.

Without --recursive, SRC must be a regular file and is copied to DST.
With --recursive, SRC must be a directory; every file under it is
submitted to the shared goroutine pool for parallel copy.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return rc.app.Copy(cmd.Context(), args[0], args[1], flags.recursive)
		},
	}

	cmd.Flags().BoolVarP(&flags.recursive, "recursive", "r", false, "recursively copy a directory tree using the goroutine pool")

	return cmd
}
