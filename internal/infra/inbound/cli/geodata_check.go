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
			return runGeoDataCheck(cmd, geoDataService, args[0])
		},
	}

	return cmd
}

func runGeoDataCheck(cmd *cobra.Command, geoDataService application.GeoDataService, path string) error {
	// CheckCoverage needs an io.Reader (like InspectTrackInput), so — same
	// as inspect.go — this adapter is the one that opens the track file; a
	// plain I/O error here (file missing, unreadable) falls through to
	// exit_code.go's generic fallback, exactly as it does for "inspect"
	// (contracts/cli.md).
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	output, err := geoDataService.CheckCoverage(file)
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), formatCoverage(output))

	return nil
}

// formatCoverage renders a CheckCoverageOutput as the English,
// human-readable report described in contracts/cli.md (FR-013 through
// FR-017).
func formatCoverage(output application.CheckCoverageOutput) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Coverage: %s\n", output.Status)

	if len(output.UncoveredSegments) > 0 {
		fmt.Fprintln(&b, "Uncovered segments:")
		for _, segment := range output.UncoveredSegments {
			fmt.Fprintf(&b, "  - lat [%.6f, %.6f], lon [%.6f, %.6f]: missing %s\n",
				segment.StartLatitude, segment.EndLatitude, segment.StartLongitude, segment.EndLongitude, segment.Missing)
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
