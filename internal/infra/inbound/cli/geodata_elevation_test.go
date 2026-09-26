package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

func executeGeoDataElevationCommand(t *testing.T, service *mockapplication.MockGeoDataService, args ...string) (stdout string, err error) {
	t.Helper()

	cmd := cli.NewGeoDataElevationCommand(service)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), err
}

func newGeoDataServiceMock(t *testing.T) *mockapplication.MockGeoDataService {
	t.Helper()
	return mockapplication.NewMockGeoDataService(gomock.NewController(t))
}

func Test_GeoDataElevationCommand_Args(t *testing.T) {
	t.Run("should return a usage error when --lat is missing", func(t *testing.T) {
		// when
		_, err := executeGeoDataElevationCommand(t, newGeoDataServiceMock(t), "--lon", "10")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.ErrorContains(t, err, "--lat")
	})

	t.Run("should return a usage error when --lon is missing", func(t *testing.T) {
		// when
		_, err := executeGeoDataElevationCommand(t, newGeoDataServiceMock(t), "--lat", "10")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.ErrorContains(t, err, "--lon")
	})

	t.Run("should return a usage error for a latitude that is not a number", func(t *testing.T) {
		// when
		_, err := executeGeoDataElevationCommand(t, newGeoDataServiceMock(t), "--lat", "abc", "--lon", "10")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.ErrorContains(t, err, "--lat")
	})

	t.Run("should return a usage error for a longitude that is infinite", func(t *testing.T) {
		// when
		_, err := executeGeoDataElevationCommand(t, newGeoDataServiceMock(t), "--lat", "10", "--lon", "inf")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error for a stray argument", func(t *testing.T) {
		// when
		_, err := executeGeoDataElevationCommand(t, newGeoDataServiceMock(t), "--lat", "10", "--lon", "10", "extra")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}

func Test_GeoDataElevationCommand_Execute(t *testing.T) {
	t.Run("should ask for the elevation of the coordinate, negative values included", func(t *testing.T) {
		// given
		service := newGeoDataServiceMock(t)
		service.EXPECT().ElevationAt(-23.5505, -46.6333).Return(builddomain.NewElevationReadingBuilder().Build(), nil)

		// when
		_, err := executeGeoDataElevationCommand(t, service, "--lat", "-23.5505", "--lon", "-46.6333")

		// then
		assert.NoError(t, err)
	})

	t.Run("should print the elevation in meters, the source and the cell", func(t *testing.T) {
		// given
		service := newGeoDataServiceMock(t)
		reading := builddomain.NewElevationReadingBuilder().
			WithMeters(760).WithCell(412, 88).
			WithSource(builddomain.NewGeoDataSourceBuilder().WithName("srtm-sp").Build()).Build()
		service.EXPECT().ElevationAt(gomock.Any(), gomock.Any()).Return(reading, nil)

		// when
		stdout, err := executeGeoDataElevationCommand(t, service, "--lat", "-23.5", "--lon", "-46.6")

		// then
		require.NoError(t, err)
		assert.Equal(t, "Elevation: 760.0 m\nSource: srtm-sp (cell row 412, column 88)\n", stdout)
	})

	t.Run("should say the file has no value, not print a number", func(t *testing.T) {
		// given
		service := newGeoDataServiceMock(t)
		reading := builddomain.NewElevationReadingBuilder().WithoutValue().WithCell(3, 4).
			WithSource(builddomain.NewGeoDataSourceBuilder().WithName("srtm-sp").Build()).Build()
		service.EXPECT().ElevationAt(gomock.Any(), gomock.Any()).Return(reading, nil)

		// when
		stdout, err := executeGeoDataElevationCommand(t, service, "--lat", "-23.5", "--lon", "-46.6")

		// then
		require.NoError(t, err)
		assert.Equal(t, "Elevation: no value (the file has no data for this point)\nSource: srtm-sp (cell row 3, column 4)\n", stdout)
		assert.Equal(t, 0, cli.ExitCode(err))
	})

	t.Run("should print a negative elevation with its sign", func(t *testing.T) {
		// given
		service := newGeoDataServiceMock(t)
		service.EXPECT().ElevationAt(gomock.Any(), gomock.Any()).Return(builddomain.NewElevationReadingBuilder().WithMeters(-12.34).Build(), nil)

		// when
		stdout, err := executeGeoDataElevationCommand(t, service, "--lat", "31.5", "--lon", "35.4")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Elevation: -12.3 m\n")
	})

	t.Run("should return the not-covered error with code 25 and print nothing", func(t *testing.T) {
		// given
		service := newGeoDataServiceMock(t)
		service.EXPECT().ElevationAt(gomock.Any(), gomock.Any()).Return(domain.ElevationReading{}, domain.ErrElevationNotCovered)

		// when
		stdout, err := executeGeoDataElevationCommand(t, service, "--lat", "0", "--lon", "0")

		// then
		assert.Equal(t, 25, cli.ExitCode(err))
		assert.Empty(t, stdout)
	})

	t.Run("should return the invalid-coordinate error with code 26", func(t *testing.T) {
		// given
		service := newGeoDataServiceMock(t)
		service.EXPECT().ElevationAt(gomock.Any(), gomock.Any()).Return(domain.ElevationReading{}, domain.ErrInvalidCoordinate)

		// when
		_, err := executeGeoDataElevationCommand(t, service, "--lat", "91", "--lon", "0")

		// then
		assert.Equal(t, 26, cli.ExitCode(err))
	})

	t.Run("should return an unreadable-content error with code 21", func(t *testing.T) {
		// given
		service := newGeoDataServiceMock(t)
		service.EXPECT().ElevationAt(gomock.Any(), gomock.Any()).Return(domain.ElevationReading{}, domain.ErrGeoDataContentUnreadable)

		// when
		_, err := executeGeoDataElevationCommand(t, service, "--lat", "1", "--lon", "1")

		// then
		assert.Equal(t, 21, cli.ExitCode(err))
	})

	t.Run("should return an unsupported-unit error with code 22", func(t *testing.T) {
		// given
		service := newGeoDataServiceMock(t)
		service.EXPECT().ElevationAt(gomock.Any(), gomock.Any()).Return(domain.ElevationReading{}, domain.ErrElevationUnitUnsupported)

		// when
		_, err := executeGeoDataElevationCommand(t, service, "--lat", "1", "--lon", "1")

		// then
		assert.Equal(t, 22, cli.ExitCode(err))
	})
}
