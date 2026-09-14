package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
)

// NewGeoDataRemoveCommand creates the "geodata remove" command, which
// exposes RemoveGeoDataService (FR-011, FR-012).
func NewGeoDataRemoveCommand(removeGeoDataService application.RemoveGeoDataService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a registered geo data source, without deleting its file",
		// See root.go: error presentation and exit codes are handled
		// entirely by the composition root.
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return newUsageError(fmt.Errorf("accepts exactly one name argument, received %d", len(args)))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGeoDataRemove(cmd, removeGeoDataService, args[0])
		},
	}

	return cmd
}

func runGeoDataRemove(cmd *cobra.Command, removeGeoDataService application.RemoveGeoDataService, name string) error {
	if err := removeGeoDataService.Execute(application.RemoveGeoDataInput{Name: name}); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Removed %q.\n", name)

	return nil
}
