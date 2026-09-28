package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// NewFlightCommand creates the "fly" command, which chains the six existing
// stages behind a single call (FlightService.Fly): from a GPS track straight
// to a video, using the geo data already registered, without requiring any
// other command or intermediate file (the seventh stage). defaults are the
// plan parameters, defaultResolution the frame resolution, defaultAppearance
// the trail/marker/background the frames are drawn with (008-frame-
// appearance) and defaultQuality the video quality used when the
// corresponding flag is not given — the same defaults "plan", "render all"
// and "video" already use.
func NewFlightCommand(
	flightService application.FlightService,
	defaults domain.PlanParameters,
	defaultResolution domain.Resolution,
	defaultAppearance domain.Appearance,
	defaultQuality domain.VideoQuality,
	options ...RenderOption,
) *cobra.Command {
	settings := newRenderSettings(options)
	var outputFlag, qualityFlag, resolutionFlag, keepFlag string
	var durationFlag, fpsFlag, distanceFlag, tiltFlag, aspectFlag string
	var trailColorFlag, trailWidthFlag, markerColorFlag, markerRadiusFlag, backgroundColorFlag string
	var overwriteFlag bool

	cmd := &cobra.Command{
		Use:   "fly <track-file>",
		Short: "Turn a GPS track into a flight video in a single command",
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
			if outputFlag == "" {
				return newUsageError(fmt.Errorf("--output is required"))
			}
			// The video is always an MP4 file: another extension would be a name
			// that says otherwise.
			if !strings.EqualFold(filepath.Ext(outputFlag), ".mp4") {
				return newUsageError(fmt.Errorf("--output must end in .mp4: the video is always an MP4 file"))
			}

			// Every parameter flag is parsed exactly the way the command that
			// already owns it parses it (research.md item 9), so the same value
			// always means the same thing and the same invalid value fails the
			// same way (FR-003).
			parameters, err := parsePlanParameters(cmd, defaults, durationFlag, fpsFlag, distanceFlag, tiltFlag, aspectFlag)
			if err != nil {
				return err
			}
			resolution, err := domain.ParseResolution(resolutionFlag)
			if err != nil {
				return err
			}
			quality, err := domain.ParseVideoQuality(qualityFlag)
			if err != nil {
				return newUsageError(fmt.Errorf("--quality %w", err))
			}
			appearance, err := parseAppearance(cmd, trailColorFlag, trailWidthFlag, markerColorFlag, markerRadiusFlag, backgroundColorFlag, defaultAppearance)
			if err != nil {
				return err
			}
			warnIfFramingCut(cmd.ErrOrStderr(), domain.CameraPlan{Parameters: parameters}, resolution)

			request := domain.FlightRequest{
				Parameters: parameters,
				Resolution: resolution,
				Appearance: appearance,
				Quality:    quality,
				Output:     outputFlag,
				Keep:       keepFlag,
				Overwrite:  overwriteFlag,
			}

			return runFly(cmd, settings, flightService, args[0], request)
		},
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return newUsageError(err) })

	cmd.Flags().StringVar(&outputFlag, "output", "", "Video file to write (required)")
	cmd.Flags().StringVar(&durationFlag, "duration", "", "Video duration in seconds (default: computed from the track's length)")
	cmd.Flags().StringVar(&fpsFlag, "fps", strconv.FormatFloat(defaults.FrameRate, 'g', -1, 64), "Frames per second, from 1 to 120")
	cmd.Flags().StringVar(&distanceFlag, "distance", levelName(defaults.Distance), "How far the camera flies from the track: low, medium, or high")
	cmd.Flags().StringVar(&tiltFlag, "tilt", levelName(defaults.Tilt), "How steeply the camera looks down: low (near the horizon), medium, or high (near vertical)")
	cmd.Flags().StringVar(&aspectFlag, "aspect", defaults.Aspect.String(), "Shape of the video, WIDTH:HEIGHT: the opening and the closing frame the whole track for it (for example 9:16 vertical, 16:9 horizontal)")
	cmd.Flags().StringVar(&resolutionFlag, "resolution", formatResolution(defaultResolution), resolutionUsage)
	cmd.Flags().StringVar(&trailColorFlag, "trail-color", formatColor(defaultAppearance.TrailColor), trailColorUsage)
	cmd.Flags().StringVar(&trailWidthFlag, "trail-width", strconv.FormatFloat(defaultAppearance.TrailWidthRatio, 'g', -1, 64), trailWidthUsage)
	cmd.Flags().StringVar(&markerColorFlag, "marker-color", formatColor(defaultAppearance.MarkerColor), markerColorUsage)
	cmd.Flags().StringVar(&markerRadiusFlag, "marker-radius", strconv.FormatFloat(defaultAppearance.MarkerRadiusRatio, 'g', -1, 64), markerRadiusUsage)
	cmd.Flags().StringVar(&backgroundColorFlag, "background-color", formatColor(defaultAppearance.BackgroundColor), backgroundColorUsage)
	cmd.Flags().StringVar(&qualityFlag, "quality", defaultQuality.String(), "Quality of the video: low (fast to make, small), medium (for publishing) or high (for keeping)")
	cmd.Flags().StringVar(&keepFlag, "keep", "", "Directory to keep the plan, the slice and the frames in, and to reuse them from on a later run (default: a temporary directory, removed at the end)")
	cmd.Flags().BoolVar(&overwriteFlag, "overwrite", false, "Replace the video file, and any stale intermediate under --keep, if they already exist")

	return cmd
}

