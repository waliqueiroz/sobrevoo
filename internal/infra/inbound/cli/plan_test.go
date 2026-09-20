package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

// planDefaults are the defaults the composition root would inject: 30 fps,
// medium distance and tilt, and an automatic duration.
func planDefaults() domain.PlanParameters {
	return domain.PlanParameters{FrameRate: 30, Distance: domain.LevelMedium, Tilt: domain.LevelMedium}
}

// executePlanCommand runs the "plan" command against an existing temporary
// file, with cameraPlanService (a test double) as its only dependency, and
// returns stdout and the resulting error.
func executePlanCommand(t *testing.T, cameraPlanService application.CameraPlanService, extraArgs ...string) (stdout string, err error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "track.gpx")
	require.NoError(t, os.WriteFile(path, []byte("irrelevant, the service is mocked"), 0o600))

	cmd := cli.NewPlanCommand(cameraPlanService, planDefaults())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(append([]string{path}, extraArgs...))

	err = cmd.Execute()

	return out.String(), err
}

func Test_PlanCommand_Args(t *testing.T) {
	t.Run("should return a usage error when no file argument is given", func(t *testing.T) {
		// given
		cmd := cli.NewPlanCommand(nil, planDefaults())
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{})

		// when
		err := cmd.Execute()

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.Contains(t, err.Error(), "accepts exactly one file argument")
	})

	t.Run("should return a usage error when more than one file argument is given", func(t *testing.T) {
		// given
		cmd := cli.NewPlanCommand(nil, planDefaults())
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"a.gpx", "b.gpx"})

		// when
		err := cmd.Execute()

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error for an unknown flag", func(t *testing.T) {
		// given
		cmd := cli.NewPlanCommand(nil, planDefaults())
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"a.gpx", "--nonsense"})

		// when
		err := cmd.Execute()

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return the open error when the file does not exist", func(t *testing.T) {
		// given
		cmd := cli.NewPlanCommand(nil, planDefaults())
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{filepath.Join(t.TempDir(), "missing.gpx")})

		// when
		err := cmd.Execute()

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Equal(t, 4, cli.ExitCode(err))
	})
}

func Test_PlanCommand_Parameters(t *testing.T) {
	// expectParameters makes the mocked service assert the parameters it receives.
	expectParameters := func(t *testing.T, want domain.PlanParameters) application.CameraPlanService {
		t.Helper()
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		service.EXPECT().Generate(gomock.Any(), want).Return(builddomain.NewCameraPlanBuilder().Build(), nil)
		return service
	}

	t.Run("should pass the injected defaults, with an automatic duration, when no flag is given", func(t *testing.T) {
		// given
		service := expectParameters(t, planDefaults())

		// when
		_, err := executePlanCommand(t, service)

		// then
		assert.NoError(t, err)
	})

	t.Run("should pass a duration given in seconds, decimals included", func(t *testing.T) {
		// given
		want := planDefaults()
		want.Duration = new(45500 * time.Millisecond)
		service := expectParameters(t, want)

		// when
		_, err := executePlanCommand(t, service, "--duration", "45.5")

		// then
		assert.NoError(t, err)
	})

	t.Run("should pass a fractional frame rate", func(t *testing.T) {
		// given
		want := planDefaults()
		want.FrameRate = 29.97
		service := expectParameters(t, want)

		// when
		_, err := executePlanCommand(t, service, "--fps", "29.97")

		// then
		assert.NoError(t, err)
	})

	t.Run("should translate the distance and tilt levels", func(t *testing.T) {
		// given
		want := planDefaults()
		want.Distance = domain.LevelHigh
		want.Tilt = domain.LevelLow
		service := expectParameters(t, want)

		// when
		_, err := executePlanCommand(t, service, "--distance", "high", "--tilt", "low")

		// then
		assert.NoError(t, err)
	})

	t.Run("should use the injected defaults for the flags it does not receive", func(t *testing.T) {
		// given: defaults different from the usual ones
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		want := domain.PlanParameters{FrameRate: 24, Distance: domain.LevelLow, Tilt: domain.LevelHigh}
		service.EXPECT().Generate(gomock.Any(), want).Return(builddomain.NewCameraPlanBuilder().Build(), nil)

		path := filepath.Join(t.TempDir(), "track.gpx")
		require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))
		cmd := cli.NewPlanCommand(service, want)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{path})

		// when
		err := cmd.Execute()

		// then
		assert.NoError(t, err)
	})

	t.Run("should let a business-invalid duration and frame rate reach the service, which owns those rules", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(domain.CameraPlan{}, domain.ErrInvalidDuration)
		service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(domain.CameraPlan{}, domain.ErrInvalidFrameRate)

		// when
		_, durationErr := executePlanCommand(t, service, "--duration", "0")
		_, rateErr := executePlanCommand(t, service, "--fps", "200")

		// then
		assert.Equal(t, 10, cli.ExitCode(durationErr))
		assert.Equal(t, 11, cli.ExitCode(rateErr))
	})
}

