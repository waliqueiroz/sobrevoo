package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// NewGeoDataElevationCommand creates the "geodata elevation" command, which
// exposes GeoDataService.ElevationAt (FR-017, FR-018 of the fourth stage). The
// coordinate comes in flags, not arguments, so a negative value is never taken
// for a flag.
func NewGeoDataElevationCommand(geoDataService application.GeoDataService) *cobra.Command {
	var latFlag, lonFlag string

	cmd := &cobra.Command{
		Use:   "elevation --lat <degrees> --lon <degrees>",
		Short: "Read the elevation, in meters, of a coordinate from the registered elevation data",
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
			for _, name := range []string{"lat", "lon"} {
				if !cmd.Flags().Changed(name) {
					return newUsageError(fmt.Errorf("--%s is required", name))
				}
			}

			latitude, err := parseFiniteNumber(latFlag)
			if err != nil {
				return newUsageError(fmt.Errorf("--lat: %w", err))
			}
			longitude, err := parseFiniteNumber(lonFlag)
			if err != nil {
				return newUsageError(fmt.Errorf("--lon: %w", err))
			}

			reading, err := geoDataService.ElevationAt(latitude, longitude)
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), formatElevationReading(reading))
			return nil
		},
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return newUsageError(err) })

	cmd.Flags().StringVar(&latFlag, "lat", "", "Latitude in decimal degrees, from -90 to 90")
	cmd.Flags().StringVar(&lonFlag, "lon", "", "Longitude in decimal degrees, from -180 to 180")

	return cmd
}

// formatElevationReading renders a reading as the English, human-readable
// text described in contracts/cli.md.
func formatElevationReading(reading domain.ElevationReading) string {
	source := fmt.Sprintf("Source: %s (cell row %d, column %d)\n", reading.Source.Name, reading.Row, reading.Col)

	if !reading.HasValue {
		return "Elevation: no value (the file has no data for this point)\n" + source
	}
	return fmt.Sprintf("Elevation: %.1f m\n", reading.Meters) + source
}
