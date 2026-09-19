package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

// executeGeoDataListCommand runs the "geodata list" command with
// geoDataService as its only dependency and returns stdout and the
// resulting error. The service is a test double — this is a unit test of
// the CLI adapter alone (Constitution Principle III), never a real
// GeoDataService.
func executeGeoDataListCommand(t *testing.T, geoDataService application.GeoDataService) (stdout string, err error) {
	t.Helper()

	cmd := cli.NewGeoDataListCommand(geoDataService)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err = cmd.Execute()

	return out.String(), err
}

func Test_GeoDataListCommand_Execute(t *testing.T) {
	t.Run("should print name, type and area for two or more registered sources", func(t *testing.T) {
		// given
		summaries := []domain.GeoDataSummary{
			{Source: domain.GeoDataSource{Name: "europa-mapa", Type: domain.DataTypeBaseMap, BoundingBox: domain.BoundingBox{MinLatitude: 40, MaxLatitude: 50, MinLongitude: 10, MaxLongitude: 20}}, Available: true},
			{Source: domain.GeoDataSource{Name: "europa-relevo", Type: domain.DataTypeElevation, BoundingBox: domain.BoundingBox{MinLatitude: 50, MaxLatitude: 51, MinLongitude: 10, MaxLongitude: 11}}, Available: true},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().List().Return(summaries, nil)

		// when
		stdout, err := executeGeoDataListCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "europa-mapa")
		assert.Contains(t, stdout, "base map")
		assert.Contains(t, stdout, "europa-relevo")
		assert.Contains(t, stdout, "elevation")
	})

	t.Run("should flag a source whose file was moved or deleted without omitting the others", func(t *testing.T) {
		// given
		summaries := []domain.GeoDataSummary{
			{Source: domain.GeoDataSource{Name: "europa-mapa"}, Available: true},
			{Source: domain.GeoDataSource{Name: "europa-relevo"}, Available: false},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().List().Return(summaries, nil)

		// when
		stdout, err := executeGeoDataListCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "europa-mapa")
		assert.Contains(t, stdout, "europa-relevo")
		assert.Contains(t, stdout, "file not found")
	})

	t.Run("should print a clear message when no source is registered", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().List().Return(nil, nil)

		// when
		stdout, err := executeGeoDataListCommand(t, mockedService)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, cli.ExitCode(err))
		assert.Contains(t, stdout, "No geo data registered")
	})
}
