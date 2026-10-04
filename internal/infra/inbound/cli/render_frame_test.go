package cli_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

var defaultResolution = domain.Resolution{Width: 1080, Height: 1920}
var defaultAppearance = builddomain.NewAppearanceBuilder().Build()
var defaultOverlay = builddomain.NewOverlayConfigBuilder().Build()

type renderCommandMocks struct {
	planService  *mockapplication.MockCameraPlanService
	sliceService *mockapplication.MockGeoSliceService
	frameService *mockapplication.MockFrameService
}

func newRenderCommandMocks(t *testing.T) renderCommandMocks {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	return renderCommandMocks{
		planService:  mockapplication.NewMockCameraPlanService(mockCtrl),
		sliceService: mockapplication.NewMockGeoSliceService(mockCtrl),
		frameService: mockapplication.NewMockFrameService(mockCtrl),
	}
}

// planOfFrames is a plan of n frames for a vertical video, the shape of the
// default resolution.
func planOfFrames(n int) domain.CameraPlan {
	frames := make([]domain.CameraFrame, n)
	for i := range frames {
		frames[i] = builddomain.NewCameraFrameBuilder().WithIndex(i).Build()
	}
	parameters := builddomain.NewPlanParametersBuilder().WithAspect(domain.AspectRatio{Width: 9, Height: 16}).Build()
	return builddomain.NewCameraPlanBuilder().WithParameters(parameters).WithFrames(frames...).Build()
}

func executeRenderFrameCommand(t *testing.T, m renderCommandMocks, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd := cli.NewRenderFrameCommand(m.planService, m.sliceService, m.frameService, defaultResolution, defaultAppearance, defaultOverlay)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), errOut.String(), err
}

func Test_RenderFrameCommand_Args(t *testing.T) {
	t.Run("should return a usage error when the number of arguments is not two", func(t *testing.T) {
		for _, args := range [][]string{
			{"--number", "1", "--output", "f.png"},
			{"plan.json", "--number", "1", "--output", "f.png"},
			{"plan.json", "slice.zip", "extra", "--number", "1", "--output", "f.png"},
		} {
			// given
			m := newRenderCommandMocks(t)

			// when
			_, _, err := executeRenderFrameCommand(t, m, args...)

			// then
			require.Error(t, err)
			assert.Equal(t, 2, cli.ExitCode(err), "%v", args)
		}
	})

	t.Run("should return a usage error when --number or --output is missing", func(t *testing.T) {
		for _, args := range [][]string{
			{"plan.json", "slice.zip", "--output", "f.png"},
			{"plan.json", "slice.zip", "--number", "1"},
			{"plan.json", "slice.zip"},
		} {
			// given
			m := newRenderCommandMocks(t)

			// when
			_, _, err := executeRenderFrameCommand(t, m, args...)

			// then
			require.Error(t, err)
			assert.Equal(t, 2, cli.ExitCode(err), "%v", args)
		}
	})

	t.Run("should return a usage error for an unknown flag", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "1", "--output", "f.png", "--nonsense")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}

