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
	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

// executeGeoDataCheckCommand runs the "geodata check" command against an
// existing temporary file, with geoDataService as its only dependency, and
// returns stdout and the resulting error. The service is a test double —
// this is a unit test of the CLI adapter alone (Constitution Principle
// III), never a real GeoDataService.
func executeGeoDataCheckCommand(t *testing.T, geoDataService application.GeoDataService, extraArgs ...string) (stdout string, err error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "track.gpx")
	require.NoError(t, os.WriteFile(path, []byte("irrelevant, the service is mocked"), 0o600))

	cmd := cli.NewGeoDataCheckCommand(geoDataService)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(append([]string{path}, extraArgs...))

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
		output := domain.CoverageReport{
			Status:               domain.CoverageStatusFull,
			BaseMapSourcesUsed:   []domain.GeoDataSource{{Name: "europa-mapa"}},
			ElevationSourcesUsed: []domain.GeoDataSource{{Name: "europa-relevo"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().CheckCoverage(gomock.Any(), gomock.Any()).Return(output, nil)

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
		output := domain.CoverageReport{
			Status: domain.CoverageStatusPartial,
			UncoveredSegments: []domain.UncoveredSegment{
				{StartLatitude: 45, StartLongitude: 15, EndLatitude: 46, EndLongitude: 16, Missing: domain.MissingElevation},
			},
			BaseMapSourcesUsed: []domain.GeoDataSource{{Name: "europa-mapa"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().CheckCoverage(gomock.Any(), gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Coverage: partial")
		assert.Contains(t, stdout, "missing elevation")
	})

	t.Run("should report a missing-base-map verdict when only an elevation source covers the track", func(t *testing.T) {
		// given
		output := domain.CoverageReport{
			Status: domain.CoverageStatusPartial,
			UncoveredSegments: []domain.UncoveredSegment{
				{StartLatitude: 45, StartLongitude: 15, EndLatitude: 46, EndLongitude: 16, Missing: domain.MissingBaseMap},
			},
			ElevationSourcesUsed: []domain.GeoDataSource{{Name: "europa-relevo"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().CheckCoverage(gomock.Any(), gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Coverage: partial")
		assert.Contains(t, stdout, "missing base map")
	})

	t.Run("should print the uncovered segment's start and end coordinates, each as (lat, lon)", func(t *testing.T) {
		// given
		output := domain.CoverageReport{
			Status: domain.CoverageStatusPartial,
			UncoveredSegments: []domain.UncoveredSegment{
				{StartLatitude: 60, StartLongitude: 30, EndLatitude: 61, EndLongitude: 31, Missing: domain.MissingBoth},
			},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().CheckCoverage(gomock.Any(), gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Uncovered segments:\n  - from (60.000000, 30.000000) to (61.000000, 31.000000): missing base map and elevation\n")
	})

	t.Run("should print the specific source chosen when two base map sources overlap", func(t *testing.T) {
		// given: the resolution itself (smaller area wins) is verified at
		// the application layer (geo_data_service_test.go) — this only
		// checks the CLI surfaces whichever source the service picked.
		output := domain.CoverageReport{
			Status:             domain.CoverageStatusFull,
			BaseMapSourcesUsed: []domain.GeoDataSource{{Name: "regiao-especifica"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().CheckCoverage(gomock.Any(), gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "regiao-especifica")
	})

	t.Run("should report full coverage for a track crossing the antimeridian without special-casing it", func(t *testing.T) {
		// given: antimeridian handling itself is verified at the
		// application layer (geo_data_service_test.go) and in
		// BoundingBox.Contains/AreaDegrees (bounding_box_test.go).
		output := domain.CoverageReport{
			Status:               domain.CoverageStatusFull,
			BaseMapSourcesUsed:   []domain.GeoDataSource{{Name: "antimeridiano-mapa"}},
			ElevationSourcesUsed: []domain.GeoDataSource{{Name: "antimeridiano-relevo"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().CheckCoverage(gomock.Any(), gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Coverage: full")
	})

	t.Run("should not list a registered source whose file no longer exists among the sources used", func(t *testing.T) {
		// given: excluding the source itself is verified at the
		// application layer (geo_data_service_test.go) — this only checks
		// the CLI never prints a source the service did not report.
		output := domain.CoverageReport{
			Status: domain.CoverageStatusPartial,
			UncoveredSegments: []domain.UncoveredSegment{
				{Missing: domain.MissingBaseMap},
			},
			ElevationSourcesUsed: []domain.GeoDataSource{{Name: "europa-relevo"}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().CheckCoverage(gomock.Any(), gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.NotContains(t, stdout, "europa-mapa")
		assert.Contains(t, stdout, "Base map sources used: (none)")
	})

	t.Run("should report no coverage when nothing is registered", func(t *testing.T) {
		// given
		output := domain.CoverageReport{
			Status:            domain.CoverageStatusNone,
			UncoveredSegments: []domain.UncoveredSegment{{Missing: domain.MissingBoth}},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().CheckCoverage(gomock.Any(), gomock.Any()).Return(output, nil)

		// when
		stdout, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Coverage: none")
	})

	t.Run("should map an empty track file to the same exit code as inspect", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().CheckCoverage(gomock.Any(), gomock.Any()).Return(domain.CoverageReport{}, domain.ErrEmptyFile)

		// when
		_, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.Error(t, err)
		assert.Equal(t, 1, cli.ExitCode(err))
	})

	t.Run("should pass the requested base map and elevation names as a SourceSelection", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		baseMap, elevation := "mapa-b", "relevo-a"
		mockedService.EXPECT().
			CheckCoverage(gomock.Any(), domain.SourceSelection{BaseMapName: &baseMap, ElevationName: &elevation}).
			Return(domain.CoverageReport{Status: domain.CoverageStatusFull}, nil)

		// when
		_, err := executeGeoDataCheckCommand(t, mockedService, "--base-map", "mapa-b", "--elevation", "relevo-a")

		// then
		require.NoError(t, err)
	})

	t.Run("should pass an empty SourceSelection when neither flag is given", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().
			CheckCoverage(gomock.Any(), domain.SourceSelection{}).
			Return(domain.CoverageReport{Status: domain.CoverageStatusFull}, nil)

		// when
		_, err := executeGeoDataCheckCommand(t, mockedService)

		// then
		require.NoError(t, err)
	})

	t.Run("should map a requested source of the wrong type to exit code 56", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().CheckCoverage(gomock.Any(), gomock.Any()).Return(domain.CoverageReport{}, domain.ErrDataSourceTypeMismatch)

		// when
		_, err := executeGeoDataCheckCommand(t, mockedService, "--base-map", "relevo-a")

		// then
		require.Error(t, err)
		assert.Equal(t, 56, cli.ExitCode(err))
	})

	t.Run("should map a requested source that does not exist to exit code 9", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().CheckCoverage(gomock.Any(), gomock.Any()).Return(domain.CoverageReport{}, domain.ErrDataSourceNotRegistered)

		// when
		_, err := executeGeoDataCheckCommand(t, mockedService, "--base-map", "nao-existe")

		// then
		require.Error(t, err)
		assert.Equal(t, 9, cli.ExitCode(err))
	})
}
