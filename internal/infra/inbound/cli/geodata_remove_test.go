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

// executeGeoDataRemoveCommand runs the "geodata remove" command with
// geoDataService as its only dependency and returns stdout and the
// resulting error. The service is a test double — this is a unit test of
// the CLI adapter alone (Constitution Principle III), never a real
// GeoDataService.
func executeGeoDataRemoveCommand(t *testing.T, geoDataService application.GeoDataService, args ...string) (stdout string, err error) {
	t.Helper()

	cmd := cli.NewGeoDataRemoveCommand(geoDataService)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), err
}

func Test_GeoDataRemoveCommand_Args(t *testing.T) {
	t.Run("should return a usage error when no name argument is given", func(t *testing.T) {
		// given/when
		_, err := executeGeoDataRemoveCommand(t, nil)

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}

func Test_GeoDataRemoveCommand_Execute(t *testing.T) {
	t.Run("should confirm removal when the service succeeds", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().Remove("europa-mapa").Return(nil)

		// when
		stdout, err := executeGeoDataRemoveCommand(t, mockedService, "europa-mapa")

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, cli.ExitCode(err))
		assert.Contains(t, stdout, "europa-mapa")
	})

	t.Run("should map a name not registered to exit code 9", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().Remove(gomock.Any()).Return(domain.ErrDataSourceNotRegistered)

		// when
		_, err := executeGeoDataRemoveCommand(t, mockedService, "nao-existe")

		// then
		require.Error(t, err)
		assert.Equal(t, 9, cli.ExitCode(err))
	})
}
