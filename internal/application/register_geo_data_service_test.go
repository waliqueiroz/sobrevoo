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

		service := application.NewRegisterGeoDataService(registry, nil, nil)

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

		service := application.NewRegisterGeoDataService(registry, nil, nil)

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

		service := application.NewRegisterGeoDataService(registry, inspector, nil)

		// when
		_, err := service.Execute(application.RegisterGeoDataInput{Name: "europa-central-mapa", Path: "/data/mapa.mbtiles"})

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should register a source with the type, format and area discovered by the inspector, stamped with the current time", func(t *testing.T) {
		// given
		registeredAt := time.Date(2026, time.March, 4, 10, 30, 0, 0, time.UTC)
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

		clock := mock_domain.NewMockClock(mockCtrl)
		clock.EXPECT().Now().Return(registeredAt)

		wantSource := domain.GeoDataSource{
			Name:         "europa-central-mapa",
			Path:         "/data/mapa.mbtiles",
			Type:         domain.DataTypeBaseMap,
			Format:       domain.DataFormatMBTiles,
			BoundingBox:  inspected.BoundingBox,
			RegisteredAt: registeredAt,
		}
		registry.EXPECT().Save(wantSource).Return(nil)

		service := application.NewRegisterGeoDataService(registry, inspector, clock)

		// when
		output, err := service.Execute(application.RegisterGeoDataInput{Name: "europa-central-mapa", Path: "/data/mapa.mbtiles"})

		// then
		require.NoError(t, err)
		assert.Equal(t, wantSource, output.Source)
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

		clock := mock_domain.NewMockClock(mockCtrl)
		clock.EXPECT().Now().Return(time.Now())

		service := application.NewRegisterGeoDataService(registry, inspector, clock)

		// when
		_, err := service.Execute(application.RegisterGeoDataInput{Name: "europa-central-mapa", Path: "/data/mapa.mbtiles"})

		// then
		assert.ErrorIs(t, err, wantErr)
	})
}