func Test_PlanCommand_UsageErrors(t *testing.T) {
	// no expectations: the service must never be called for these
	newService := func(t *testing.T) application.CameraPlanService {
		return mockapplication.NewMockCameraPlanService(gomock.NewController(t))
	}

	for name, args := range map[string][]string{
		"a non-numeric duration": {"--duration", "abc"},
		"a NaN duration":         {"--duration", "nan"},
		"an infinite duration":   {"--duration", "inf"},
		"an empty duration":      {"--duration", ""},
		"a non-numeric fps":      {"--fps", "abc"},
		"a NaN fps":              {"--fps", "NaN"},
		"an unknown distance":    {"--distance", "perto"},
		"an unknown tilt":        {"--tilt", "perto"},
	} {
		t.Run("should return a usage error for "+name, func(t *testing.T) {
			// when
			_, err := executePlanCommand(t, newService(t), args...)

			// then
			require.Error(t, err)
			assert.Equal(t, 2, cli.ExitCode(err))
			assert.Contains(t, err.Error(), args[0])
		})
	}

	t.Run("should list the accepted levels when a level is unknown", func(t *testing.T) {
		// when
		_, err := executePlanCommand(t, newService(t), "--distance", "perto")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "low, medium, high")
	})

	t.Run("should return a usage error for --overwrite without --export", func(t *testing.T) {
		// when
		_, err := executePlanCommand(t, newService(t), "--overwrite")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.Contains(t, err.Error(), "--overwrite requires --export")
	})

	t.Run("should saturate an absurdly large duration and let the domain refuse it", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		service.EXPECT().Generate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, parameters domain.PlanParameters) (domain.CameraPlan, error) {
			require.NotNil(t, parameters.Duration)
			return domain.CameraPlan{}, parameters.Validate()
		}).Times(2)

		// when
		_, huge := executePlanCommand(t, service, "--duration", "1e30")
		_, hugeNegative := executePlanCommand(t, service, "--duration", "-1e30")

		// then
		assert.Equal(t, 10, cli.ExitCode(huge))
		assert.Equal(t, 10, cli.ExitCode(hugeNegative))
	})
}

