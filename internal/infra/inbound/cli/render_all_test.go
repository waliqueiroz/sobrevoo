package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

func executeRenderAllCommand(t *testing.T, m renderCommandMocks, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd := cli.NewRenderAllCommand(m.planService, m.sliceService, m.frameService, defaultResolution, defaultAppearance, defaultOverlay)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), errOut.String(), err
}

func Test_RenderAllCommand_Args(t *testing.T) {
	t.Run("should return a usage error when the number of arguments is not two", func(t *testing.T) {
		for _, args := range [][]string{
			{"--output", "dir"},
			{"plan.json", "--output", "dir"},
			{"plan.json", "slice.zip", "extra", "--output", "dir"},
		} {
			// given
			m := newRenderCommandMocks(t)

			// when
			_, _, err := executeRenderAllCommand(t, m, args...)

			// then
			require.Error(t, err)
			assert.Equal(t, 2, cli.ExitCode(err), "%v", args)
		}
	})

	t.Run("should return a usage error when --output is missing", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)

		// when
		_, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error for an unknown flag", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)

		// when
		_, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "dir", "--nonsense")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}

func Test_RenderAllCommand_Execute(t *testing.T) {
	plan := planOfFrames(60)
	slice := builddomain.NewGeoSliceBuilder().Build()

	t.Run("should load the plan and the slice, and draw all the frames at the default resolution, printing the summary", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		gomock.InOrder(
			m.planService.EXPECT().Load("plan.json").Return(plan, nil),
			m.sliceService.EXPECT().Load("slice.zip").Return(slice, nil),
			m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, domain.FrameSetRequest{
				Directory: "/tmp/q", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: defaultOverlay, Overwrite: false,
			}, gomock.Any()).Return(domain.RenderSummary{
				Requested: 60, Drawn: 60, Resolution: defaultResolution, Elapsed: 31*time.Minute + 7*time.Second,
			}, nil),
		)

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		require.NoError(t, err)
		assert.Equal(t, "Frames: 60 requested, 60 drawn, 0 kept (already in the destination)\n"+
			"Resolution: 1080x1920\n"+
			"Time: 00:31:07\n"+
			"Holes (in the frames drawn now): none\n"+
			"Destination: /tmp/q (frame_000000.png to frame_000059.png)\n", stdout)
	})

	t.Run("should not read the slice or draw when the plan cannot be loaded", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load("plan.json").Return(domain.CameraPlan{}, domain.ErrPlanFormatVersionUnsupported)

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		assert.Equal(t, 18, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should not draw when the slice cannot be loaded", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load("slice.zip").Return(domain.GeoSlice{}, domain.ErrSliceFileInvalid)

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		assert.Equal(t, 27, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should return the error of drawing with its exit code, printing nothing when no frame was drawn", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(domain.RenderSummary{Requested: 60}, domain.ErrFrameDestinationInvalid)

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		assert.Equal(t, 35, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should return any other error with the generic exit code", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(domain.RenderSummary{}, errors.New("disk full"))

		// when
		_, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		assert.Equal(t, 4, cli.ExitCode(err))
	})

	// summaryOf runs the command over a drawing whose summary is the given one.
	summaryOf := func(t *testing.T, summary domain.RenderSummary) string {
		t.Helper()
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, nil)

		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		require.NoError(t, err)
		return stdout
	}

	t.Run("should count the frames with holes by cause", func(t *testing.T) {
		// given / when
		stdout := summaryOf(t, domain.RenderSummary{Requested: 60, Drawn: 60, MapHoleFrames: 87, ElevationHoleFrames: 4, Resolution: defaultResolution})

		// then
		assert.Contains(t, stdout, "Holes (in the frames drawn now): 87 with missing map tiles, 4 with elevation without value\n")
	})

	t.Run("should count a cause that did not happen as zero when the other did", func(t *testing.T) {
		// given / when
		stdout := summaryOf(t, domain.RenderSummary{Requested: 60, Drawn: 60, ElevationHoleFrames: 3, Resolution: defaultResolution})

		// then
		assert.Contains(t, stdout, "Holes (in the frames drawn now): 0 with missing map tiles, 3 with elevation without value\n")
	})

	t.Run("should say there was no hole when frames were drawn and none had one", func(t *testing.T) {
		// given / when
		stdout := summaryOf(t, domain.RenderSummary{Requested: 60, Drawn: 60, Resolution: defaultResolution})

		// then
		assert.Contains(t, stdout, "Holes (in the frames drawn now): none\n")
	})

	t.Run("should say no frame was drawn now, and how many were kept, when every frame was there already", func(t *testing.T) {
		// given / when
		stdout := summaryOf(t, domain.RenderSummary{Requested: 60, Kept: 60, Resolution: defaultResolution})

		// then
		assert.Contains(t, stdout, "Frames: 60 requested, 0 drawn, 60 kept (already in the destination)\n")
		assert.Contains(t, stdout, "Holes (in the frames drawn now): none drawn now\n")
	})
}

