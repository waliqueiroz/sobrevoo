package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
)

// NewGeoDataRegisterCommand creates the "geodata register" command, which
// exposes GeoDataService.Register (FR-001 through FR-008).
func NewGeoDataRegisterCommand(geoDataService application.GeoDataService) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "register <file>",
		Short: "Register a local base map (MBTiles) or elevation (GeoTIFF) data file",
		// See root.go: error presentation and exit codes are handled
		// entirely by the composition root.
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return newUsageError(fmt.Errorf("accepts exactly one file argument, received %d", len(args)))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return newUsageError(fmt.Errorf("--name is required"))
			}
			return runGeoDataRegister(cmd, geoDataService, args[0], name)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name to register this data source under (required, must be unique)")

	return cmd
}

func runGeoDataRegister(cmd *cobra.Command, geoDataService application.GeoDataService, path, name string) error {
	source, err := geoDataService.Register(name, path)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Registered %q as %s, covering %s\n",
		source.Name, source.Type, formatBoundingBox(source.BoundingBox))

	return nil
}
