package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/application/mock_application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

// executeGeoDataCheckCommand runs the "geodata check" command against an
// existing temporary file, with checkCoverageService as its only
// dependency, and returns stdout and the resulting error. The service is a
// test double — this is a unit test of the CLI adapter alone (Constitution
// Principle III), never a real CheckCoverageService.
func executeGeoDataCheckCommand(t *testing.T, checkCoverageService application.CheckCoverageService) (stdout string, err error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "track.gpx")
	require.NoError(t, os.WriteFile(path, []byte("irrelevant, the service is mocked"), 0o600))

	cmd := cli.NewGeoDataCheckCommand(checkCoverageService)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{path})

	err = cmd.Execute()

	return out.String(), err
}

func Test_GeoDataCheckCommand_Args(t *testing.T) {
	t.Run("should return a usage error when no file argument is given", func(t *testing.T) {
		// given
		cmd := cli.NewGeoDataCheckCommand(nil)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{})

		// when
		err := cmd.Execute()

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}

func Test_GeoDataCheckCommand_Execute(t *testing.T) {
	t.Run("should report full coverage with the sources used", func(t *testing.T) {
		// given
		output := application.CheckCoverageOutput{
			Status:               application.CoverageStatusFull,
			BaseMapSourcesUsed:   []domain.GeoDataSource{{Name: "europa-mapa"}},
			ElevationSourcesUsed: []domain.GeoDataSource{{Name: "europa-relevo"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockCheckCoverageService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, cli.ExitCode(err))
		assert.Contains(t, stdout, "Coverage: full")
		assert.Contains(t, stdout, "europa-mapa")
		assert.Contains(t, stdout, "europa-relevo")
	})

	t.Run("should report a missing-elevation verdict when only a base map covers the track", func(t *testing.T) {
		// given
		output := application.CheckCoverageOutput{
			Status: application.CoverageStatusPartial,
			UncoveredSegments: []application.UncoveredSegment{
				{StartLatitude: 45, StartLongitude: 15, EndLatitude: 46, EndLongitude: 16, Missing: application.MissingElevation},
			},
			BaseMapSourcesUsed: []domain.GeoDataSource{{Name: "europa-mapa"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockCheckCoverageService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Coverage: partial")
		assert.Contains(t, stdout, "missing elevation")
	})

	t.Run("should report a missing-base-map verdict when only an elevation source covers the track", func(t *testing.T) {
		// given
		output := application.CheckCoverageOutput{
			Status: application.CoverageStatusPartial,
			UncoveredSegments: []application.UncoveredSegment{
				{StartLatitude: 45, StartLongitude: 15, EndLatitude: 46, EndLongitude: 16, Missing: application.MissingBaseMap},
			},
			ElevationSourcesUsed: []domain.GeoDataSource{{Name: "europa-relevo"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockCheckCoverageService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Coverage: partial")
		assert.Contains(t, stdout, "missing base map")
	})

	t.Run("should print the uncovered segment's start and end coordinates", func(t *testing.T) {
		// given
		output := application.CheckCoverageOutput{
			Status: application.CoverageStatusPartial,
			UncoveredSegments: []application.UncoveredSegment{
				{StartLatitude: 60, StartLongitude: 30, EndLatitude: 61, EndLongitude: 31, Missing: application.MissingBoth},
			},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockCheckCoverageService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "60.000000")
		assert.Contains(t, stdout, "30.000000")
	})

	t.Run("should print the specific source chosen when two base map sources overlap", func(t *testing.T) {
		// given: the resolution itself (smaller area wins) is verified at
		// the application layer (check_coverage_service_test.go) — this
		// only checks the CLI surfaces whichever source the service picked.
		output := application.CheckCoverageOutput{
			Status:             application.CoverageStatusFull,
			BaseMapSourcesUsed: []domain.GeoDataSource{{Name: "regiao-especifica"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockCheckCoverageService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "regiao-especifica")
	})

	t.Run("should report full coverage for a track crossing the antimeridian without special-casing it", func(t *testing.T) {
		// given: antimeridian handling itself is verified at the
		// application layer (check_coverage_service_test.go) and in
		// BoundingBox.Contains/AreaDegrees (bounding_box_test.go).
		output := application.CheckCoverageOutput{
			Status:               application.CoverageStatusFull,
			BaseMapSourcesUsed:   []domain.GeoDataSource{{Name: "antimeridiano-mapa"}},
			ElevationSourcesUsed: []domain.GeoDataSource{{Name: "antimeridiano-relevo"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockCheckCoverageService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Coverage: full")
	})

	t.Run("should not list a registered source whose file no longer exists among the sources used", func(t *testing.T) {
		// given: excluding the source itself is verified at the
		// application layer (check_coverage_service_test.go) — this only
		// checks the CLI never prints a source the service did not report.
		output := application.CheckCoverageOutput{
			Status: application.CoverageStatusPartial,
			UncoveredSegments: []application.UncoveredSegment{
				{Missing: application.MissingBaseMap},
			},
			ElevationSourcesUsed: []domain.GeoDataSource{{Name: "europa-relevo"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockCheckCoverageService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.NotContains(t, stdout, "europa-mapa")
		assert.Contains(t, stdout, "Base map sources used: (none)")
	})

	t.Run("should report no coverage when nothing is registered", func(t *testing.T) {
		// given
		output := application.CheckCoverageOutput{
			Status:            application.CoverageStatusNone,
			UncoveredSegments: []application.UncoveredSegment{{Missing: application.MissingBoth}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockCheckCoverageService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Coverage: none")
	})

	t.Run("should map an empty track file to the same exit code as inspect", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockCheckCoverageService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(application.CheckCoverageOutput{}, domain.ErrEmptyFile)

		// when
		_, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.Error(t, err)
		assert.Equal(t, 1, cli.ExitCode(err))
	})
}
