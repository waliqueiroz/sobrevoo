package application_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/build_domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mock_domain"
)

// alwaysAvailable configures a mocked FileChecker to report every path as
// present.
func alwaysAvailable(ctrl *gomock.Controller) domain.FileChecker {
	checker := mock_domain.NewMockFileChecker(ctrl)
	checker.EXPECT().Exists(gomock.Any()).Return(true).AnyTimes()
	return checker
}

func newGeoDataServiceForCoverage(t *testing.T, ctrl *gomock.Controller, track domain.Track, sources []domain.GeoDataSource, fileChecker domain.FileChecker) application.GeoDataService {
	t.Helper()

	parser := mock_domain.NewMockTrackParser(ctrl)
	parser.EXPECT().Parse(gomock.Any()).Return(track, nil)

	registry := mock_domain.NewMockGeoDataRegistry(ctrl)
	registry.EXPECT().List().Return(sources, nil)

	return application.NewGeoDataService(registry, nil, fileChecker, parser, testMinPoints, testMaxPlausibleSpeedKmh)
}

func Test_geoDataService_Register(t *testing.T) {
	t.Run("should reject a name already used by another registered source", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName("europa-central-mapa").Return(domain.GeoDataSource{Name: "europa-central-mapa"}, true, nil)

		service := application.NewGeoDataService(registry, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Register("europa-central-mapa", "/data/mapa.mbtiles")

		// then
		assert.ErrorIs(t, err, domain.ErrDataSourceNameAlreadyUsed)
	})

	t.Run("should propagate the registry's FindByName error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName(gomock.Any()).Return(domain.GeoDataSource{}, false, wantErr)

		service := application.NewGeoDataService(registry, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Register("x", "/data/mapa.mbtiles")

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

		service := application.NewGeoDataService(registry, inspector, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Register("europa-central-mapa", "/data/mapa.mbtiles")

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

		service := application.NewGeoDataService(registry, inspector, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)
		before := time.Now()

		// when
		source, err := service.Register("europa-central-mapa", "/data/mapa.mbtiles")

		// then
		after := time.Now()
		require.NoError(t, err)
		assert.Equal(t, "europa-central-mapa", source.Name)
		assert.Equal(t, "/data/mapa.mbtiles", source.Path)
		assert.Equal(t, domain.DataTypeBaseMap, source.Type)
		assert.Equal(t, domain.DataFormatMBTiles, source.Format)
		assert.Equal(t, inspected.BoundingBox, source.BoundingBox)
		assert.False(t, source.RegisteredAt.Before(before))
		assert.False(t, source.RegisteredAt.After(after))
		assert.Equal(t, source, saved)
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

		service := application.NewGeoDataService(registry, inspector, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Register("europa-central-mapa", "/data/mapa.mbtiles")

		// then
		assert.ErrorIs(t, err, wantErr)
	})
}

func Test_geoDataService_List(t *testing.T) {
	t.Run("should propagate the registry's error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().List().Return(nil, wantErr)

		service := application.NewGeoDataService(registry, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.List()

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should return an empty list when no source is registered", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().List().Return(nil, nil)

		service := application.NewGeoDataService(registry, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		summaries, err := service.List()

		// then
		require.NoError(t, err)
		assert.Empty(t, summaries)
	})

	t.Run("should mark a source as unavailable when its file is no longer found", func(t *testing.T) {
		// given
		source := domain.GeoDataSource{Name: "europa-mapa", Path: "/data/mapa.mbtiles"}

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().List().Return([]domain.GeoDataSource{source}, nil)

		fileChecker := mock_domain.NewMockFileChecker(mockCtrl)
		fileChecker.EXPECT().Exists("/data/mapa.mbtiles").Return(false)

		service := application.NewGeoDataService(registry, nil, fileChecker, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		summaries, err := service.List()

		// then
		require.NoError(t, err)
		require.Len(t, summaries, 1)
		assert.Equal(t, source, summaries[0].Source)
		assert.False(t, summaries[0].Available)
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

		service := application.NewGeoDataService(registry, nil, fileChecker, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		summaries, err := service.List()

		// then
		require.NoError(t, err)
		require.Len(t, summaries, 2)
		assert.True(t, summaries[0].Available)
		assert.False(t, summaries[1].Available)
	})
}

func Test_geoDataService_Remove(t *testing.T) {
	t.Run("should reject a name that does not match any registered source", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName("nao-existe").Return(domain.GeoDataSource{}, false, nil)

		service := application.NewGeoDataService(registry, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		err := service.Remove("nao-existe")

		// then
		assert.ErrorIs(t, err, domain.ErrDataSourceNotRegistered)
	})

	t.Run("should propagate the registry's FindByName error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName(gomock.Any()).Return(domain.GeoDataSource{}, false, wantErr)

		service := application.NewGeoDataService(registry, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		err := service.Remove("x")

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should delete the registered source with the given name", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().FindByName("europa-mapa").Return(domain.GeoDataSource{Name: "europa-mapa"}, true, nil)
		registry.EXPECT().Delete("europa-mapa").Return(nil)

		service := application.NewGeoDataService(registry, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		err := service.Remove("europa-mapa")

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

		service := application.NewGeoDataService(registry, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		err := service.Remove("europa-mapa")

		// then
		assert.ErrorIs(t, err, wantErr)
	})
}

func Test_geoDataService_CheckCoverage(t *testing.T) {
	t.Run("should propagate the parser's error unchanged", func(t *testing.T) {
		// given
		wantErr := domain.ErrEmptyFile

		mockCtrl := gomock.NewController(t)
		parser := mock_domain.NewMockTrackParser(mockCtrl)
		parser.EXPECT().Parse(gomock.Any()).Return(domain.Track{}, wantErr)

		service := application.NewGeoDataService(nil, nil, nil, parser, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.CheckCoverage(strings.NewReader(""))

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should propagate the registry's List error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		parser := mock_domain.NewMockTrackParser(mockCtrl)
		parser.EXPECT().Parse(gomock.Any()).Return(track, nil)
		registry := mock_domain.NewMockGeoDataRegistry(mockCtrl)
		registry.EXPECT().List().Return(nil, wantErr)

		service := application.NewGeoDataService(registry, nil, nil, parser, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.CheckCoverage(strings.NewReader(""))

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should list every base map source that covered at least one point, sorted by name", func(t *testing.T) {
		// given: two non-overlapping base map sources, each covering one of the two points
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(1).WithLongitude(1).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
		).Build()

		sourceB := build_domain.NewGeoDataSourceBuilder().WithName("b-mapa").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}).Build()
		sourceA := build_domain.NewGeoDataSourceBuilder().WithName("a-mapa").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 2, MinLongitude: 0, MaxLongitude: 2}).Build()

		mockCtrl := gomock.NewController(t)
		service := newGeoDataServiceForCoverage(t, mockCtrl, track, []domain.GeoDataSource{sourceB, sourceA}, alwaysAvailable(mockCtrl))

		// when
		output, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		require.Len(t, output.BaseMapSourcesUsed, 2)
		assert.Equal(t, "a-mapa", output.BaseMapSourcesUsed[0].Name)
		assert.Equal(t, "b-mapa", output.BaseMapSourcesUsed[1].Name)
	})

	t.Run("should report full coverage when a base map and an elevation source cover every point", func(t *testing.T) {
		// given
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		).Build()

		baseMap := build_domain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).Build()
		elevation := build_domain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).Build()

		mockCtrl := gomock.NewController(t)
		service := newGeoDataServiceForCoverage(t, mockCtrl, track, []domain.GeoDataSource{baseMap, elevation}, alwaysAvailable(mockCtrl))

		// when
		output, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		assert.Equal(t, application.CoverageStatusFull, output.Status)
		assert.Empty(t, output.UncoveredSegments)
		require.Len(t, output.BaseMapSourcesUsed, 1)
		assert.Equal(t, "europa-mapa", output.BaseMapSourcesUsed[0].Name)
		require.Len(t, output.ElevationSourcesUsed, 1)
		assert.Equal(t, "europa-relevo", output.ElevationSourcesUsed[0].Name)
	})

	t.Run("should report partial coverage when only a base map covers the whole track and no elevation is registered", func(t *testing.T) {
		// given
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		).Build()

		baseMap := build_domain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).Build()

		mockCtrl := gomock.NewController(t)
		service := newGeoDataServiceForCoverage(t, mockCtrl, track, []domain.GeoDataSource{baseMap}, alwaysAvailable(mockCtrl))

		// when
		output, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		assert.Equal(t, application.CoverageStatusPartial, output.Status)
		require.Len(t, output.UncoveredSegments, 1)
		assert.Equal(t, application.MissingElevation, output.UncoveredSegments[0].Missing)
		assert.NotEmpty(t, output.BaseMapSourcesUsed)
		assert.Empty(t, output.ElevationSourcesUsed)
	})

	t.Run("should report partial coverage when only an elevation source covers the whole track and no base map is registered", func(t *testing.T) {
		// given
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		).Build()

		elevation := build_domain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).Build()

		mockCtrl := gomock.NewController(t)
		service := newGeoDataServiceForCoverage(t, mockCtrl, track, []domain.GeoDataSource{elevation}, alwaysAvailable(mockCtrl))

		// when
		output, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		assert.Equal(t, application.CoverageStatusPartial, output.Status)
		require.Len(t, output.UncoveredSegments, 1)
		assert.Equal(t, application.MissingBaseMap, output.UncoveredSegments[0].Missing)
	})

	t.Run("should report a single uncovered segment with the coordinates of the point outside every registered source", func(t *testing.T) {
		// given
		covered := domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(60).WithLongitude(30).Build(), // outside "covered"
		).Build()

		baseMap := build_domain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).WithBoundingBox(covered).Build()
		elevation := build_domain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).WithBoundingBox(covered).Build()

		mockCtrl := gomock.NewController(t)
		service := newGeoDataServiceForCoverage(t, mockCtrl, track, []domain.GeoDataSource{baseMap, elevation}, alwaysAvailable(mockCtrl))

		// when
		output, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		assert.Equal(t, application.CoverageStatusPartial, output.Status)
		require.Len(t, output.UncoveredSegments, 1)
		assert.Equal(t, application.MissingBoth, output.UncoveredSegments[0].Missing)
		assert.Equal(t, 60.0, output.UncoveredSegments[0].StartLatitude)
		assert.Equal(t, 30.0, output.UncoveredSegments[0].StartLongitude)
		assert.Equal(t, 60.0, output.UncoveredSegments[0].EndLatitude)
		assert.Equal(t, 30.0, output.UncoveredSegments[0].EndLongitude)
	})

	t.Run("should report no coverage at all when no source is registered", func(t *testing.T) {
		// given
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		service := newGeoDataServiceForCoverage(t, mockCtrl, track, nil, alwaysAvailable(mockCtrl))

		// when
		output, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		assert.Equal(t, application.CoverageStatusNone, output.Status)
		require.Len(t, output.UncoveredSegments, 1)
		assert.Empty(t, output.BaseMapSourcesUsed)
		assert.Empty(t, output.ElevationSourcesUsed)
	})

	t.Run("should pick the more specific of two overlapping base map sources", func(t *testing.T) {
		// given
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(45.1).WithLongitude(15.1).Build(),
		).Build()

		wide := build_domain.NewGeoDataSourceBuilder().WithName("regiao-ampla").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 90, MinLongitude: 0, MaxLongitude: 90}).Build()
		narrow := build_domain.NewGeoDataSourceBuilder().WithName("regiao-especifica").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}).Build()
		elevation := build_domain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).
			WithBoundingBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 90, MinLongitude: 0, MaxLongitude: 90}).Build()

		mockCtrl := gomock.NewController(t)
		service := newGeoDataServiceForCoverage(t, mockCtrl, track, []domain.GeoDataSource{wide, narrow, elevation}, alwaysAvailable(mockCtrl))

		// when
		output, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		require.Len(t, output.BaseMapSourcesUsed, 1)
		assert.Equal(t, "regiao-especifica", output.BaseMapSourcesUsed[0].Name)
	})

	t.Run("should break a tie between equally specific sources by the oldest RegisteredAt", func(t *testing.T) {
		// given
		box := domain.BoundingBox{MinLatitude: 44, MaxLatitude: 47, MinLongitude: 14, MaxLongitude: 17}
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(45.1).WithLongitude(15.1).Build(),
		).Build()

		older := build_domain.NewGeoDataSourceBuilder().WithName("mais-antigo").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(box).WithRegisteredAt(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).Build()
		newer := build_domain.NewGeoDataSourceBuilder().WithName("mais-novo").WithType(domain.DataTypeBaseMap).
			WithBoundingBox(box).WithRegisteredAt(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)).Build()
		elevation := build_domain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).WithBoundingBox(box).Build()

		mockCtrl := gomock.NewController(t)
		service := newGeoDataServiceForCoverage(t, mockCtrl, track, []domain.GeoDataSource{newer, older, elevation}, alwaysAvailable(mockCtrl))

		// when
		output, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		require.Len(t, output.BaseMapSourcesUsed, 1)
		assert.Equal(t, "mais-antigo", output.BaseMapSourcesUsed[0].Name)
	})

	t.Run("should report coverage correctly for a track crossing the antimeridian", func(t *testing.T) {
		// given
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.9).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.9).Build(),
		).Build()

		box := domain.BoundingBox{MinLatitude: -1, MaxLatitude: 1, MinLongitude: 170, MaxLongitude: -170, CrossesAntimeridian: true}
		baseMap := build_domain.NewGeoDataSourceBuilder().WithName("antimeridiano-mapa").WithType(domain.DataTypeBaseMap).WithBoundingBox(box).Build()
		elevation := build_domain.NewGeoDataSourceBuilder().WithName("antimeridiano-relevo").WithType(domain.DataTypeElevation).WithBoundingBox(box).Build()

		mockCtrl := gomock.NewController(t)
		service := newGeoDataServiceForCoverage(t, mockCtrl, track, []domain.GeoDataSource{baseMap, elevation}, alwaysAvailable(mockCtrl))

		// when
		output, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		assert.Equal(t, application.CoverageStatusFull, output.Status)
	})

	t.Run("should exclude a registered source whose file no longer exists", func(t *testing.T) {
		// given
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		).Build()

		baseMap := build_domain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).WithPath("/data/mapa.mbtiles").Build()
		elevation := build_domain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).WithPath("/data/relevo.tif").Build()

		mockCtrl := gomock.NewController(t)
		fileChecker := mock_domain.NewMockFileChecker(mockCtrl)
		fileChecker.EXPECT().Exists("/data/mapa.mbtiles").Return(false)
		fileChecker.EXPECT().Exists("/data/relevo.tif").Return(true)

		service := newGeoDataServiceForCoverage(t, mockCtrl, track, []domain.GeoDataSource{baseMap, elevation}, fileChecker)

		// when
		output, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		assert.Equal(t, application.CoverageStatusPartial, output.Status)
		require.Len(t, output.UncoveredSegments, 1)
		assert.Equal(t, application.MissingBaseMap, output.UncoveredSegments[0].Missing)
		assert.Empty(t, output.BaseMapSourcesUsed)
	})
}
