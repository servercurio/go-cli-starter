package cli

import (
	"github.com/spf13/cobra"

	"github.com/servercurio/go-cli-starter/internal/version"
)

// newVersionCommand prints the embedded build metadata and exits. Bypasses
// PersistentPreRunE so it never touches the filesystem, env, or database —
// useful when scripting against a binary whose config might be invalid.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and commit metadata",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			if _, err := out.Write([]byte(version.Tag() + " (" + version.Commit() + ")\n")); err != nil {
				return err
			}
			return nil
		},
	}
}
