package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

// flightDefaults is the same shape of defaults "plan"/"render all"/"video"
// already use.
func flightDefaults() (domain.PlanParameters, domain.Resolution, domain.Appearance, domain.OverlayConfig, domain.VideoQuality) {
	return domain.PlanParameters{FrameRate: 30, Distance: domain.LevelMedium, Tilt: domain.LevelMedium, Simplification: domain.LevelMedium, Smoothing: domain.LevelMedium, Aspect: domain.AspectRatio{Width: 9, Height: 16}},
		domain.Resolution{Width: 1080, Height: 1920},
		domain.Appearance{
			TrailColor: domain.RGB{R: 0xFF, G: 0xB0, B: 0x00}, TrailWidthRatio: 0.005,
			MarkerColor: domain.RGB{R: 0xE5, G: 0x25, B: 0x2A}, MarkerRadiusRatio: 0.012,
			BackgroundColor: domain.RGB{R: 0x20, G: 0x26, B: 0x2E},
		},
		domain.OverlayConfig{Enabled: true, Distance: true, Elevation: true, Time: true, Profile: true},
		domain.VideoQualityMedium
}

func newFlightCommandMocks(t *testing.T) *mockapplication.MockFlightService {
	t.Helper()
	return mockapplication.NewMockFlightService(gomock.NewController(t))
}

func executeFlyCommand(t *testing.T, flightService *mockapplication.MockFlightService, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	parameters, resolution, appearance, overlay, quality := flightDefaults()
	cmd := cli.NewFlightCommand(flightService, parameters, resolution, appearance, overlay, quality)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), errOut.String(), err
}

// aTrackFile writes a placeholder track file (the service is mocked, so its
// content is irrelevant) and returns its path.
func aTrackFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "track.gpx")
	require.NoError(t, os.WriteFile(path, []byte("irrelevant, the service is mocked"), 0o600))
	return path
}

// aFlight is what the summary of a whole run of 380 frames looks like.
func aFlight() domain.FlightSummary {
	return domain.FlightSummary{
		Completed:       []domain.FlightStage{domain.StageTrackProcessing, domain.StageCameraPlanning, domain.StageGeoDataSlicing, domain.StageFrameRendering, domain.StageVideoEncoding},
		FramesDirectory: "/tmp/sobrevoo-fly-1",
		Render: domain.RenderSummary{
			Requested: 380, Drawn: 380, Resolution: domain.Resolution{Width: 1080, Height: 1920},
		},
		Video: domain.VideoSummary{
			Frames: 380, FrameRate: 30, Resolution: domain.Resolution{Width: 1080, Height: 1920},
			Quality: domain.VideoQualityMedium, Encoder: domain.EncoderInfo{Name: "ffmpeg", Version: "7.1", Codec: "libx264"},
			SizeBytes: 1234,
		},
		Elapsed: 65 * time.Second,
	}
}

func Test_FlightCommand_Args(t *testing.T) {
	t.Run("should return a usage error when there is no track file", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, err := executeFlyCommand(t, m, "--output", "flight.mp4")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error when there is one argument too many", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, err := executeFlyCommand(t, m, "track.gpx", "extra", "--output", "flight.mp4")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error, without asking for anything, when --output is missing", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t))

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.ErrorContains(t, err, "--output is required")
	})

	t.Run("should refuse an output that does not end in .mp4 as a usage error", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mkv")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.EqualError(t, err, "--output must end in .mp4: the video is always an MP4 file")
	})

	t.Run("should return a usage error for an unknown flag", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--nonsense")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}

func Test_FlightCommand_Parameters(t *testing.T) {
	t.Run("should pass the duration, fps, distance, tilt and aspect flags to the request's parameters, the same way plan parses them", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			p := x.(domain.FlightRequest).Parameters
			return p.Duration != nil && *p.Duration == 45*time.Second &&
				p.FrameRate == 24 && p.Distance == domain.LevelHigh && p.Tilt == domain.LevelLow &&
				p.Aspect == domain.AspectRatio{Width: 16, Height: 9}
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4",
			"--duration", "45", "--fps", "24", "--distance", "high", "--tilt", "low", "--aspect", "16:9")

		// then
		require.NoError(t, err)
	})

	t.Run("should pass the simplification and smoothing flags to the request's parameters, the same way plan parses them", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			p := x.(domain.FlightRequest).Parameters
			return p.Simplification == domain.LevelHigh && p.Smoothing == domain.LevelLow
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4",
			"--simplification", "high", "--smoothing", "low")

		// then
		require.NoError(t, err)
	})

	t.Run("should use the given defaults when no parameter flag is given", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		parameters, _, _, _, _ := flightDefaults()
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.FlightRequest).Parameters == parameters
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.NoError(t, err)
	})

	t.Run("should refuse an invalid distance with the same usage error plan already gives, without flying anything", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--distance", "ultra")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should refuse an invalid simplification or smoothing with the same usage error plan already gives, without flying anything", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, simplificationErr := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--simplification", "ultra")
		_, _, smoothingErr := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--smoothing", "ultra")

		// then
		require.Error(t, simplificationErr)
		assert.Equal(t, 2, cli.ExitCode(simplificationErr))
		require.Error(t, smoothingErr)
		assert.Equal(t, 2, cli.ExitCode(smoothingErr))
	})

	t.Run("should pass the resolution flag to the request, the same way render all parses it", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), domain.FlightRequest{
			Parameters: func() domain.PlanParameters { p, _, _, _, _ := flightDefaults(); return p }(),
			Resolution: domain.Resolution{Width: 640, Height: 360},
			Appearance: func() domain.Appearance { _, _, a, _, _ := flightDefaults(); return a }(),
			Overlay:    func() domain.OverlayConfig { _, _, _, o, _ := flightDefaults(); return o }(),
			Quality:    domain.VideoQualityMedium,
			Output:     "flight.mp4",
		}, gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--resolution", "640x360")

		// then
		require.NoError(t, err)
	})

	t.Run("should refuse an invalid resolution with the same error render all already gives", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--resolution", "3x3")

		// then
		require.Error(t, err)
		assert.Equal(t, 34, cli.ExitCode(err))
	})

	t.Run("should pass the quality flag to the request, the same way video parses it", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.FlightRequest).Quality == domain.VideoQualityHigh
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--quality", "high")

		// then
		require.NoError(t, err)
	})

	t.Run("should refuse an unknown quality with the same usage error video already gives", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--quality", "ultra")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.EqualError(t, err, `--quality "ultra": use one of low, medium, high`)
	})
}

