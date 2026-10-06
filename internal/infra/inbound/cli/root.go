// Package cli implements Sobrevoo's command-line adapter: it converts flags
// and arguments into use case input, and formats use case output back into
// text. It contains no business rule of its own (Constitution Principle
// III) — everything here is glue around internal/application.
package cli

import "github.com/spf13/cobra"

// NewRootCommand creates the "sobrevoo" root command. Subcommands (such as
// "inspect") are attached by the composition root (cmd/sobrevoo/main.go).
// version is resolved by that same composition root (cmd/sobrevoo/version.go)
// and handed here ready to use — this package never resolves it itself.
func NewRootCommand(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "sobrevoo",
		Short: "Sobrevoo generates flyover videos from GPS tracks",
		// Error presentation and process exit codes are handled entirely by
		// the composition root, based on the error Execute() returns —
		// Cobra's own "Error: ..." + usage banner would duplicate or
		// contradict that (contracts/cli.md).
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       version,
	}

	// Setting Version makes Cobra register a --version flag that short-
	// circuits before any other command code runs (including RunE), prints
	// the template below, and returns nil — already satisfying "no file,
	// registry or network access" for free. The default template inserts
	// the word "version" between the two; this one keeps the line down to
	// exactly the program name and the version, nothing else
	// (contracts/version-flag.md).
	root.SetVersionTemplate("{{.DisplayName}} {{.Version}}\n")

	return root
}