func Test_RenderFrameCommand_Execute(t *testing.T) {
	plan := planOfFrames(1260)
	slice := builddomain.NewGeoSliceBuilder().Build()

	t.Run("should load the plan, then the slice, and draw the frame at the default resolution, printing the summary", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		gomock.InOrder(
			m.planService.EXPECT().Load("plan.json").Return(plan, nil),
			m.sliceService.EXPECT().Load("slice.zip").Return(slice, nil),
			m.frameService.EXPECT().DrawFrame(gomock.Any(), plan, slice, domain.SingleFrameRequest{
				Number: 300, Path: "/tmp/q.png", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: defaultOverlay, Overwrite: false,
			}).Return(domain.RenderSummary{Requested: 1, Drawn: 1, Resolution: defaultResolution, Elapsed: 2 * time.Second}, nil),
		)

		// when
		stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "300", "--output", "/tmp/q.png")

		// then
		require.NoError(t, err)
		assert.Equal(t, "Frame 300 of 1260 drawn to /tmp/q.png\n"+
			"Resolution: 1080x1920\n"+
			"Time: 00:00:02\n"+
			"Holes: map tiles missing: no, elevation without value: no\n", stdout)
	})

	// summaryOfHoles runs the command over a frame whose summary is the given one.
	summaryOfHoles := func(t *testing.T, summary domain.RenderSummary) string {
		t.Helper()
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, nil)

		stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "0", "--output", "f.png")

		require.NoError(t, err)
		return stdout
	}

	t.Run("should not warn when the resolution is wider than the plan's video", func(t *testing.T) {
		// given: a 9:16 plan drawn at a horizontal resolution
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.RenderSummary{Requested: 1, Drawn: 1}, nil)

		// when
		_, stderr, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "0", "--output", "f.png", "--resolution", "1920x1080")

		// then: a wider image only sees more, nothing is cut
		require.NoError(t, err)
		assert.Empty(t, stderr)
	})

	t.Run("should warn when a horizontal plan is drawn at a vertical resolution", func(t *testing.T) {
		// given
		horizontalPlan := builddomain.NewCameraPlanBuilder().
			WithParameters(builddomain.NewPlanParametersBuilder().WithAspect(domain.LandscapeAspectRatio).Build()).
			WithFrames(builddomain.NewCameraFrameBuilder().Build()).Build()
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(horizontalPlan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.RenderSummary{Requested: 1, Drawn: 1}, nil)

		// when
		stdout, stderr, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "0", "--output", "f.png")

		// then
		require.NoError(t, err)
		assert.Contains(t, stderr, "Warning: the plan was made for a 16:9 video, but the resolution is 1080x1920")
		assert.Contains(t, stdout, "Frame 0 of 1 drawn")
	})

	t.Run("should say a frame had a hole of the map", func(t *testing.T) {
		// given / when
		stdout := summaryOfHoles(t, domain.RenderSummary{Drawn: 1, MapHoleFrames: 1})

		// then
		assert.Contains(t, stdout, "Holes: map tiles missing: yes, elevation without value: no\n")
	})

	t.Run("should say a frame had a hole of the elevation", func(t *testing.T) {
		// given / when
		stdout := summaryOfHoles(t, domain.RenderSummary{Drawn: 1, ElevationHoleFrames: 1})

		// then
		assert.Contains(t, stdout, "Holes: map tiles missing: no, elevation without value: yes\n")
	})

	t.Run("should say a frame had holes of both kinds", func(t *testing.T) {
		// given / when
		stdout := summaryOfHoles(t, domain.RenderSummary{Drawn: 1, MapHoleFrames: 1, ElevationHoleFrames: 1})

		// then
		assert.Contains(t, stdout, "Holes: map tiles missing: yes, elevation without value: yes\n")
	})

	t.Run("should write the elapsed time as hours, minutes and seconds", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(domain.RenderSummary{Drawn: 1, Elapsed: time.Hour + 2*time.Minute + 3*time.Second + 600*time.Millisecond}, nil)

		// when
		stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "0", "--output", "f.png")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Time: 01:02:04\n")
	})

	t.Run("should refuse a number that is not a whole number of the plan before it reads the slice", func(t *testing.T) {
		for _, number := range []string{"3.5", "abc", "1260", "-1"} {
			// given: the slice service has no expectation, so reading the slice fails the test
			m := newRenderCommandMocks(t)
			m.planService.EXPECT().Load("plan.json").Return(plan, nil)

			// when
			stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", number, "--output", "f.png")

			// then
			assert.ErrorIs(t, err, domain.ErrFrameOutOfRange, number)
			assert.Equal(t, 33, cli.ExitCode(err))
			assert.Empty(t, stdout)
		}
	})

	t.Run("should not read the slice or draw when the plan cannot be loaded", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load("plan.json").Return(domain.CameraPlan{}, domain.ErrPlanFileInvalid)

		// when
		stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "1", "--output", "f.png")

		// then
		assert.Equal(t, 17, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should not draw when the slice cannot be loaded", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load("slice.zip").Return(domain.GeoSlice{}, domain.ErrSliceFormatVersionUnsupported)

		// when
		stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "1", "--output", "f.png")

		// then
		assert.Equal(t, 28, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	// drawFails runs the command over a drawing that fails with err.
	drawFails := func(t *testing.T, want error) (stdout string, err error) {
		t.Helper()
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.RenderSummary{}, want)

		stdout, _, err = executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "1", "--output", "f.png")
		return stdout, err
	}

	t.Run("should return a slice of another plan with the exit code 29, printing nothing", func(t *testing.T) {
		// given / when
		stdout, err := drawFails(t, domain.ErrSliceDoesNotMatchPlan)

		// then
		assert.Equal(t, 29, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should return an invalid slice with the exit code 27, printing nothing", func(t *testing.T) {
		// given / when
		stdout, err := drawFails(t, domain.ErrSliceFileInvalid)

		// then
		assert.Equal(t, 27, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should return an invalid destination with the exit code 35, printing nothing", func(t *testing.T) {
		// given / when
		stdout, err := drawFails(t, domain.ErrFrameDestinationInvalid)

		// then
		assert.Equal(t, 35, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should return any other error with the generic exit code 4, printing nothing", func(t *testing.T) {
		// given / when
		stdout, err := drawFails(t, errors.New("something else"))

		// then
		assert.Equal(t, 4, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should give the context of the command to the drawing", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		var got context.Context
		m.frameService.EXPECT().DrawFrame(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, _ domain.CameraPlan, _ domain.GeoSlice, _ domain.SingleFrameRequest) (domain.RenderSummary, error) {
				got = ctx
				return domain.RenderSummary{Drawn: 1}, nil
			})

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "1", "--output", "f.png")

		// then
		require.NoError(t, err)
		assert.NotNil(t, got)
	})
}

func Test_RenderFrameCommand_Resolution(t *testing.T) {
	plan := planOfFrames(1260)
	slice := builddomain.NewGeoSliceBuilder().Build()

	t.Run("should draw at the resolution asked for, and say it in the summary", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		want := domain.Resolution{Width: 960, Height: 540}
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), plan, slice, domain.SingleFrameRequest{Number: 3, Path: "f.png", Resolution: want, Appearance: defaultAppearance, Overlay: defaultOverlay}).
			Return(domain.RenderSummary{Drawn: 1, Resolution: want}, nil)

		// when
		stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--resolution", "960x540")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Resolution: 960x540\n")
	})

	// refuses runs the command with a resolution that is not acceptable: no service is used.
	refuses := func(t *testing.T, resolution string) {
		t.Helper()
		m := newRenderCommandMocks(t) // no expectations: any call to a service fails the test

		stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--resolution="+resolution)

		assert.ErrorIs(t, err, domain.ErrInvalidResolution)
		assert.Equal(t, 34, cli.ExitCode(err))
		assert.Empty(t, stdout)
	}

	t.Run("should refuse an odd resolution before reading anything", func(t *testing.T) { refuses(t, "1921x1080") })
	t.Run("should refuse a resolution over the limits before reading anything", func(t *testing.T) { refuses(t, "8000x4500") })
	t.Run("should refuse text that is not a resolution before reading anything", func(t *testing.T) { refuses(t, "abc") })
	t.Run("should refuse a resolution with no height before reading anything", func(t *testing.T) { refuses(t, "960") })
	t.Run("should refuse a zero resolution before reading anything", func(t *testing.T) { refuses(t, "0x0") })
	t.Run("should refuse a negative resolution before reading anything", func(t *testing.T) { refuses(t, "-960x540") })
}