func Test_FlightCommand_Appearance(t *testing.T) {
	t.Run("should use the given defaults when no appearance flag is given", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		_, _, appearance, _, _ := flightDefaults()
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.FlightRequest).Appearance == appearance
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.NoError(t, err)
	})

	t.Run("should pass the five appearance values given, all at once, to the request", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		want := domain.Appearance{
			TrailColor: domain.RGB{R: 0x00, G: 0xFF, B: 0x00}, TrailWidthRatio: 0.02,
			MarkerColor: domain.RGB{R: 0x00, G: 0x00, B: 0xFF}, MarkerRadiusRatio: 0.05,
			BackgroundColor: domain.RGB{R: 0xFF, G: 0xFF, B: 0xFF},
		}
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.FlightRequest).Appearance == want
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4",
			"--trail-color", "#00FF00", "--trail-width", "0.02", "--marker-color", "#0000FF", "--marker-radius", "0.05", "--background-color", "#FFFFFF")

		// then
		require.NoError(t, err)
	})

	t.Run("should refuse a malformed color with the same error render frame already gives, without flying anything", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--background-color", "sky")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
		assert.Equal(t, 52, cli.ExitCode(err))
	})

	t.Run("should refuse a trail width outside the documented range with the same error render frame already gives", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--trail-width", "1")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidTrailWidth)
		assert.Equal(t, 53, cli.ExitCode(err))
	})
}

func Test_FlightCommand_Overlay(t *testing.T) {
	t.Run("should use the given defaults when no overlay flag is given", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		_, _, _, overlay, _ := flightDefaults()
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.FlightRequest).Overlay == overlay
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.NoError(t, err)
	})

	t.Run("should turn on only the blocks named in --overlay-blocks, passed to the request", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		want, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{domain.OverlayBlockTime})
		require.NoError(t, err)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.FlightRequest).Overlay == want
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err = executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--overlay-blocks=time")

		// then
		require.NoError(t, err)
	})

	t.Run("should refuse an unknown overlay block with the same error render frame already gives, without flying anything", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--overlay-blocks", "altitude")

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidOverlayBlock)
		assert.Equal(t, 55, cli.ExitCode(err))
	})
}

func Test_FlightCommand_Selection(t *testing.T) {
	t.Run("should pass an empty SourceSelection when neither flag is given", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.FlightRequest).Selection == domain.SourceSelection{}
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.NoError(t, err)
	})

	t.Run("should pass the requested base map and elevation names as a SourceSelection", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		baseMap, elevation := "mapa-b", "relevo-a"
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			got := x.(domain.FlightRequest).Selection
			return got.BaseMapName != nil && *got.BaseMapName == baseMap &&
				got.ElevationName != nil && *got.ElevationName == elevation
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--base-map", baseMap, "--elevation", elevation)

		// then
		require.NoError(t, err)
	})

	t.Run("should map a requested source of the wrong type to the same exit code geodata slice already gives", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.FlightSummary{}, domain.ErrDataSourceTypeMismatch)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--base-map", "relevo-a")

		// then
		assert.ErrorIs(t, err, domain.ErrDataSourceTypeMismatch)
		assert.Equal(t, 56, cli.ExitCode(err))
	})
}