func runFly(
	cmd *cobra.Command,
	settings renderSettings,
	flightService application.FlightService,
	trackPath string,
	request domain.FlightRequest,
) error {
	file, err := os.Open(trackPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Ctrl+C, or a request to stop, ends the run in an orderly way, as soon as
	// the stage in progress allows it.
	ctx, stop := interruptContext(commandContext(cmd))
	defer stop()

	printer := newFlyProgressPrinter(cmd.ErrOrStderr(), settings.isTerminal(cmd.ErrOrStderr()))
	summary, err := flightService.Fly(ctx, file, request, printer.report)
	printer.finish()

	out := cmd.OutOrStdout()
	switch {
	case errors.Is(err, domain.ErrFlightInterrupted):
		fmt.Fprint(out, formatFlightInterrupted(summary))
		return err
	case err != nil:
		return err
	}

	fmt.Fprint(out, formatFlightSummary(request.Output, summary))
	return nil
}

// formatFlightSummary renders the summary of a single-command run, in
// English, in the labels and order of contracts/cli.md: the frames summary
// (as "render all" shows it), then the video summary (as "video" shows it),
// then the run's own total time.
func formatFlightSummary(output string, summary domain.FlightSummary) string {
	var b strings.Builder
	b.WriteString(formatFramesSummary(summary.Render.Requested, summary.FramesDirectory, summary.Render))
	b.WriteString(formatVideoSummary(output, summary.Video))
	fmt.Fprintf(&b, "Total time: %s\n", formatElapsed(summary.Elapsed))
	return b.String()
}

// formatFlightInterrupted renders the summary of a run that was interrupted:
// which stages had already completed, and — when they had started — the
// partial progress of the frame-rendering and video-encoding stages.
func formatFlightInterrupted(summary domain.FlightSummary) string {
	var b strings.Builder

	if len(summary.Completed) == 0 {
		b.WriteString("Interrupted before any stage completed\n")
	} else {
		last := summary.Completed[len(summary.Completed)-1]
		fmt.Fprintf(&b, "Interrupted after stage %d/5 (%s); completed: %s\n", len(summary.Completed), last, joinStages(summary.Completed))
	}
	if summary.Render.Requested > 0 {
		fmt.Fprintf(&b, "Frames: %d of %d drawn\n", summary.Render.Drawn+summary.Render.Kept, summary.Render.Requested)
	}
	if summary.Video.Frames > 0 {
		fmt.Fprintf(&b, "Encoded: %d of %d frames\n", summary.Video.Encoded, summary.Video.Frames)
	}

	return b.String()
}

// joinStages lists stages by their label, separated by commas.
func joinStages(stages []domain.FlightStage) string {
	labels := make([]string, len(stages))
	for i, stage := range stages {
		labels[i] = stage.String()
	}
	return strings.Join(labels, ", ")
}
