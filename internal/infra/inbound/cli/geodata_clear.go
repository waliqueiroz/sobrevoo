package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
)

// NewGeoDataClearCommand creates the "geodata clear" command, which
// exposes GeoDataService.Clear (010-geo-data-source-control FR-001
// through FR-003): removes every registered source at once, never
// touching a data file, only with explicit confirmation.
func NewGeoDataClearCommand(geoDataService application.GeoDataService) *cobra.Command {
	var confirmFlag bool

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove every registered geo data source at once, without deleting any file",
		// See root.go: error presentation and exit codes are handled
		// entirely by the composition root.
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return newUsageError(fmt.Errorf("accepts no arguments, received %d", len(args)))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGeoDataClear(cmd, geoDataService, confirmFlag)
		},
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return newUsageError(err) })

	cmd.Flags().BoolVar(&confirmFlag, "confirm", false, "Confirm removing every registered source (required; never a prompt)")

	return cmd
}

func runGeoDataClear(cmd *cobra.Command, geoDataService application.GeoDataService, confirm bool) error {
	removed, err := geoDataService.Clear(confirm)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Cleared the registry: %d entries removed.\n", removed)

	return nil
}