func Test_FlightCommand_Execute(t *testing.T) {
	t.Run("should open the track file and fly it, with the default parameters, resolution and quality, without overwriting", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		parameters, resolution, appearance, overlay, quality := flightDefaults()
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), domain.FlightRequest{
			Parameters: parameters, Resolution: resolution, Appearance: appearance, Overlay: overlay, Quality: quality, Output: "flight.mp4", Overwrite: false,
		}, gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.NoError(t, err)
	})

	t.Run("should return the error of opening the track file as it is, without flying anything", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		missing := filepath.Join(t.TempDir(), "missing.gpx")

		// when
		_, _, err := executeFlyCommand(t, m, missing, "--output", "flight.mp4")

		// then
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("should print the frames summary, then the video summary, then the total time", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(aFlight(), nil)

		// when
		stdout, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4")

		// then: no destination line, since the temporary directory of the
		// frames is already gone
		require.NoError(t, err)
		assert.Equal(t, "Frames: 380 requested, 380 drawn, 0 kept (already in the destination)\n"+
			"Resolution: 1080x1920\n"+
			"Time: 00:00:00\n"+
			"Holes (in the frames drawn now): none\n"+
			"Video written to flight.mp4\n"+
			"Frames: 380\n"+
			"Duration: 00:00:12.667\n"+
			"Resolution: 1080x1920\n"+
			"Frame rate: 30 fps\n"+
			"Quality: medium\n"+
			"Size: 1.2 KiB\n"+
			"Encoder: ffmpeg 7.1 (libx264)\n"+
			"Time: 00:00:00\n"+
			"Total time: 00:01:05\n", stdout)
	})

	t.Run("should print where the frames are kept, with --keep", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		flight := aFlight()
		flight.FramesDirectory = "/tmp/kept/frames"
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(flight, nil)

		// when
		stdout, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--keep", "/tmp/kept")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Holes (in the frames drawn now): none\n"+
			"Destination: /tmp/kept/frames (frame_000000.png to frame_000379.png)\n"+
			"Video written to flight.mp4\n")
	})

	t.Run("should point to another --keep, not another --output, for frames of another set under --keep", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		conflict := fmt.Errorf("%w: 380 frames are of another set; use --overwrite to replace them", domain.ErrFrameSetConflict)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.FlightSummary{}, conflict)

		// when
		stdout, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--keep", "/tmp/kept")

		// then
		assert.Equal(t, 37, cli.ExitCode(err))
		assert.EqualError(t, err, "frame destination holds frames of another set: 380 frames are of another set; use --overwrite to replace them, or another --keep")
		assert.Empty(t, stdout)
	})

	t.Run("should return the error of the run as it is, and print no summary", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		failure := errors.New("the registered geo data does not cover the whole area the plan needs")
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.FlightSummary{}, failure)

		// when
		stdout, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.ErrorIs(t, err, failure)
		assert.Empty(t, stdout)
	})

	t.Run("should ask to overwrite with --overwrite", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.FlightRequest).Overwrite
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--overwrite")

		// then
		assert.NoError(t, err)
	})

	t.Run("should pass the --keep directory to the request", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.FlightRequest).Keep == "/tmp/kept"
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4", "--keep", "/tmp/kept")

		// then
		assert.NoError(t, err)
	})

	t.Run("should leave the request's Keep empty when --keep is not given", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.FlightRequest).Keep == ""
		}), gomock.Any()).Return(aFlight(), nil)

		// when
		_, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4")

		// then
		assert.NoError(t, err)
	})
}

func Test_FlightCommand_Interrupted(t *testing.T) {
	t.Run("should say which stages completed and return the error", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		summary := domain.FlightSummary{
			Completed:   []domain.FlightStage{domain.StageTrackProcessing, domain.StageCameraPlanning, domain.StageGeoDataSlicing},
			Interrupted: true,
		}
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, domain.ErrFlightInterrupted)

		// when
		stdout, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.ErrorIs(t, err, domain.ErrFlightInterrupted)
		assert.Equal(t, 51, cli.ExitCode(err))
		assert.Equal(t, "Interrupted after stage 3/5 (slicing the geo data); completed: treating the track, planning the camera, slicing the geo data\n", stdout)
	})

	t.Run("should say the partial frames drawn when interrupted while drawing", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		summary := domain.FlightSummary{
			Completed:   []domain.FlightStage{domain.StageTrackProcessing, domain.StageCameraPlanning, domain.StageGeoDataSlicing},
			Render:      domain.RenderSummary{Requested: 380, Drawn: 210, Interrupted: true},
			Interrupted: true,
		}
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, domain.ErrFlightInterrupted)

		// when
		stdout, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.ErrorIs(t, err, domain.ErrFlightInterrupted)
		assert.Contains(t, stdout, "Frames: 210 of 380 drawn\n")
	})

	t.Run("should say nothing completed when interrupted before the plan exists", func(t *testing.T) {
		// given
		m := newFlightCommandMocks(t)
		m.EXPECT().Fly(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.FlightSummary{Interrupted: true}, domain.ErrFlightInterrupted)

		// when
		stdout, _, err := executeFlyCommand(t, m, aTrackFile(t), "--output", "flight.mp4")

		// then
		require.ErrorIs(t, err, domain.ErrFlightInterrupted)
		assert.Equal(t, "Interrupted before any stage completed\n", stdout)
	})
}
