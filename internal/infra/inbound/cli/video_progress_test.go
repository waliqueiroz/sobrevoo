package cli_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

// executeVideoWith runs "video" with options for the command.
func executeVideoWith(t *testing.T, m videoCommandMocks, options []cli.RenderOption, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	var cmd *cobra.Command = cli.NewVideoCommand(m.planService, m.videoService, domain.VideoQualityMedium, options...)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), errOut.String(), err
}

// encoding makes the video service report the given progress and end with the
// summary.
func encoding(reports []domain.VideoProgress, summary domain.VideoSummary, result error) func(context.Context, domain.CameraPlan, domain.VideoRequest, func(domain.VideoProgress)) (domain.VideoSummary, error) {
	return func(_ context.Context, _ domain.CameraPlan, _ domain.VideoRequest, progress func(domain.VideoProgress)) (domain.VideoSummary, error) {
		for _, report := range reports {
			progress(report)
		}
		return summary, result
	}
}

func Test_VideoCommand_Progress(t *testing.T) {
	plan := planOfFrames(380)
	asTerminal := cli.WithTerminalCheck(func(io.Writer) bool { return true })
	notTerminal := cli.WithTerminalCheck(func(io.Writer) bool { return false })

	// every half second, 25 frames at a time, up to 380
	var reports []domain.VideoProgress
	for done, elapsed := 25, 500*time.Millisecond; done < 380; done, elapsed = done+25, elapsed+500*time.Millisecond {
		reports = append(reports, domain.VideoProgress{Done: done, Total: 380, Elapsed: elapsed})
	}
	reports = append(reports, domain.VideoProgress{Done: 380, Total: 380, Elapsed: 8 * time.Second})

	setUp := func(t *testing.T, reports []domain.VideoProgress, result error) videoCommandMocks {
		t.Helper()
		m := newVideoCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(encoding(reports, aVideo(), result))
		return m
	}

	t.Run("should rewrite one line on stderr with each report when stderr is a terminal, and end it", func(t *testing.T) {
		// given
		m := setUp(t, []domain.VideoProgress{
			{Done: 152, Total: 380, Elapsed: 3 * time.Second},
			{Done: 380, Total: 380, Elapsed: 8 * time.Second},
		}, nil)

		// when
		stdout, stderr, err := executeVideoWith(t, m, []cli.RenderOption{asTerminal}, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.NoError(t, err)
		assert.Equal(t, "\rEncoding frame 152/380 (40.0%), elapsed 00:00:03"+
			"\rEncoding frame 380/380 (100.0%), elapsed 00:00:08\n", stderr)
		assert.NotContains(t, stdout, "Encoding frame", "the progress is not part of the summary")
	})

	t.Run("should write a line only every five seconds, and at the last report, when stderr is not a terminal", func(t *testing.T) {
		// given
		m := setUp(t, reports, nil)

		// when
		_, stderr, err := executeVideoWith(t, m, []cli.RenderOption{notTerminal}, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.NoError(t, err)
		assert.Equal(t, "Encoding frame 250/380 (65.8%), elapsed 00:00:05\n"+
			"Encoding frame 380/380 (100.0%), elapsed 00:00:08\n", stderr)
	})

	t.Run("should write nothing before five seconds have passed, when stderr is not a terminal", func(t *testing.T) {
		// given
		m := setUp(t, []domain.VideoProgress{
			{Done: 50, Total: 380, Elapsed: time.Second},
			{Done: 100, Total: 380, Elapsed: 4 * time.Second},
		}, nil)

		// when
		_, stderr, err := executeVideoWith(t, m, []cli.RenderOption{notTerminal}, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.NoError(t, err)
		assert.Empty(t, stderr)
	})

	t.Run("should write a line at the last report even if it comes before five seconds", func(t *testing.T) {
		// given
		m := setUp(t, []domain.VideoProgress{{Done: 380, Total: 380, Elapsed: 2 * time.Second}}, nil)

		// when
		_, stderr, err := executeVideoWith(t, m, []cli.RenderOption{notTerminal}, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.NoError(t, err)
		assert.Equal(t, "Encoding frame 380/380 (100.0%), elapsed 00:00:02\n", stderr)
	})

	t.Run("should write nothing on stderr when nothing was reported, on a terminal or not", func(t *testing.T) {
		// given
		m := setUp(t, nil, nil)
		other := setUp(t, nil, nil)

		// when
		_, onTerminal, errTerminal := executeVideoWith(t, m, []cli.RenderOption{asTerminal}, "plan.json", "frames", "--output", "flight.mp4")
		_, offTerminal, errLog := executeVideoWith(t, other, []cli.RenderOption{notTerminal}, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.NoError(t, errTerminal)
		require.NoError(t, errLog)
		assert.Empty(t, onTerminal)
		assert.Empty(t, offTerminal)
	})

	t.Run("should end the line of a terminal when the assembly fails, and print no summary", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(encoding([]domain.VideoProgress{{Done: 100, Total: 380, Elapsed: time.Second}}, domain.VideoSummary{}, domain.ErrVideoEncodingFailed))

		// when
		stdout, stderr, err := executeVideoWith(t, m, []cli.RenderOption{asTerminal}, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.ErrorIs(t, err, domain.ErrVideoEncodingFailed)
		assert.Equal(t, "\rEncoding frame 100/380 (26.3%), elapsed 00:00:01\n", stderr)
		assert.Empty(t, stdout)
	})
}
