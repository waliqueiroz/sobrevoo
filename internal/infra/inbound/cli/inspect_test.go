package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/application/mock_application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/build_domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

// executeInspectCommand runs the "inspect" command against an existing
// temporary file, with inspectTrackService as its only dependency, and
// returns stdout and the resulting error. The service is a test double —
// this is a unit test of the CLI adapter alone (Constitution Principle III),
// never a real track parser/simplifier/smoother.
func executeInspectCommand(t *testing.T, inspectTrackService application.InspectTrackService, extraArgs ...string) (stdout string, err error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "track.gpx")
	require.NoError(t, os.WriteFile(path, []byte("irrelevant, the service is mocked"), 0o600))

	cmd := cli.NewInspectCommand(inspectTrackService, domain.LevelMedium)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(append([]string{path}, extraArgs...))

	err = cmd.Execute()

	return out.String(), err
}

func Test_InspectCommand_Args(t *testing.T) {
	t.Run("should return a usage error when no file argument is given", func(t *testing.T) {
		// given
		cmd := cli.NewInspectCommand(nil, domain.LevelMedium)
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
		cmd := cli.NewInspectCommand(nil, domain.LevelMedium)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"a.gpx", "b.gpx"})

		// when
		err := cmd.Execute()

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}

func Test_InspectCommand_Execute(t *testing.T) {
	t.Run("should print every field of the summary when the service succeeds", func(t *testing.T) {
		// given
		summary := build_domain.NewTrackSummaryBuilder().Build()

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, nil)

		// when
		stdout, err := executeInspectCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, cli.ExitCode(err))
		assert.Contains(t, stdout, "Format: GPX")
		assert.Contains(t, stdout, "Points: 10 -> 8 (original -> treated)")
		assert.Contains(t, stdout, "Distance: 1.00 km")
		assert.Contains(t, stdout, "Elevation gain: 50.0 m")
		assert.Contains(t, stdout, "Duration: 10m0s")
		assert.Contains(t, stdout, "Bounding box:")
		assert.Contains(t, stdout, "Discarded points:")
	})

	t.Run("should indicate elevation gain is unavailable instead of printing a computed value", func(t *testing.T) {
		// given
		summary := build_domain.NewTrackSummaryBuilder().WithoutElevationGainMeters().Build()

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, nil)

		// when
		stdout, err := executeInspectCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Elevation gain: not available (no altitude data)")
	})

	t.Run("should indicate duration is unavailable instead of printing a computed value", func(t *testing.T) {
		// given
		summary := build_domain.NewTrackSummaryBuilder().WithoutDuration().Build()

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, nil)

		// when
		stdout, err := executeInspectCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Duration: not available (no time data)")
	})

	t.Run("should indicate the bounding box crosses the antimeridian", func(t *testing.T) {
		// given
		boundingBox := domain.BoundingBox{
			MinLongitude:        179.9,
			MaxLongitude:        -179.9,
			CrossesAntimeridian: true,
		}
		summary := build_domain.NewTrackSummaryBuilder().WithBoundingBox(boundingBox).Build()

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, nil)

		// when
		stdout, err := executeInspectCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "(crosses the antimeridian)")
	})

	t.Run("should print the discarded point counts broken down by reason", func(t *testing.T) {
		// given
		discarded := domain.DiscardStats{ImpossibleCoordinates: 1, ConsecutiveDuplicates: 2, ImplausibleJumps: 3}
		summary := build_domain.NewTrackSummaryBuilder().WithDiscarded(discarded).Build()

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, nil)

		// when
		stdout, err := executeInspectCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Discarded points: 6 (impossible coordinates: 1, consecutive duplicates: 2, implausible jumps: 3)")
	})

	t.Run("should not print any summary when the service fails", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.TrackSummary{}, domain.ErrEmptyFile)

		// when
		stdout, err := executeInspectCommand(t, mockedService)

		// then
		require.Error(t, err)
		assert.Empty(t, stdout)
	})

	t.Run("should map ErrEmptyFile to exit code 1", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.TrackSummary{}, domain.ErrEmptyFile)

		// when
		_, err := executeInspectCommand(t, mockedService)

		// then
		require.ErrorIs(t, err, domain.ErrEmptyFile)
		assert.Equal(t, 1, cli.ExitCode(err))
	})

	t.Run("should map ErrUnsupportedFormat to exit code 2", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.TrackSummary{}, domain.ErrUnsupportedFormat)

		// when
		_, err := executeInspectCommand(t, mockedService)

		// then
		require.ErrorIs(t, err, domain.ErrUnsupportedFormat)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should map ErrInsufficientPoints to exit code 3", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.TrackSummary{}, domain.ErrInsufficientPoints)

		// when
		_, err := executeInspectCommand(t, mockedService)

		// then
		require.ErrorIs(t, err, domain.ErrInsufficientPoints)
		assert.Equal(t, 3, cli.ExitCode(err))
	})

	t.Run("should map ErrInsufficientPointsAfterCleaning to exit code 3", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.TrackSummary{}, domain.ErrInsufficientPointsAfterCleaning)

		// when
		_, err := executeInspectCommand(t, mockedService)

		// then
		require.ErrorIs(t, err, domain.ErrInsufficientPointsAfterCleaning)
		assert.Equal(t, 3, cli.ExitCode(err))
	})

	t.Run("should map an unrecognized error to the generic exit code 4", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.TrackSummary{}, errors.New("boom"))

		// when
		_, err := executeInspectCommand(t, mockedService)

		// then
		require.Error(t, err)
		assert.Equal(t, 4, cli.ExitCode(err))
	})

	t.Run("should map a missing file to the generic exit code 4 without calling the service", func(t *testing.T) {
		// given: the mock has no EXPECT(), so any call to it fails the test
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)

		cmd := cli.NewInspectCommand(mockedService, domain.LevelMedium)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{filepath.Join(t.TempDir(), "does-not-exist.gpx")})

		// when
		err := cmd.Execute()

		// then
		require.Error(t, err)
		assert.Equal(t, 4, cli.ExitCode(err))
	})

	t.Run("should convert the --simplification and --smoothing flags into the levels passed to the service", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), domain.LevelHigh, domain.LevelLow).
			Return(build_domain.NewTrackSummaryBuilder().Build(), nil)

		// when
		_, err := executeInspectCommand(t, mockedService, "--simplification=high", "--smoothing=low")

		// then
		require.NoError(t, err)
	})

	t.Run("should use the given default level when no flag is provided", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "track.gpx")
		require.NoError(t, os.WriteFile(path, []byte("irrelevant"), 0o600))

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)
		mockedService.EXPECT().Inspect(gomock.Any(), domain.LevelHigh, domain.LevelHigh).
			Return(build_domain.NewTrackSummaryBuilder().Build(), nil)

		cmd := cli.NewInspectCommand(mockedService, domain.LevelHigh)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{path})

		// when
		err := cmd.Execute()

		// then
		require.NoError(t, err)
	})

	t.Run("should return a usage error for an invalid --simplification value without calling the service", func(t *testing.T) {
		// given: the mock has no EXPECT(), so any call to it fails the test
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)

		// when
		_, err := executeInspectCommand(t, mockedService, "--simplification=bogus")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error for an invalid --smoothing value without calling the service", func(t *testing.T) {
		// given: the mock has no EXPECT(), so any call to it fails the test
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockInspectTrackService(mockCtrl)

		// when
		_, err := executeInspectCommand(t, mockedService, "--smoothing=bogus")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}
