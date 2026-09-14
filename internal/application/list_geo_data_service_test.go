package application_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mock_domain"
)

func Test_listGeoDataService_Execute(t *testing.T) {
	t.Run("should propagate the registry's error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().List().Return(nil, wantErr)

		service := application.NewListGeoDataService(registry, nil)

		// when
		_, err := service.Execute()

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should return an empty list when no source is registered", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().List().Return(nil, nil)

		service := application.NewListGeoDataService(registry, nil)

		// when
		output, err := service.Execute()

		// then
		require.NoError(t, err)
		assert.Empty(t, output.Sources)
	})

	t.Run("should mark a source as unavailable when its file is no longer found", func(t *testing.T) {
		// given
		source := domain.GeoDataSource{Name: "europa-mapa", Path: "/data/mapa.mbtiles"}

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().List().Return([]domain.GeoDataSource{source}, nil)

		fileChecker := mock_domain.NewMockFileChecker(mockCtrl)
		fileChecker.EXPECT().Exists("/data/mapa.mbtiles").Return(false)

		service := application.NewListGeoDataService(registry, fileChecker)

		// when
		output, err := service.Execute()

		// then
		require.NoError(t, err)
		require.Len(t, output.Sources, 1)
		assert.Equal(t, source, output.Sources[0].Source)
		assert.False(t, output.Sources[0].Available)
	})

	t.Run("should mark a source as available when its file is still present, leaving other sources unaffected", func(t *testing.T) {
		// given
		present := domain.GeoDataSource{Name: "europa-mapa", Path: "/data/mapa.mbtiles"}
		missing := domain.GeoDataSource{Name: "europa-relevo", Path: "/data/relevo.tif"}

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().List().Return([]domain.GeoDataSource{present, missing}, nil)

		fileChecker := mock_domain.NewMockFileChecker(mockCtrl)
		fileChecker.EXPECT().Exists("/data/mapa.mbtiles").Return(true)
		fileChecker.EXPECT().Exists("/data/relevo.tif").Return(false)

		service := application.NewListGeoDataService(registry, fileChecker)

		// when
		output, err := service.Execute()

		// then
		require.NoError(t, err)
		require.Len(t, output.Sources, 2)
		assert.True(t, output.Sources[0].Available)
		assert.False(t, output.Sources[1].Available)
	})
}
