package cli

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// NewPlanCommand creates the "plan" command, which exposes
// CameraPlanService (FR-001 through FR-025 of the third stage). defaults are
// the plan parameters used when a flag is not given; a nil Duration means the
// duration is computed from the track.
func NewPlanCommand(cameraPlanService application.CameraPlanService, defaults domain.PlanParameters) *cobra.Command {
	var durationFlag, fpsFlag, distanceFlag, tiltFlag, exportFlag string
	var overwriteFlag bool

	cmd := &cobra.Command{
		Use:   "plan <file>",
		Short: "Plan the camera flight over a GPS track and print a summary of the plan",
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
			parameters, err := parsePlanParameters(cmd, defaults, durationFlag, fpsFlag, distanceFlag, tiltFlag)
			if err != nil {
				return err
			}

			if overwriteFlag && exportFlag == "" {
				return newUsageError(fmt.Errorf("--overwrite requires --export"))
			}

			return runPlan(cmd, cameraPlanService, args[0], parameters, exportFlag, overwriteFlag)
		},
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return newUsageError(err) })

	cmd.Flags().StringVar(&durationFlag, "duration", "", "Video duration in seconds (default: computed from the track's length)")
	cmd.Flags().StringVar(&fpsFlag, "fps", strconv.FormatFloat(defaults.FrameRate, 'g', -1, 64), "Frames per second, from 1 to 120")
	cmd.Flags().StringVar(&distanceFlag, "distance", levelName(defaults.Distance), "How far the camera flies from the track: low, medium, or high")
	cmd.Flags().StringVar(&tiltFlag, "tilt", levelName(defaults.Tilt), "How steeply the camera looks down: low (near the horizon), medium, or high (near vertical)")
	cmd.Flags().StringVar(&exportFlag, "export", "", "Write the complete plan to this file, as JSON")
	cmd.Flags().BoolVar(&overwriteFlag, "overwrite", false, "With --export, replace the file if it already exists")

	return cmd
}

func parsePlanParameters(cmd *cobra.Command, defaults domain.PlanParameters, durationFlag, fpsFlag, distanceFlag, tiltFlag string) (domain.PlanParameters, error) {
	parameters := defaults

	if cmd.Flags().Changed("duration") {
		seconds, err := parseFiniteNumber(durationFlag)
		if err != nil {
			return domain.PlanParameters{}, newUsageError(fmt.Errorf("--duration: %w", err))
		}
		parameters.Duration = new(secondsToDuration(seconds))
	}

	frameRate, err := parseFiniteNumber(fpsFlag)
	if err != nil {
		return domain.PlanParameters{}, newUsageError(fmt.Errorf("--fps: %w", err))
	}
	parameters.FrameRate = frameRate

	if parameters.Distance, err = parseLevel(distanceFlag); err != nil {
		return domain.PlanParameters{}, newUsageError(fmt.Errorf("--distance: %w", err))
	}
	if parameters.Tilt, err = parseLevel(tiltFlag); err != nil {
		return domain.PlanParameters{}, newUsageError(fmt.Errorf("--tilt: %w", err))
	}

	return parameters, nil
}

// parseFiniteNumber parses a decimal number, rejecting anything that is not a
// finite number (including "nan" and "inf", which strconv would accept).
func parseFiniteNumber(text string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("invalid number %q", text)
	}
	return value, nil
}

// secondsToDuration converts seconds to a time.Duration, saturating instead
// of overflowing for absurd values (which the domain then rejects).
func secondsToDuration(seconds float64) time.Duration {
	nanoseconds := math.Round(seconds * float64(time.Second))
	switch {
	case nanoseconds >= math.MaxInt64:
		return math.MaxInt64
	case nanoseconds <= math.MinInt64:
		return math.MinInt64
	}
	return time.Duration(nanoseconds)
}

func runPlan(cmd *cobra.Command, cameraPlanService application.CameraPlanService, path string, parameters domain.PlanParameters, exportPath string, overwrite bool) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	plan, err := cameraPlanService.Generate(file, parameters)
	if err != nil {
		return err
	}

	// Export first: on failure nothing is printed, so an error never comes
	// with a plan (contracts/cli.md).
	if exportPath != "" {
		if err := cameraPlanService.Export(plan, exportPath, overwrite); err != nil {
			return err
		}
	}

	fmt.Fprint(cmd.OutOrStdout(), formatPlanSummary(plan))
	if exportPath != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Plan written to %s\n", exportPath)
	}

	return nil
}

// formatPlanSummary renders a plan's summary as the English, human-readable
// text described in contracts/cli.md (FR-019).
func formatPlanSummary(plan domain.CameraPlan) string {
	summary := plan.Summary

	var b strings.Builder
	fmt.Fprintf(&b, "Duration: %.1f s (%s)\n", summary.Duration.Seconds(), durationModeLabel(summary.DurationMode))
	fmt.Fprintf(&b, "Frame rate: %.1f fps\n", summary.FrameRate)
	fmt.Fprintf(&b, "Frames: %d\n", summary.FrameCount)
	fmt.Fprintf(&b, "Camera altitude: %.1f m - %.1f m\n", summary.MinCameraAltitude, summary.MaxCameraAltitude)
	fmt.Fprintf(&b, "Camera distance: %.1f m - %.1f m\n", summary.MinCameraDistance, summary.MaxCameraDistance)
	fmt.Fprintf(&b, "Time reference: %s\n", formatTimeReference(plan))

	if len(summary.SmoothedSpans) == 0 {
		b.WriteString("Smoothed spans: none\n")
		return b.String()
	}

	fmt.Fprintf(&b, "Smoothed spans: %d\n", len(summary.SmoothedSpans))
	for _, span := range summary.SmoothedSpans {
		fmt.Fprintf(&b, "  %.2f s - %.2f s (%s)\n", span.Start.Seconds(), span.End.Seconds(), span.Quantity)
	}

	return b.String()
}

func durationModeLabel(mode domain.DurationMode) string {
	if mode == domain.DurationModeAutomatic {
		return "automatic"
	}
	return "requested"
}

func formatTimeReference(plan domain.CameraPlan) string {
	if plan.TimeFallbackReason == "" {
		return string(plan.TimeReference)
	}
	return fmt.Sprintf("%s (%s)", plan.TimeReference, plan.TimeFallbackReason)
}
