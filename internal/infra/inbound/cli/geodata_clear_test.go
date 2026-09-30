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

// executeGeoDataClearCommand runs the "geodata clear" command with
// geoDataService as its only dependency and returns stdout and the
// resulting error. The service is a test double — this is a unit test of
// the CLI adapter alone (Constitution Principle III), never a real
// GeoDataService.
func executeGeoDataClearCommand(t *testing.T, geoDataService application.GeoDataService, args ...string) (stdout string, err error) {
	t.Helper()

	cmd := cli.NewGeoDataClearCommand(geoDataService)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), err
}

func Test_GeoDataClearCommand_Args(t *testing.T) {
	t.Run("should return a usage error when an extra positional argument is given", func(t *testing.T) {
		// given/when
		_, err := executeGeoDataClearCommand(t, nil, "--confirm", "extra")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}

func Test_GeoDataClearCommand_Execute(t *testing.T) {
	t.Run("should refuse without --confirm, propagating the not-confirmed error", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().Clear(false).Return(3, domain.ErrRegistryClearNotConfirmed)

		// when
		_, err := executeGeoDataClearCommand(t, mockedService)

		// then
		require.Error(t, err)
		assert.Equal(t, 57, cli.ExitCode(err))
	})

	t.Run("should report how many entries were removed with --confirm", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().Clear(true).Return(3, nil)

		// when
		stdout, err := executeGeoDataClearCommand(t, mockedService, "--confirm")

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, cli.ExitCode(err))
		assert.Contains(t, stdout, "3 entries removed")
	})

	t.Run("should report zero entries removed on an already empty registry", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		mockedService := mockapplication.NewMockGeoDataService(mockCtrl)
		mockedService.EXPECT().Clear(true).Return(0, nil)

		// when
		stdout, err := executeGeoDataClearCommand(t, mockedService, "--confirm")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "0 entries removed")
	})
}
