package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/application/mock_application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

// executeGeoDataRegisterCommand runs the "geodata register" command with
// registerGeoDataService as its only dependency and returns stdout and the
// resulting error. The service is a test double — this is a unit test of
// the CLI adapter alone (Constitution Principle III), never a real
// GeoDataInspector/GeoDataRegistry.
func executeGeoDataRegisterCommand(t *testing.T, registerGeoDataService application.RegisterGeoDataService, args ...string) (stdout string, err error) {
	t.Helper()

	cmd := cli.NewGeoDataRegisterCommand(registerGeoDataService)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), err
}

func Test_GeoDataRegisterCommand_Args(t *testing.T) {
	t.Run("should return a usage error when no file argument is given", func(t *testing.T) {
		// given/when
		_, err := executeGeoDataRegisterCommand(t, nil, "--name", "x")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error when --name is missing", func(t *testing.T) {
		// given/when
		_, err := executeGeoDataRegisterCommand(t, nil, "/data/mapa.mbtiles")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.Contains(t, err.Error(), "--name is required")
	})
}

func Test_GeoDataRegisterCommand_Execute(t *testing.T) {
	t.Run("should print the registered source's name, type and area when the service succeeds", func(t *testing.T) {
		// given
		output := application.RegisterGeoDataOutput{
			Source: domain.GeoDataSource{
				Name:        "europa-central-mapa",
				Type:        domain.DataTypeBaseMap,
				BoundingBox: domain.BoundingBox{MinLatitude: 40, MaxLatitude: 50, MinLongitude: 10, MaxLongitude: 20},
			},
		}

		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockRegisterGeoDataService(mockCtrl)
		mockedService.EXPECT().Execute(application.RegisterGeoDataInput{Name: "europa-central-mapa", Path: "/data/mapa.mbtiles"}).Return(output, nil)

		// when
		stdout, err := executeGeoDataRegisterCommand(t, mockedService, "/data/mapa.mbtiles", "--name", "europa-central-mapa")

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, cli.ExitCode(err))
		assert.Contains(t, stdout, "europa-central-mapa")
		assert.Contains(t, stdout, "base map")
	})

	t.Run("should map a nonexistent file to exit code 5", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockRegisterGeoDataService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(application.RegisterGeoDataOutput{}, domain.ErrDataFileNotFound)

		// when
		_, err := executeGeoDataRegisterCommand(t, mockedService, "/data/does-not-exist.mbtiles", "--name", "x")

		// then
		require.Error(t, err)
		assert.Equal(t, 5, cli.ExitCode(err))
	})

	t.Run("should map an unreadable file to exit code 6", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockRegisterGeoDataService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(application.RegisterGeoDataOutput{}, domain.ErrDataFileUnreadable)

		// when
		_, err := executeGeoDataRegisterCommand(t, mockedService, "/data/no-permission.mbtiles", "--name", "x")

		// then
		require.Error(t, err)
		assert.Equal(t, 6, cli.ExitCode(err))
	})

	t.Run("should map an unsupported format to exit code 7", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockRegisterGeoDataService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(application.RegisterGeoDataOutput{}, domain.ErrUnsupportedDataFormat)

		// when
		_, err := executeGeoDataRegisterCommand(t, mockedService, "/data/invalid.dat", "--name", "x")

		// then
		require.Error(t, err)
		assert.Equal(t, 7, cli.ExitCode(err))
	})

	t.Run("should map a name already in use to exit code 8", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mock_application.NewMockRegisterGeoDataService(mockCtrl)
		mockedService.EXPECT().Execute(gomock.Any()).Return(application.RegisterGeoDataOutput{}, domain.ErrDataSourceNameAlreadyUsed)

		// when
		_, err := executeGeoDataRegisterCommand(t, mockedService, "/data/mapa.mbtiles", "--name", "europa-central-mapa")

		// then
		require.Error(t, err)
		assert.Equal(t, 8, cli.ExitCode(err))
	})
}
