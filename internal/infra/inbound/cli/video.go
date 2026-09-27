package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// NewVideoCommand creates the "video" command, which loads an exported camera
// plan and joins the frames "render all" drew for it into one video file
// (VideoService.Assemble). defaultQuality is the quality of the video when the
// user chooses none.
func NewVideoCommand(
	cameraPlanService application.CameraPlanService,
	videoService application.VideoService,
	defaultQuality domain.VideoQuality,
	options ...RenderOption,
) *cobra.Command {
	settings := newRenderSettings(options)
	var outputFlag, qualityFlag string
	var overwriteFlag bool

	cmd := &cobra.Command{
		Use:   "video <plan-file> <frames-directory>",
		Short: "Join the frames of a flight into a video",
		// See root.go: error presentation and exit codes are handled
		// entirely by the composition root.
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return newUsageError(fmt.Errorf("accepts a plan file and a frames directory, received %d arguments", len(args)))
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

			quality, err := domain.ParseVideoQuality(qualityFlag)
			if err != nil {
				return newUsageError(fmt.Errorf("--quality %w", err))
			}

			return runVideo(cmd, settings, cameraPlanService, videoService, args[0], domain.VideoRequest{
				Directory: args[1],
				Output:    outputFlag,
				Quality:   quality,
				Overwrite: overwriteFlag,
			})
		},
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return newUsageError(err) })

	cmd.Flags().StringVar(&outputFlag, "output", "", "Video file to write (required)")
	cmd.Flags().StringVar(&qualityFlag, "quality", defaultQuality.String(), "Quality of the video: low (fast to make, small), medium (for publishing) or high (for keeping)")
	cmd.Flags().BoolVar(&overwriteFlag, "overwrite", false, "Replace the video file if it already exists")

	return cmd
}

func runVideo(
	cmd *cobra.Command,
	settings renderSettings,
	cameraPlanService application.CameraPlanService,
	videoService application.VideoService,
	planPath string,
	request domain.VideoRequest,
) error {
	plan, err := cameraPlanService.Load(planPath)
	if err != nil {
		return err
	}

	// Ctrl+C, or a request to stop, ends the encoding in an orderly way: the
	// encoder is stopped, nothing is left behind, and the user is told.
	ctx, stop := interruptContext(commandContext(cmd))
	defer stop()

	printer := &videoProgressPrinter{out: cmd.ErrOrStderr(), terminal: settings.isTerminal(cmd.ErrOrStderr())}
	summary, err := videoService.Assemble(ctx, plan, request, printer.report)
	printer.finish()

	switch {
	case errors.Is(err, domain.ErrVideoInterrupted):
		fmt.Fprintf(cmd.OutOrStdout(), "Interrupted: %d of %d frames were encoded; no video was written, run the same command again to start over\nTime: %s\n",
			summary.Encoded, summary.Frames, formatElapsed(summary.Elapsed))
		return err
	case err != nil:
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), formatVideoSummary(request.Output, summary))
	return nil
}

// formatVideoSummary renders the summary of a video assembled, in English, in the
// labels and order of specs/006-video-assembly/contracts/cli.md.
func formatVideoSummary(output string, summary domain.VideoSummary) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Video written to %s\n", output)
	fmt.Fprintf(&b, "Frames: %d\n", summary.Frames)
	fmt.Fprintf(&b, "Duration: %s\n", formatVideoDuration(summary.Duration()))
	fmt.Fprintf(&b, "Resolution: %s\n", formatResolution(summary.Resolution))
	fmt.Fprintf(&b, "Frame rate: %s fps\n", formatFrameRate(summary.FrameRate))
	fmt.Fprintf(&b, "Quality: %s\n", summary.Quality)
	fmt.Fprintf(&b, "Size: %s\n", formatSize(summary.SizeBytes))
	fmt.Fprintf(&b, "Encoder: %s\n", summary.Encoder)
	fmt.Fprintf(&b, "Time: %s\n", formatElapsed(summary.Elapsed))

	return b.String()
}

// formatVideoDuration writes a duration as HH:MM:SS.mmm.
func formatVideoDuration(d time.Duration) string {
	milliseconds := int64(d.Round(time.Millisecond) / time.Millisecond)
	seconds := milliseconds / 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", seconds/3600, seconds%3600/60, seconds%60, milliseconds%1000)
}

// formatFrameRate writes a frame rate as the shortest decimal that stands for it:
// 30, 29.97.
func formatFrameRate(rate float64) string {
	return strconv.FormatFloat(rate, 'f', -1, 64)
}

// formatSize writes a size in bytes, or in KiB, MiB or GiB with one decimal.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	size := float64(bytes)
	for _, name := range []string{"KiB", "MiB", "GiB"} {
		size /= unit
		if size < unit || name == "GiB" {
			return fmt.Sprintf("%.1f %s", size, name)
		}
	}
	return ""
}
