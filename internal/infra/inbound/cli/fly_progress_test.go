package cli_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

// executeFlyCommandWith runs "fly" with options for the command.
func executeFlyCommandWith(t *testing.T, flightService *mockapplication.MockFlightService, options []cli.RenderOption, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	parameters, resolution, appearance, overlay, quality := flightDefaults()
	cmd := cli.NewFlightCommand(flightService, parameters, resolution, appearance, overlay, quality, options...)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), errOut.String(), err
}

// flying makes the flight service report the five stages, with frame progress
// during the fourth and video progress during the fifth, and end with summary.
func flying(summary domain.FlightSummary, result error) func(context.Context, io.Reader, domain.FlightRequest, func(domain.FlightProgress)) (domain.FlightSummary, error) {
	return func(_ context.Context, _ io.Reader, _ domain.FlightRequest, progress func(domain.FlightProgress)) (domain.FlightSummary, error) {
		if progress == nil {
			return summary, result
		}
		progress(domain.FlightProgress{Stage: domain.StageTrackProcessing})
		progress(domain.FlightProgress{Stage: domain.StageCameraPlanning})
		progress(domain.FlightProgress{Stage: domain.StageGeoDataSlicing})
		progress(domain.FlightProgress{Stage: domain.StageFrameRendering})
		render := domain.RenderProgress{Done: 10, Total: 20, Elapsed: 4 * time.Second}
		progress(domain.FlightProgress{Stage: domain.StageFrameRendering, Render: &render})
		progress(domain.FlightProgress{Stage: domain.StageVideoEncoding})
		video := domain.VideoProgress{Done: 15, Total: 20, Elapsed: 6 * time.Second}
		progress(domain.FlightProgress{Stage: domain.StageVideoEncoding, Video: &video})
		return summary, result
	}
}

func Test_FlightCommand_Progress(t *testing.T) {
	notTerminal := cli.WithTerminalCheck(func(io.Writer) bool { return false })

	t.Run("should write one stage announcement line for each of the five stages, in order", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(flying(aFlight(), nil))

		// when
		_, stderr, err := executeFlyCommandWith(t, m, []cli.RenderOption{notTerminal}, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.NoError(t, err)
		assert.Contains(t, stderr, "Stage 1/5: treating the track\n")
		assert.Contains(t, stderr, "Stage 2/5: planning the camera\n")
		assert.Contains(t, stderr, "Stage 3/5: slicing the geo data\n")
		assert.Contains(t, stderr, "Stage 4/5: drawing the frames\n")
		assert.Contains(t, stderr, "Stage 5/5: encoding the video\n")
	})

	t.Run("should forward the frame-rendering progress in the same format render all already uses", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(flying(aFlight(), nil))

		// when
		_, stderr, err := executeFlyCommandWith(t, m, []cli.RenderOption{notTerminal}, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.NoError(t, err)
		assert.Contains(t, stderr, "Drawing frame 10/20 (50.0%), elapsed 00:00:04\n")
	})

	t.Run("should forward the video-encoding progress in the same format video already uses", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(flying(aFlight(), nil))

		// when
		_, stderr, err := executeFlyCommandWith(t, m, []cli.RenderOption{notTerminal}, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.NoError(t, err)
		assert.Contains(t, stderr, "Encoding frame 15/20 (75.0%), elapsed 00:00:06\n")
	})

	t.Run("should say, on a line under the stage's announcement, that the plan and the slice were reused", func(t *testing.T) {
		// given: each stage is announced as it starts, and the reuse comes after
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, _ io.Reader, _ domain.FlightRequest, progress func(domain.FlightProgress)) (domain.FlightSummary, error) {
				progress(domain.FlightProgress{Stage: domain.StageTrackProcessing})
				progress(domain.FlightProgress{Stage: domain.StageCameraPlanning})
				progress(domain.FlightProgress{Stage: domain.StageCameraPlanning, Reused: true})
				progress(domain.FlightProgress{Stage: domain.StageGeoDataSlicing})
				progress(domain.FlightProgress{Stage: domain.StageGeoDataSlicing, Reused: true})
				progress(domain.FlightProgress{Stage: domain.StageFrameRendering})
				return aFlight(), nil
			})

		// when
		_, stderr, err := executeFlyCommandWith(t, m, []cli.RenderOption{notTerminal}, aTrackFile(t), "--output", "flight.mp4", "--keep", "/tmp/kept")

		// then
		require.NoError(t, err)
		assert.Equal(t, "Stage 1/5: treating the track\n"+
			"Stage 2/5: planning the camera\n"+
			"  unchanged since the last run under --keep, reusing plan.json\n"+
			"Stage 3/5: slicing the geo data\n"+
			"  unchanged, reusing slice.zip\n"+
			"Stage 4/5: drawing the frames\n", stderr)
	})

	t.Run("should not say anything was reused when it was not", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(flying(aFlight(), nil))

		// when
		_, stderr, err := executeFlyCommandWith(t, m, []cli.RenderOption{notTerminal}, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.NoError(t, err)
		assert.Contains(t, stderr, "Stage 2/5: planning the camera\n")
		assert.NotContains(t, stderr, "unchanged")
	})

	t.Run("should not mix the stage announcements into stdout", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(flying(aFlight(), nil))

		// when
		stdout, _, err := executeFlyCommandWith(t, m, []cli.RenderOption{notTerminal}, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.NoError(t, err)
		assert.NotContains(t, stdout, "Stage")
	})
}
