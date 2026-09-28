package cli_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

// executeRenderAllWith runs "render all" with options for the command.
func executeRenderAllWith(t *testing.T, m renderCommandMocks, options []cli.RenderOption, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	var cmd *cobra.Command = cli.NewRenderAllCommand(m.planService, m.sliceService, m.frameService, defaultResolution, defaultAppearance, options...)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), errOut.String(), err
}

// drawing makes the frame service report the progress of a drawing of `total`
// frames, one every 400 ms, and end with the summary.
func drawing(total int, summary domain.RenderSummary, result error) func(context.Context, domain.CameraPlan, domain.GeoSlice, domain.FrameSetRequest, func(domain.RenderProgress)) (domain.RenderSummary, error) {
	return func(_ context.Context, _ domain.CameraPlan, _ domain.GeoSlice, _ domain.FrameSetRequest, progress func(domain.RenderProgress)) (domain.RenderSummary, error) {
		for i := 1; i <= total; i++ {
			progress(domain.RenderProgress{Done: i, Total: total, Elapsed: time.Duration(i) * 400 * time.Millisecond})
		}
		return summary, result
	}
}

func Test_RenderAllCommand_Progress(t *testing.T) {
	plan := planOfFrames(60)
	slice := builddomain.NewGeoSliceBuilder().Build()
	summary := domain.RenderSummary{Requested: 60, Drawn: 60, Resolution: defaultResolution}

	setUp := func(t *testing.T, run func(context.Context, domain.CameraPlan, domain.GeoSlice, domain.FrameSetRequest, func(domain.RenderProgress)) (domain.RenderSummary, error)) renderCommandMocks {
		t.Helper()
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(run)
		return m
	}
	asTerminal := cli.WithTerminalCheck(func(io.Writer) bool { return true })
	notTerminal := cli.WithTerminalCheck(func(io.Writer) bool { return false })

	t.Run("should write a line to stderr every ten frames and at the last one when stderr is not a terminal", func(t *testing.T) {
		// given
		m := setUp(t, drawing(60, summary, nil))

		// when
		stdout, stderr, err := executeRenderAllWith(t, m, []cli.RenderOption{notTerminal}, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		require.NoError(t, err)
		assert.Equal(t, "Drawing frame 10/60 (16.7%), elapsed 00:00:04\n"+
			"Drawing frame 20/60 (33.3%), elapsed 00:00:08\n"+
			"Drawing frame 30/60 (50.0%), elapsed 00:00:12\n"+
			"Drawing frame 40/60 (66.7%), elapsed 00:00:16\n"+
			"Drawing frame 50/60 (83.3%), elapsed 00:00:20\n"+
			"Drawing frame 60/60 (100.0%), elapsed 00:00:24\n", stderr)
		assert.NotContains(t, stdout, "Drawing frame", "the progress is not part of the summary")
	})

	t.Run("should write a line at the last frame even when it is not a multiple of ten", func(t *testing.T) {
		// given
		m := setUp(t, drawing(13, summary, nil))

		// when
		_, stderr, err := executeRenderAllWith(t, m, []cli.RenderOption{notTerminal}, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		require.NoError(t, err)
		lines := strings.Split(strings.TrimSpace(stderr), "\n")
		require.Len(t, lines, 2)
		assert.Contains(t, lines[1], "Drawing frame 13/13 (100.0%)")
	})

	t.Run("should rewrite one line with a carriage return when stderr is a terminal, and end it", func(t *testing.T) {
		// given
		m := setUp(t, drawing(60, summary, nil))

		// when
		_, stderr, err := executeRenderAllWith(t, m, []cli.RenderOption{asTerminal}, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		require.NoError(t, err)
		assert.Contains(t, stderr, "\rDrawing frame 1/60 (1.7%), elapsed 00:00:00")
		assert.Contains(t, stderr, "\rDrawing frame 60/60 (100.0%), elapsed 00:00:24")
		assert.True(t, strings.HasSuffix(stderr, "\n"))
		assert.Equal(t, 1, strings.Count(stderr, "\n"), "a single line, rewritten")
	})

	t.Run("should write nothing to stderr when nothing was drawn", func(t *testing.T) {
		// given
		m := setUp(t, drawing(0, domain.RenderSummary{Requested: 60, Kept: 60, Resolution: defaultResolution}, nil))

		// when
		_, stderr, err := executeRenderAllWith(t, m, []cli.RenderOption{asTerminal}, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		require.NoError(t, err)
		assert.Empty(t, stderr)
	})

	t.Run("should say how many frames are ready, and how to go on, when the drawing was interrupted", func(t *testing.T) {
		// given
		interrupted := domain.RenderSummary{Requested: 1260, Drawn: 400, Kept: 12, Resolution: defaultResolution, Interrupted: true}
		m := setUp(t, drawing(0, interrupted, domain.ErrRenderInterrupted))

		// when
		stdout, _, err := executeRenderAllWith(t, m, []cli.RenderOption{notTerminal}, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		assert.ErrorIs(t, err, domain.ErrRenderInterrupted)
		assert.Equal(t, 38, cli.ExitCode(err))
		assert.True(t, strings.HasPrefix(stdout, "Interrupted: 412 of 1260 frames are ready; run the same command again to continue\n"), stdout)
		assert.Contains(t, stdout, "Frames: 1260 requested, 400 drawn, 12 kept (already in the destination)\n")
		assert.Contains(t, stdout, "Resolution: 1080x1920\n")
	})

	t.Run("should print what it did, and then fail, when a frame could not be written after others were", func(t *testing.T) {
		// given
		partial := domain.RenderSummary{Requested: 60, Drawn: 7, Resolution: defaultResolution}
		m := setUp(t, drawing(0, partial, domain.ErrFrameDestinationInvalid))

		// when
		stdout, _, err := executeRenderAllWith(t, m, []cli.RenderOption{notTerminal}, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		assert.Equal(t, 35, cli.ExitCode(err))
		assert.Contains(t, stdout, "Frames: 60 requested, 7 drawn, 0 kept (already in the destination)\n")
		assert.NotContains(t, stdout, "Interrupted")
	})

	t.Run("should count the frames kept in the summary", func(t *testing.T) {
		// given
		m := setUp(t, drawing(0, domain.RenderSummary{Requested: 60, Drawn: 48, Kept: 12, Resolution: defaultResolution}, nil))

		// when
		stdout, _, err := executeRenderAllWith(t, m, []cli.RenderOption{notTerminal}, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Frames: 60 requested, 48 drawn, 12 kept (already in the destination)\n")
	})
}