func Test_RenderAllCommand_Resolution(t *testing.T) {
	plan := planOfFrames(60)
	slice := builddomain.NewGeoSliceBuilder().Build()

	t.Run("should draw at the resolution asked for, and say it in the summary", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		want := domain.Resolution{Width: 3840, Height: 2160}
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, domain.FrameSetRequest{Directory: "/tmp/q", Resolution: want, Appearance: defaultAppearance, Overlay: defaultOverlay}, gomock.Any()).
			Return(domain.RenderSummary{Requested: 60, Drawn: 60, Resolution: want}, nil)

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q", "--resolution", "3840x2160")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Resolution: 3840x2160\n")
	})

	t.Run("should refuse an invalid resolution before reading anything", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t) // no expectations: any call to a service fails the test

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q", "--resolution", "1921x1080")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidResolution)
		assert.Equal(t, 34, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})
}

func Test_RenderAllCommand_Overwrite(t *testing.T) {
	plan := planOfFrames(60)
	slice := builddomain.NewGeoSliceBuilder().Build()

	t.Run("should hand --overwrite to the drawing", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, domain.FrameSetRequest{Directory: "/tmp/q", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: defaultOverlay, Overwrite: true}, gomock.Any()).
			Return(domain.RenderSummary{Requested: 60, Drawn: 60, Resolution: defaultResolution}, nil)

		// when
		_, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q", "--overwrite")

		// then
		require.NoError(t, err)
	})

	t.Run("should say how many frames of a previous set it removed", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(domain.RenderSummary{Requested: 60, Drawn: 60, Removed: 20, Resolution: defaultResolution}, nil)

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q", "--overwrite")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Removed: 20 frames from a previous set\n")
	})

	t.Run("should not mention removals when there were none", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(domain.RenderSummary{Requested: 60, Drawn: 60, Resolution: defaultResolution}, nil)

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		require.NoError(t, err)
		assert.NotContains(t, stdout, "Removed")
	})

	t.Run("should return a directory of another set with the exit code 37, printing nothing", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		conflict := fmt.Errorf("%w: 60 frames are of another set; use --overwrite to replace them, or another --output", domain.ErrFrameSetConflict)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.RenderSummary{Requested: 60}, conflict)

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		assert.Equal(t, 37, cli.ExitCode(err))
		assert.ErrorContains(t, err, "use --overwrite to replace them, or another --output")
		assert.Empty(t, stdout)
	})
}

func Test_RenderAllCommand_Appearance(t *testing.T) {
	plan := planOfFrames(60)
	slice := builddomain.NewGeoSliceBuilder().Build()

	t.Run("should draw with the default appearance when no appearance flag is given", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, domain.FrameSetRequest{Directory: "/tmp/q", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: defaultOverlay}, gomock.Any()).
			Return(domain.RenderSummary{Requested: 60, Drawn: 60}, nil)

		// when
		_, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		require.NoError(t, err)
	})

	t.Run("should draw with the five appearance values given, all at once", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		want := domain.Appearance{
			TrailColor: domain.RGB{R: 0x00, G: 0xFF, B: 0x00}, TrailWidthRatio: 0.02,
			MarkerColor: domain.RGB{R: 0x00, G: 0x00, B: 0xFF}, MarkerRadiusRatio: 0.05,
			BackgroundColor: domain.RGB{R: 0xFF, G: 0xFF, B: 0xFF},
		}
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, domain.FrameSetRequest{Directory: "/tmp/q", Resolution: defaultResolution, Appearance: want, Overlay: defaultOverlay}, gomock.Any()).
			Return(domain.RenderSummary{Requested: 60, Drawn: 60}, nil)

		// when
		_, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q",
			"--trail-color", "#00FF00", "--trail-width", "0.02", "--marker-color", "#0000FF", "--marker-radius", "0.05", "--background-color", "#FFFFFF")

		// then
		require.NoError(t, err)
	})

	t.Run("should refuse a malformed color before reading anything, with the same error as render frame", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t) // no expectations: any call to a service fails the test

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q", "--marker-color", "blue")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
		assert.Equal(t, 52, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should refuse a marker radius outside the documented range, with the same error as render frame", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)

		// when
		_, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q", "--marker-radius", "0.5")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidMarkerRadius)
		assert.Equal(t, 54, cli.ExitCode(err))
	})
}

