package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// NewInspectCommand creates the "inspect" command, which exposes
// InspectTrackService (FR-001 through FR-027). defaultLevel (FR-016) is
// used as the --simplification/--smoothing flags' default value, so the
// service always receives a concrete Level either way — it has no notion
// of an "unset" level to fall back on itself.
func NewInspectCommand(inspectTrackService application.InspectTrackService, defaultLevel domain.Level) *cobra.Command {
	var simplificationFlag, smoothingFlag string

	cmd := &cobra.Command{
		Use:   "inspect <file>",
		Short: "Read a GPS track file and print a summary of it",
		// See root.go: error presentation and exit codes are handled
		// entirely by the composition root. Set here too (not just on the
		// root command) so this holds even when this command is executed
		// directly, as the test suite does.
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return newUsageError(fmt.Errorf("accepts exactly one file argument, received %d", len(args)))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			simplificationLevel, err := parseLevel(simplificationFlag)
			if err != nil {
				return newUsageError(fmt.Errorf("--simplification: %w", err))
			}

			smoothingLevel, err := parseLevel(smoothingFlag)
			if err != nil {
				return newUsageError(fmt.Errorf("--smoothing: %w", err))
			}

			return runInspect(cmd, inspectTrackService, args[0], simplificationLevel, smoothingLevel)
		},
	}

	defaultLevelName := levelName(defaultLevel)
	cmd.Flags().StringVar(&simplificationFlag, "simplification", defaultLevelName, "Simplification level applied to the treated route: low, medium, or high")
	cmd.Flags().StringVar(&smoothingFlag, "smoothing", defaultLevelName, "Smoothing level applied to the treated route: low, medium, or high")

	return cmd
}

func runInspect(cmd *cobra.Command, inspectTrackService application.InspectTrackService, path string, simplificationLevel, smoothingLevel domain.Level) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	output, err := inspectTrackService.Execute(application.InspectTrackInput{
		Reader:              file,
		SimplificationLevel: simplificationLevel,
		SmoothingLevel:      smoothingLevel,
	})
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), formatSummary(output))

	return nil
}

// levelName renders a Level as the CLI flag value that represents it.
func levelName(level domain.Level) string {
	switch level {
	case domain.LevelLow:
		return "low"
	case domain.LevelHigh:
		return "high"
	default:
		return "medium"
	}
}

// parseLevel converts a --simplification/--smoothing flag value into a
// domain.Level.
func parseLevel(name string) (domain.Level, error) {
	switch name {
	case "low":
		return domain.LevelLow, nil
	case "medium":
		return domain.LevelMedium, nil
	case "high":
		return domain.LevelHigh, nil
	default:
		return 0, fmt.Errorf("invalid level %q, must be one of low, medium, high", name)
	}
}

// formatSummary renders an InspectTrackOutput as the English, human-readable
// summary described in contracts/cli.md (FR-025). All runtime I/O of the
// tool is in English, independently of the language used by the project's
// specification artifacts (see spec.md's scope note on this).
func formatSummary(output application.InspectTrackOutput) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Format: %s\n", output.Format)
	fmt.Fprintf(&b, "Points: %d -> %d (original -> treated)\n", output.PointCountOriginal, output.PointCountTreated)
	fmt.Fprintf(&b, "Distance: %.2f km\n", output.TotalDistanceMeters/1000)
	fmt.Fprintf(&b, "Elevation gain: %s\n", formatElevationGain(output.ElevationGainMeters))
	fmt.Fprintf(&b, "Duration: %s\n", formatDuration(output))
	fmt.Fprintf(&b, "Bounding box: %s\n", formatBoundingBox(output.BoundingBox))
	fmt.Fprintf(&b, "Discarded points: %d (impossible coordinates: %d, consecutive duplicates: %d, implausible jumps: %d)\n",
		output.Discarded.Total(), output.Discarded.ImpossibleCoordinates, output.Discarded.ConsecutiveDuplicates, output.Discarded.ImplausibleJumps)

	return b.String()
}

func formatElevationGain(gain *float64) string {
	if gain == nil {
		return "not available (no altitude data)"
	}
	return fmt.Sprintf("%.1f m", *gain)
}

func formatDuration(output application.InspectTrackOutput) string {
	if output.Duration == nil {
		return "not available (no time data)"
	}
	return output.Duration.String()
}

func formatBoundingBox(box domain.BoundingBox) string {
	summary := fmt.Sprintf("lat [%.6f, %.6f], lon [%.6f, %.6f]", box.MinLatitude, box.MaxLatitude, box.MinLongitude, box.MaxLongitude)
	if box.CrossesAntimeridian {
		summary += " (crosses the antimeridian)"
	}
	return summary
}
