package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mock_domain"
)

func Test_registerGeoDataService_Execute(t *testing.T) {
	t.Run("should reject a name already used by another registered source", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName("europa-central-mapa").Return(domain.GeoDataSource{Name: "europa-central-mapa"}, true, nil)

		service := application.NewRegisterGeoDataService(registry, nil)

		// when
		_, err := service.Execute(application.RegisterGeoDataInput{Name: "europa-central-mapa", Path: "/data/mapa.mbtiles"})

		// then
		assert.ErrorIs(t, err, domain.ErrDataSourceNameAlreadyUsed)
	})

	t.Run("should propagate the registry's FindByName error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName(gomock.Any()).Return(domain.GeoDataSource{}, false, wantErr)

		service := application.NewRegisterGeoDataService(registry, nil)

		// when
		_, err := service.Execute(application.RegisterGeoDataInput{Name: "x", Path: "/data/mapa.mbtiles"})

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should propagate the inspector's error unchanged when the name is not yet used", func(t *testing.T) {
		// given
		wantErr := domain.ErrUnsupportedDataFormat

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName("europa-central-mapa").Return(domain.GeoDataSource{}, false, nil)

		inspector := mock_domain.NewMockGeoDataInspector(mockCtrl)
		inspector.EXPECT().Inspect("/data/mapa.mbtiles").Return(domain.InspectedGeoData{}, wantErr)

		service := application.NewRegisterGeoDataService(registry, inspector)

		// when
		_, err := service.Execute(application.RegisterGeoDataInput{Name: "europa-central-mapa", Path: "/data/mapa.mbtiles"})

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should register a source with the type, format and area discovered by the inspector, stamped with the current time", func(t *testing.T) {
		// given
		inspected := domain.InspectedGeoData{
			Format:      domain.DataFormatMBTiles,
			Type:        domain.DataTypeBaseMap,
			BoundingBox: domain.BoundingBox{MinLatitude: 40, MaxLatitude: 50, MinLongitude: 10, MaxLongitude: 20},
		}

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName("europa-central-mapa").Return(domain.GeoDataSource{}, false, nil)

		inspector := mock_domain.NewMockGeoDataInspector(mockCtrl)
		inspector.EXPECT().Inspect("/data/mapa.mbtiles").Return(inspected, nil)

		var saved domain.GeoDataSource
		registry.EXPECT().Save(gomock.Any()).DoAndReturn(func(source domain.GeoDataSource) error {
			saved = source
			return nil
		})

		service := application.NewRegisterGeoDataService(registry, inspector)
		before := time.Now()

		// when
		output, err := service.Execute(application.RegisterGeoDataInput{Name: "europa-central-mapa", Path: "/data/mapa.mbtiles"})

		// then
		after := time.Now()
		require.NoError(t, err)
		assert.Equal(t, "europa-central-mapa", output.Source.Name)
		assert.Equal(t, "/data/mapa.mbtiles", output.Source.Path)
		assert.Equal(t, domain.DataTypeBaseMap, output.Source.Type)
		assert.Equal(t, domain.DataFormatMBTiles, output.Source.Format)
		assert.Equal(t, inspected.BoundingBox, output.Source.BoundingBox)
		assert.False(t, output.Source.RegisteredAt.Before(before))
		assert.False(t, output.Source.RegisteredAt.After(after))
		assert.Equal(t, output.Source, saved)
	})

	t.Run("should propagate the registry's Save error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName(gomock.Any()).Return(domain.GeoDataSource{}, false, nil)
		registry.EXPECT().Save(gomock.Any()).Return(wantErr)

		inspector := mock_domain.NewMockGeoDataInspector(mockCtrl)
		inspector.EXPECT().Inspect(gomock.Any()).Return(domain.InspectedGeoData{}, nil)

		service := application.NewRegisterGeoDataService(registry, inspector)

		// when
		_, err := service.Execute(application.RegisterGeoDataInput{Name: "europa-central-mapa", Path: "/data/mapa.mbtiles"})

		// then
		assert.ErrorIs(t, err, wantErr)
	})
}