func Test_RenderFrameCommand_Overwrite(t *testing.T) {
	plan := planOfFrames(1260)
	slice := builddomain.NewGeoSliceBuilder().Build()

	t.Run("should hand --overwrite to the drawing", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), plan, slice, domain.SingleFrameRequest{Number: 3, Path: "f.png", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: defaultOverlay, Overwrite: true}).
			Return(domain.RenderSummary{Drawn: 1, Resolution: defaultResolution}, nil)

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--overwrite")

		// then
		require.NoError(t, err)
	})

	t.Run("should return a file that exists with the exit code 36, saying how to replace it", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		exists := fmt.Errorf("%w: f.png (use --overwrite to replace it)", domain.ErrFrameDestinationExists)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.RenderSummary{}, exists)

		// when
		stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png")

		// then
		assert.Equal(t, 36, cli.ExitCode(err))
		assert.ErrorContains(t, err, "use --overwrite to replace it")
		assert.Empty(t, stdout)
	})
}

func Test_RenderFrameCommand_Appearance(t *testing.T) {
	plan := planOfFrames(1260)
	slice := builddomain.NewGeoSliceBuilder().Build()

	t.Run("should draw with the default appearance when no appearance flag is given", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), plan, slice, domain.SingleFrameRequest{Number: 3, Path: "f.png", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: defaultOverlay}).
			Return(domain.RenderSummary{Drawn: 1}, nil)

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png")

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
		m.frameService.EXPECT().DrawFrame(gomock.Any(), plan, slice, domain.SingleFrameRequest{Number: 3, Path: "f.png", Resolution: defaultResolution, Appearance: want, Overlay: defaultOverlay}).
			Return(domain.RenderSummary{Drawn: 1}, nil)

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png",
			"--trail-color", "#00FF00", "--trail-width", "0.02", "--marker-color", "#0000FF", "--marker-radius", "0.05", "--background-color", "#FFFFFF")

		// then
		require.NoError(t, err)
	})

	t.Run("should refuse a malformed trail color with ErrInvalidColor, not a usage error", func(t *testing.T) {
		// given: no service has an expectation, so any call fails the test
		m := newRenderCommandMocks(t)

		// when
		stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--trail-color", "green")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
		assert.Equal(t, 52, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should refuse a malformed marker color with ErrInvalidColor", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--marker-color", "#FFF")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
		assert.Equal(t, 52, cli.ExitCode(err))
	})

	t.Run("should refuse a malformed background color with ErrInvalidColor", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--background-color", "#GGGGGG")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
		assert.Equal(t, 52, cli.ExitCode(err))
	})

	t.Run("should refuse a trail width that is not a number with a usage error, not ErrInvalidTrailWidth", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)

		// when
		stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--trail-width", "abc")

		// then
		assert.NotErrorIs(t, err, domain.ErrInvalidTrailWidth)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should refuse a trail width outside the documented range with ErrInvalidTrailWidth", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--trail-width", "1")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidTrailWidth)
		assert.Equal(t, 53, cli.ExitCode(err))
	})

	t.Run("should refuse a marker radius that is not a number with a usage error", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--marker-radius", "abc")

		// then
		assert.NotErrorIs(t, err, domain.ErrInvalidMarkerRadius)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should refuse a marker radius outside the documented range with ErrInvalidMarkerRadius", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--marker-radius", "-1")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidMarkerRadius)
		assert.Equal(t, 54, cli.ExitCode(err))
	})
}