func Test_PlanCommand_Execute(t *testing.T) {
	t.Run("should print the summary of the plan in the contract's format", func(t *testing.T) {
		// given
		frames := []domain.CameraFrame{
			builddomain.NewCameraFrameBuilder().WithCameraAltitude(127.3).WithCameraToMarkerDistance(300).Build(),
			builddomain.NewCameraFrameBuilder().WithCameraAltitude(912.8).WithCameraToMarkerDistance(1204.5).Build(),
		}
		plan := builddomain.NewCameraPlanBuilder().
			WithParameters(builddomain.NewPlanParametersBuilder().WithDuration(42*time.Second).WithFrameRate(30).Build()).
			WithDurationMode(domain.DurationModeAutomatic).
			WithFrames(frames...).
			WithSmoothedSpans(
				domain.SmoothedSpan{Start: 0, End: 1200 * time.Millisecond, Quantity: domain.QuantityHeading},
				domain.SmoothedSpan{Start: 31400 * time.Millisecond, End: 32100 * time.Millisecond, Quantity: domain.QuantityZoom},
			).Build()
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)

		// when
		stdout, err := executePlanCommand(t, service)

		// then
		require.NoError(t, err)
		assert.Equal(t, "Duration: 42.0 s (automatic)\n"+
			"Frame rate: 30.0 fps\n"+
			"Frames: 2\n"+
			"Camera altitude: 127.3 m - 912.8 m\n"+
			"Camera distance: 300.0 m - 1204.5 m\n"+
			"Time reference: clock\n"+
			"Smoothed spans: 2\n"+
			"  0.00 s - 1.20 s (heading)\n"+
			"  31.40 s - 32.10 s (zoom)\n", stdout)
	})

	t.Run("should mark a duration chosen by the user as requested", func(t *testing.T) {
		// given
		plan := builddomain.NewCameraPlanBuilder().WithDurationMode(domain.DurationModeExplicit).Build()
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)

		// when
		stdout, err := executePlanCommand(t, service)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "(requested)")
	})

	t.Run("should say explicitly that no span had to be smoothed", func(t *testing.T) {
		// given
		plan := builddomain.NewCameraPlanBuilder().Build()
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)

		// when
		stdout, err := executePlanCommand(t, service)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Smoothed spans: none\n")
		assert.NotContains(t, stdout, "Smoothed spans: 0")
	})

	t.Run("should show the reason when the distance was used instead of the clock", func(t *testing.T) {
		// given
		plan := builddomain.NewCameraPlanBuilder().WithTimeReference(domain.TimeReferenceDistance, "time data is inconsistent").Build()
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)

		// when
		stdout, err := executePlanCommand(t, service)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Time reference: distance (time data is inconsistent)\n")
	})

	t.Run("should not write anything to disk without --export", func(t *testing.T) {
		// given: the mock has no expectation for Export
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(builddomain.NewCameraPlanBuilder().Build(), nil)

		// when
		stdout, err := executePlanCommand(t, service)

		// then
		require.NoError(t, err)
		assert.NotContains(t, stdout, "Plan written")
	})

	t.Run("should propagate the service's errors unchanged, leaving code translation to ExitCode", func(t *testing.T) {
		// given
		cases := map[error]int{
			domain.ErrEmptyFile:          1,
			domain.ErrInsufficientPoints: 3,
			domain.ErrDurationTooShort:   12,
			domain.ErrTrackTooShort:      13,
			domain.ErrTrackTooLarge:      14,
			errors.New("boom"):           4,
		}

		for wantErr, code := range cases {
			mockCtrl := gomock.NewController(t)
			service := mockapplication.NewMockCameraPlanService(mockCtrl)
			service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(domain.CameraPlan{}, wantErr)

			// when
			stdout, err := executePlanCommand(t, service)

			// then
			assert.ErrorIs(t, err, wantErr)
			assert.Equal(t, code, cli.ExitCode(err))
			assert.Empty(t, stdout)
		}
	})
}

func Test_PlanCommand_Export(t *testing.T) {
	plan := builddomain.NewCameraPlanBuilder().Build()

	t.Run("should export the plan and print where it was written, as the last line", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		service.EXPECT().Export(plan, "out/plan.json", false).Return(nil)

		// when
		stdout, err := executePlanCommand(t, service, "--export", "out/plan.json")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Smoothed spans")
		assert.Regexp(t, `Plan written to out/plan\.json\n$`, stdout)
	})

	t.Run("should ask the service to overwrite when --overwrite is given", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		service.EXPECT().Export(plan, "plan.json", true).Return(nil)

		// when
		_, err := executePlanCommand(t, service, "--export", "plan.json", "--overwrite")

		// then
		assert.NoError(t, err)
	})

	t.Run("should propagate export errors unchanged", func(t *testing.T) {
		// given
		cases := map[error]int{
			domain.ErrPlanDestinationExists:  15,
			domain.ErrPlanDestinationInvalid: 16,
		}

		for wantErr, code := range cases {
			mockCtrl := gomock.NewController(t)
			service := mockapplication.NewMockCameraPlanService(mockCtrl)
			service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
			service.EXPECT().Export(plan, "plan.json", false).Return(wantErr)

			// when
			stdout, err := executePlanCommand(t, service, "--export", "plan.json")

			// then
			assert.ErrorIs(t, err, wantErr)
			assert.Equal(t, code, cli.ExitCode(err))
			assert.Empty(t, stdout, "an export error comes with no summary")
		}
	})

	t.Run("should not export when generating the plan fails", func(t *testing.T) {
		// given: the mock has no expectation for Export
		mockCtrl := gomock.NewController(t)
		service := mockapplication.NewMockCameraPlanService(mockCtrl)
		service.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(domain.CameraPlan{}, domain.ErrTrackTooShort)

		// when
		_, err := executePlanCommand(t, service, "--export", "plan.json")

		// then
		assert.ErrorIs(t, err, domain.ErrTrackTooShort)
	})
}
