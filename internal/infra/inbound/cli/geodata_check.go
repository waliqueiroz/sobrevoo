package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// NewGeoDataCheckCommand creates the "geodata check" command, which
// exposes GeoDataService.CheckCoverage (FR-013 through FR-018).
func NewGeoDataCheckCommand(geoDataService application.GeoDataService) *cobra.Command {
	var baseMapFlag, elevationFlag string

	cmd := &cobra.Command{
		Use:   "check <track-file>",
		Short: "Check whether a GPS track is covered by the registered geo data",
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
			selection := parseSourceSelection(cmd, baseMapFlag, elevationFlag)
			return runGeoDataCheck(cmd, geoDataService, args[0], selection)
		},
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return newUsageError(err) })

	cmd.Flags().StringVar(&baseMapFlag, "base-map", "", baseMapNameUsage)
	cmd.Flags().StringVar(&elevationFlag, "elevation", "", elevationNameUsage)

	return cmd
}

func runGeoDataCheck(cmd *cobra.Command, geoDataService application.GeoDataService, path string, selection domain.SourceSelection) error {
	// CheckCoverage needs an io.Reader (same as TrackService.Inspect),
	// so — same as inspect.go — this adapter is the one that opens the track
	// file; a
	// plain I/O error here (file missing, unreadable) falls through to
	// exit_code.go's generic fallback, exactly as it does for "inspect"
	// (contracts/cli.md).
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	output, err := geoDataService.CheckCoverage(file, selection)
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), formatCoverage(output))

	return nil
}

// formatCoverage renders a domain.CoverageReport as the English,
// human-readable report described in contracts/cli.md (FR-013 through
// FR-017).
func formatCoverage(output domain.CoverageReport) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Coverage: %s\n", output.Status)

	if len(output.UncoveredSegments) > 0 {
		fmt.Fprintln(&b, "Uncovered segments:")
		// Each segment is where it starts and where it ends, as (lat, lon) —
		// the same way the "area not covered" error of a slice says it — never
		// as ranges, which would read as a box: the start and the end of a
		// segment of a loop can be a few meters apart.
		for _, segment := range output.UncoveredSegments {
			fmt.Fprintf(&b, "  - from (%.6f, %.6f) to (%.6f, %.6f): missing %s\n",
				segment.StartLatitude, segment.StartLongitude, segment.EndLatitude, segment.EndLongitude, segment.Missing)
		}
	}

	fmt.Fprintf(&b, "Base map sources used: %s\n", formatSourceNames(output.BaseMapSourcesUsed))
	fmt.Fprintf(&b, "Elevation sources used: %s\n", formatSourceNames(output.ElevationSourcesUsed))

	return b.String()
}

func formatSourceNames(sources []domain.GeoDataSource) string {
	if len(sources) == 0 {
		return "(none)"
	}

	names := make([]string, len(sources))
	for i, source := range sources {
		names[i] = source.Name
	}

	return strings.Join(names, ", ")
}