func Test_RenderFrameCommand_Overlay(t *testing.T) {
	plan := planOfFrames(1260)
	slice := builddomain.NewGeoSliceBuilder().Build()

	t.Run("should draw with the default overlay configuration when no overlay flag is given", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), plan, slice, domain.SingleFrameRequest{Number: 3, Path: "f.png", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: defaultOverlay}).
			Return(domain.RenderSummary{Drawn: 1}, nil)

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png")

		// then
		require.NoError(t, err)
	})

	t.Run("should turn every block off with --overlays=false", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), plan, slice, domain.SingleFrameRequest{Number: 3, Path: "f.png", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: domain.OverlayConfig{}}).
			Return(domain.RenderSummary{Drawn: 1}, nil)

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--overlays=false")

		// then
		require.NoError(t, err)
	})

	t.Run("should turn on only the blocks named in --overlay-blocks", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		want, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{domain.OverlayBlockDistance, domain.OverlayBlockTime})
		require.NoError(t, err)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), plan, slice, domain.SingleFrameRequest{Number: 3, Path: "f.png", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: want}).
			Return(domain.RenderSummary{Drawn: 1}, nil)

		// when
		_, _, err = executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--overlay-blocks=distance,time")

		// then
		require.NoError(t, err)
	})

	t.Run("should turn on the speed block when named in --overlay-blocks, even though it is not in the default list", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		want, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{domain.OverlayBlockDistance, domain.OverlayBlockSpeed})
		require.NoError(t, err)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), plan, slice, domain.SingleFrameRequest{Number: 3, Path: "f.png", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: want}).
			Return(domain.RenderSummary{Drawn: 1}, nil)

		// when
		_, _, err = executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--overlay-blocks=distance,speed")

		// then
		require.NoError(t, err)
	})

	t.Run("should turn on the gain block when named in --overlay-blocks, independent of elevation", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		want, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{domain.OverlayBlockGain})
		require.NoError(t, err)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		m.frameService.EXPECT().DrawFrame(gomock.Any(), plan, slice, domain.SingleFrameRequest{Number: 3, Path: "f.png", Resolution: defaultResolution, Appearance: defaultAppearance, Overlay: want}).
			Return(domain.RenderSummary{Drawn: 1}, nil)

		// when
		_, _, err = executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--overlay-blocks=gain")

		// then
		require.NoError(t, err)
	})

	t.Run("should refuse an unknown overlay block with ErrInvalidOverlayBlock before calling the frame service", func(t *testing.T) {
		// given: no service has an expectation, so any call fails the test
		m := newRenderCommandMocks(t)

		// when
		stdout, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "3", "--output", "f.png", "--overlay-blocks=altitude")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidOverlayBlock)
		assert.Equal(t, 55, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})
}
