// Package cli implements Sobrevoo's command-line adapter: it converts flags
// and arguments into use case input, and formats use case output back into
// text. It contains no business rule of its own (Constitution Principle
// III) — everything here is glue around internal/application.
package cli

import "github.com/spf13/cobra"

// NewRootCommand creates the "sobrevoo" root command. Subcommands (such as
// "inspect") are attached by the composition root (cmd/sobrevoo/main.go).
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "sobrevoo",
		Short: "Sobrevoo generates flyover videos from GPS tracks",
		// Error presentation and process exit codes are handled entirely by
		// the composition root, based on the error Execute() returns —
		// Cobra's own "Error: ..." + usage banner would duplicate or
		// contradict that (contracts/cli.md).
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	return root
}
