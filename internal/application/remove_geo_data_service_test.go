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

func Test_removeGeoDataService_Execute(t *testing.T) {
	t.Run("should reject a name that does not match any registered source", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName("nao-existe").Return(domain.GeoDataSource{}, false, nil)

		service := application.NewRemoveGeoDataService(registry)

		// when
		err := service.Execute(application.RemoveGeoDataInput{Name: "nao-existe"})

		// then
		assert.ErrorIs(t, err, domain.ErrDataSourceNotRegistered)
	})

	t.Run("should propagate the registry's FindByName error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName(gomock.Any()).Return(domain.GeoDataSource{}, false, wantErr)

		service := application.NewRemoveGeoDataService(registry)

		// when
		err := service.Execute(application.RemoveGeoDataInput{Name: "x"})

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should delete the registered source with the given name", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName("europa-mapa").Return(domain.GeoDataSource{Name: "europa-mapa"}, true, nil)
		registry.EXPECT().Delete("europa-mapa").Return(nil)

		service := application.NewRemoveGeoDataService(registry)

		// when
		err := service.Execute(application.RemoveGeoDataInput{Name: "europa-mapa"})

		// then
		require.NoError(t, err)
	})

	t.Run("should propagate the registry's Delete error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName("europa-mapa").Return(domain.GeoDataSource{Name: "europa-mapa"}, true, nil)
		registry.EXPECT().Delete("europa-mapa").Return(wantErr)

		service := application.NewRemoveGeoDataService(registry)

		// when
		err := service.Execute(application.RemoveGeoDataInput{Name: "europa-mapa"})

		// then
		assert.ErrorIs(t, err, wantErr)
	})
}