func Test_RenderAllCommand_Overlay(t *testing.T) {
	plan := planOfFrames(60)
	slice := builddomain.NewGeoSliceBuilder().Build()

	t.Run("should draw with the default overlay configuration when no overlay flag is given", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, domain.FrameSetRequest{Directory: "/tmp/q", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: defaultOverlay}, gomock.Any()).
			Return(domain.RenderSummary{Requested: 60, Drawn: 60}, nil)

		// when
		_, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		require.NoError(t, err)
	})

	t.Run("should turn on only the blocks named in --overlay-blocks", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		want, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{domain.OverlayBlockProfile})
		require.NoError(t, err)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, domain.FrameSetRequest{Directory: "/tmp/q", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: want}, gomock.Any()).
			Return(domain.RenderSummary{Requested: 60, Drawn: 60}, nil)

		// when
		_, _, err = executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q", "--overlay-blocks=profile")

		// then
		require.NoError(t, err)
	})

	t.Run("should refuse an unknown overlay block before reading anything, with the same error as render frame", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q", "--overlay-blocks=altitude")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidOverlayBlock)
		assert.Equal(t, 55, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})
}

func Test_RenderCommands_Refusals(t *testing.T) {
	plan := planOfFrames(60)
	slice := builddomain.NewGeoSliceBuilder().Build()

	// drawFailsWith runs "render all" over a drawing that fails with err.
	drawFailsWith := func(t *testing.T, err error) (string, error) {
		t.Helper()
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.RenderSummary{Requested: 60}, err)

		stdout, _, got := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")
		return stdout, got
	}

	t.Run("should return a slice that does not cover the plan with the exit code 30", func(t *testing.T) {
		// given / when
		stdout, err := drawFailsWith(t, fmt.Errorf("%w: the plan needs lat -23.5 to -23.4", domain.ErrSliceDoesNotCoverPlan))

		// then
		assert.Equal(t, 30, cli.ExitCode(err))
		assert.ErrorContains(t, err, "the plan needs")
		assert.Empty(t, stdout)
	})

	t.Run("should return vector tiles with the exit code 31, saying the base map", func(t *testing.T) {
		// given / when
		stdout, err := drawFailsWith(t, fmt.Errorf("%w: base map %q has vector tiles (pbf)", domain.ErrTileFormatUnsupported, "bbbike"))

		// then
		assert.Equal(t, 31, cli.ExitCode(err))
		assert.ErrorContains(t, err, `base map "bbbike" has vector tiles (pbf)`)
		assert.Empty(t, stdout)
	})

	t.Run("should return a slice with no elevation with the exit code 32", func(t *testing.T) {
		// given / when
		stdout, err := drawFailsWith(t, domain.ErrNoElevationData)

		// then
		assert.Equal(t, 32, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should return a slice of another plan with the exit code 29", func(t *testing.T) {
		// given / when
		_, err := drawFailsWith(t, domain.ErrSliceDoesNotMatchPlan)

		// then
		assert.Equal(t, 29, cli.ExitCode(err))
	})

	t.Run("should return a plan that is not valid with the exit code 17, before reading the slice", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load("plan.json").Return(domain.CameraPlan{}, fmt.Errorf("%w: the plan has no frames", domain.ErrPlanFileInvalid))

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		assert.Equal(t, 17, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should return a file that cannot be read with the generic exit code", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load("slice.zip").Return(domain.GeoSlice{}, errors.New("reading the slice file: no such file or directory"))

		// when
		stdout, _, err := executeRenderAllCommand(t, m, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		assert.Equal(t, 4, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})
}
