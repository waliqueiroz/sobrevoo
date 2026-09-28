package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// NewRenderAllCommand creates the "render all" command, which loads an exported
// camera plan and the slice made for it and draws every frame of the flight into
// a directory (FrameService.DrawFrames), numbered as the plan numbers them so the
// stage that follows can join them without ambiguity. defaultResolution is the
// resolution used when the user chooses none, and defaultAppearance the trail,
// marker and background it is drawn with (008-frame-appearance).
func NewRenderAllCommand(
	cameraPlanService application.CameraPlanService,
	geoSliceService application.GeoSliceService,
	frameService application.FrameService,
	defaultResolution domain.Resolution,
	defaultAppearance domain.Appearance,
	options ...RenderOption,
) *cobra.Command {
	settings := newRenderSettings(options)
	var outputFlag, resolutionFlag string
	var trailColorFlag, trailWidthFlag, markerColorFlag, markerRadiusFlag, backgroundColorFlag string
	var overwriteFlag bool

	cmd := &cobra.Command{
		Use:   "all <plan-file> <slice-file>",
		Short: "Draw all the frames of a flight into a directory, as numbered images",
		// See root.go: error presentation and exit codes are handled
		// entirely by the composition root.
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return newUsageError(fmt.Errorf("accepts a plan file and a slice file, received %d arguments", len(args)))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputFlag == "" {
				return newUsageError(fmt.Errorf("--output is required"))
			}

			resolution, err := domain.ParseResolution(resolutionFlag)
			if err != nil {
				return err
			}
			appearance, err := parseAppearance(cmd, trailColorFlag, trailWidthFlag, markerColorFlag, markerRadiusFlag, backgroundColorFlag, defaultAppearance)
			if err != nil {
				return err
			}

			return runRenderAll(cmd, settings, cameraPlanService, geoSliceService, frameService, args[0], args[1], outputFlag, overwriteFlag, resolution, appearance)
		},
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return newUsageError(err) })

	cmd.Flags().StringVar(&outputFlag, "output", "", "Directory to write the frames to (required); it is created if it does not exist")
	cmd.Flags().StringVar(&resolutionFlag, "resolution", formatResolution(defaultResolution), resolutionUsage)
	cmd.Flags().StringVar(&trailColorFlag, "trail-color", formatColor(defaultAppearance.TrailColor), trailColorUsage)
	cmd.Flags().StringVar(&trailWidthFlag, "trail-width", strconv.FormatFloat(defaultAppearance.TrailWidthRatio, 'g', -1, 64), trailWidthUsage)
	cmd.Flags().StringVar(&markerColorFlag, "marker-color", formatColor(defaultAppearance.MarkerColor), markerColorUsage)
	cmd.Flags().StringVar(&markerRadiusFlag, "marker-radius", strconv.FormatFloat(defaultAppearance.MarkerRadiusRatio, 'g', -1, 64), markerRadiusUsage)
	cmd.Flags().StringVar(&backgroundColorFlag, "background-color", formatColor(defaultAppearance.BackgroundColor), backgroundColorUsage)
	cmd.Flags().BoolVar(&overwriteFlag, "overwrite", false, "Draw every frame again, replacing the ones already there, and remove the frames of a previous flight that this plan has no number for")

	return cmd
}

func runRenderAll(
	cmd *cobra.Command,
	settings renderSettings,
	cameraPlanService application.CameraPlanService,
	geoSliceService application.GeoSliceService,
	frameService application.FrameService,
	planPath, slicePath, directory string,
	overwrite bool,
	resolution domain.Resolution,
	appearance domain.Appearance,
) error {
	plan, err := cameraPlanService.Load(planPath)
	if err != nil {
		return err
	}

	warnIfFramingCut(cmd.ErrOrStderr(), plan, resolution)

	slice, err := geoSliceService.Load(slicePath)
	if err != nil {
		return err
	}

	ctx, stop := interruptContext(commandContext(cmd))
	defer stop()

	printer := &progressPrinter{out: cmd.ErrOrStderr(), terminal: settings.isTerminal(cmd.ErrOrStderr())}
	summary, err := frameService.DrawFrames(ctx, plan, slice, domain.FrameSetRequest{
		Directory:  directory,
		Resolution: resolution,
		Appearance: appearance,
		Overwrite:  overwrite,
	}, printer.report)
	printer.finish()

	out := cmd.OutOrStdout()
	switch {
	case errors.Is(err, domain.ErrRenderInterrupted):
		fmt.Fprintf(out, "Interrupted: %d of %d frames are ready; run the same command again to continue\n", summary.Drawn+summary.Kept, summary.Requested)
		fmt.Fprint(out, formatFramesSummary(len(plan.Frames), directory, summary))
	case err != nil && summary.Drawn+summary.Kept > 0:
		// something failed after frames were done: say what was
		fmt.Fprint(out, formatFramesSummary(len(plan.Frames), directory, summary))
	case err == nil:
		fmt.Fprint(out, formatFramesSummary(len(plan.Frames), directory, summary))
	}
	return err
}

// formatFramesSummary renders the summary of a drawing of many frames, in
// English, in the labels and order of specs/005-frame-rendering/contracts/cli.md.
func formatFramesSummary(total int, directory string, summary domain.RenderSummary) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Frames: %d requested, %d drawn, %d kept (already in the destination)\n", summary.Requested, summary.Drawn, summary.Kept)
	fmt.Fprintf(&b, "Resolution: %s\n", formatResolution(summary.Resolution))
	fmt.Fprintf(&b, "Time: %s\n", formatElapsed(summary.Elapsed))
	fmt.Fprintf(&b, "Holes (in the frames drawn now): %s\n", formatHoles(summary))
	if summary.Removed > 0 {
		fmt.Fprintf(&b, "Removed: %d frames from a previous set\n", summary.Removed)
	}
	fmt.Fprintf(&b, "Destination: %s (%s to %s)\n", directory, domain.FrameFileName(0), domain.FrameFileName(total-1))

	return b.String()
}

// formatHoles counts, by cause, the frames drawn that show terrain with no map
// image or over cells with no elevation.
func formatHoles(summary domain.RenderSummary) string {
	switch {
	case summary.Drawn == 0:
		return "none drawn now"
	case summary.MapHoleFrames == 0 && summary.ElevationHoleFrames == 0:
		return "none"
	}
	return fmt.Sprintf("%d with missing map tiles, %d with elevation without value", summary.MapHoleFrames, summary.ElevationHoleFrames)
}
