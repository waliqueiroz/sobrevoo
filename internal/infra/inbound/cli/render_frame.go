package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// NewRenderFrameCommand creates the "render frame" command, which loads an
// exported camera plan and the slice made for it and draws one frame of the
// flight (FrameService.DrawFrame): the way to check the framing before drawing
// them all. defaultResolution is the resolution used when the user chooses
// none, and defaultAppearance the trail, marker and background it is drawn
// with (008-frame-appearance).
func NewRenderFrameCommand(
	cameraPlanService application.CameraPlanService,
	geoSliceService application.GeoSliceService,
	frameService application.FrameService,
	defaultResolution domain.Resolution,
	defaultAppearance domain.Appearance,
) *cobra.Command {
	var numberFlag, outputFlag, resolutionFlag string
	var trailColorFlag, trailWidthFlag, markerColorFlag, markerRadiusFlag, backgroundColorFlag string
	var overwriteFlag bool

	cmd := &cobra.Command{
		Use:   "frame <plan-file> <slice-file>",
		Short: "Draw one frame of a flight, by its number in the plan, into an image file",
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
			if !cmd.Flags().Changed("number") {
				return newUsageError(fmt.Errorf("--number is required"))
			}
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

			return runRenderFrame(cmd, cameraPlanService, geoSliceService, frameService, args[0], args[1], numberFlag, outputFlag, overwriteFlag, resolution, appearance)
		},
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return newUsageError(err) })

	cmd.Flags().StringVar(&numberFlag, "number", "", "Number of the frame in the plan, from 0 (required)")
	cmd.Flags().StringVar(&outputFlag, "output", "", "Image file to write the frame to, as a PNG (required)")
	cmd.Flags().StringVar(&resolutionFlag, "resolution", formatResolution(defaultResolution), resolutionUsage)
	cmd.Flags().StringVar(&trailColorFlag, "trail-color", formatColor(defaultAppearance.TrailColor), trailColorUsage)
	cmd.Flags().StringVar(&trailWidthFlag, "trail-width", strconv.FormatFloat(defaultAppearance.TrailWidthRatio, 'g', -1, 64), trailWidthUsage)
	cmd.Flags().StringVar(&markerColorFlag, "marker-color", formatColor(defaultAppearance.MarkerColor), markerColorUsage)
	cmd.Flags().StringVar(&markerRadiusFlag, "marker-radius", strconv.FormatFloat(defaultAppearance.MarkerRadiusRatio, 'g', -1, 64), markerRadiusUsage)
	cmd.Flags().StringVar(&backgroundColorFlag, "background-color", formatColor(defaultAppearance.BackgroundColor), backgroundColorUsage)
	cmd.Flags().BoolVar(&overwriteFlag, "overwrite", false, "Replace the file if it already exists")

	return cmd
}

func runRenderFrame(
	cmd *cobra.Command,
	cameraPlanService application.CameraPlanService,
	geoSliceService application.GeoSliceService,
	frameService application.FrameService,
	planPath, slicePath, numberText, outputPath string,
	overwrite bool,
	resolution domain.Resolution,
	appearance domain.Appearance,
) error {
	plan, err := cameraPlanService.Load(planPath)
	if err != nil {
		return err
	}

	number, err := domain.ParseFrameNumber(numberText, len(plan.Frames))
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

	summary, err := frameService.DrawFrame(ctx, plan, slice, domain.SingleFrameRequest{
		Number:     number,
		Path:       outputPath,
		Resolution: resolution,
		Appearance: appearance,
		Overwrite:  overwrite,
	})
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), formatFrameSummary(number, len(plan.Frames), outputPath, summary))
	return nil
}

// resolutionUsage is the help of the --resolution flag of the commands that draw.
const resolutionUsage = "Size of the images as WIDTHxHEIGHT in pixels: both even, each from 180 to 3840, at most 3840x2160 in all"

// commandContext is the context of the command, or an empty one when it was
// run with none.
func commandContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

// formatFrameSummary renders the summary of one frame drawn, in English, in the
// labels and order of specs/005-frame-rendering/contracts/cli.md.
func formatFrameSummary(number, total int, path string, summary domain.RenderSummary) string {
	return fmt.Sprintf("Frame %d of %d drawn to %s\nResolution: %s\nTime: %s\nHoles: map tiles missing: %s, elevation without value: %s\n",
		number, total, path,
		formatResolution(summary.Resolution),
		formatElapsed(summary.Elapsed),
		yesNo(summary.MapHoleFrames > 0), yesNo(summary.ElevationHoleFrames > 0))
}

func formatResolution(r domain.Resolution) string {
	return fmt.Sprintf("%dx%d", r.Width, r.Height)
}

// formatElapsed writes a duration as HH:MM:SS, rounded to the second.
func formatElapsed(d time.Duration) string {
	seconds := int(d.Round(time.Second) / time.Second)
	return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds%3600/60, seconds%60)
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
