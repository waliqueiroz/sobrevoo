package application_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mockdomain"
)

func Test_geoDataService_Register(t *testing.T) {
	t.Run("should reject a name already used by another registered source", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().FindByName("europa-central-mapa").Return(domain.GeoDataSource{Name: "europa-central-mapa"}, true, nil)

		service := application.NewGeoDataService(repository, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Register("europa-central-mapa", "/data/mapa.mbtiles")

		// then
		assert.ErrorIs(t, err, domain.ErrDataSourceNameAlreadyUsed)
	})

	t.Run("should propagate the repository's FindByName error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().FindByName(gomock.Any()).Return(domain.GeoDataSource{}, false, wantErr)

		service := application.NewGeoDataService(repository, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Register("x", "/data/mapa.mbtiles")

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should propagate the inspector's error unchanged when the name is not yet used", func(t *testing.T) {
		// given
		wantErr := domain.ErrUnsupportedDataFormat

		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().FindByName("europa-central-mapa").Return(domain.GeoDataSource{}, false, nil)

		inspector := mockdomain.NewMockGeoDataInspector(mockCtrl)
		inspector.EXPECT().Inspect("/data/mapa.mbtiles").Return(domain.InspectedGeoData{}, wantErr)

		service := application.NewGeoDataService(repository, inspector, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Register("europa-central-mapa", "/data/mapa.mbtiles")

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should save and return the source built from what the inspector discovered", func(t *testing.T) {
		// given: NewGeoDataSource's own construction rules (fields, RegisteredAt)
		// are covered in internal/domain/geo_data_source_test.go — this only
		// checks the service wires the inspector's result into the repository.
		inspected := domain.InspectedGeoData{
			Format:      domain.DataFormatMBTiles,
			Type:        domain.DataTypeBaseMap,
			BoundingBox: domain.BoundingBox{MinLatitude: 40, MaxLatitude: 50, MinLongitude: 10, MaxLongitude: 20},
		}

		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().FindByName("europa-central-mapa").Return(domain.GeoDataSource{}, false, nil)

		inspector := mockdomain.NewMockGeoDataInspector(mockCtrl)
		inspector.EXPECT().Inspect("/data/mapa.mbtiles").Return(inspected, nil)

		var saved domain.GeoDataSource
		repository.EXPECT().Save(gomock.Any()).DoAndReturn(func(source domain.GeoDataSource) error {
			saved = source
			return nil
		})

		service := application.NewGeoDataService(repository, inspector, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		source, err := service.Register("europa-central-mapa", "/data/mapa.mbtiles")

		// then
		require.NoError(t, err)
		assert.Equal(t, "europa-central-mapa", source.Name)
		assert.Equal(t, "/data/mapa.mbtiles", source.Path)
		assert.Equal(t, inspected.Type, source.Type)
		assert.Equal(t, inspected.Format, source.Format)
		assert.Equal(t, inspected.BoundingBox, source.BoundingBox)
		assert.Equal(t, source, saved)
	})

	t.Run("should propagate the repository's Save error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().FindByName(gomock.Any()).Return(domain.GeoDataSource{}, false, nil)
		repository.EXPECT().Save(gomock.Any()).Return(wantErr)

		inspector := mockdomain.NewMockGeoDataInspector(mockCtrl)
		inspector.EXPECT().Inspect(gomock.Any()).Return(domain.InspectedGeoData{}, nil)

		service := application.NewGeoDataService(repository, inspector, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Register("europa-central-mapa", "/data/mapa.mbtiles")

		// then
		assert.ErrorIs(t, err, wantErr)
	})
}

func Test_geoDataService_List(t *testing.T) {
	t.Run("should propagate the repository's error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().List().Return(nil, wantErr)

		service := application.NewGeoDataService(repository, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.List()

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should return an empty list when no source is registered", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().List().Return(nil, nil)

		service := application.NewGeoDataService(repository, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		summaries, err := service.List()

		// then
		require.NoError(t, err)
		assert.Empty(t, summaries)
	})

	t.Run("should mark each source's availability using the file checker, leaving the others unaffected", func(t *testing.T) {
		// given
		present := domain.GeoDataSource{Name: "europa-mapa", Path: "/data/mapa.mbtiles"}
		missing := domain.GeoDataSource{Name: "europa-relevo", Path: "/data/relevo.tif"}

		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().List().Return([]domain.GeoDataSource{present, missing}, nil)

		fileChecker := mockdomain.NewMockFileChecker(mockCtrl)
		fileChecker.EXPECT().Exists("/data/mapa.mbtiles").Return(true)
		fileChecker.EXPECT().Exists("/data/relevo.tif").Return(false)

		service := application.NewGeoDataService(repository, nil, fileChecker, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		summaries, err := service.List()

		// then
		require.NoError(t, err)
		require.Len(t, summaries, 2)
		assert.Equal(t, present, summaries[0].Source)
		assert.True(t, summaries[0].Available)
		assert.Equal(t, missing, summaries[1].Source)
		assert.False(t, summaries[1].Available)
	})
}

func Test_geoDataService_Remove(t *testing.T) {
	t.Run("should reject a name that does not match any registered source", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().FindByName("nao-existe").Return(domain.GeoDataSource{}, false, nil)

		service := application.NewGeoDataService(repository, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		err := service.Remove("nao-existe")

		// then
		assert.ErrorIs(t, err, domain.ErrDataSourceNotRegistered)
	})

	t.Run("should propagate the repository's FindByName error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().FindByName(gomock.Any()).Return(domain.GeoDataSource{}, false, wantErr)

		service := application.NewGeoDataService(repository, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		err := service.Remove("x")

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should delete the registered source with the given name", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().FindByName("europa-mapa").Return(domain.GeoDataSource{Name: "europa-mapa"}, true, nil)
		repository.EXPECT().Delete("europa-mapa").Return(nil)

		service := application.NewGeoDataService(repository, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		err := service.Remove("europa-mapa")

		// then
		require.NoError(t, err)
	})

	t.Run("should propagate the repository's Delete error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().FindByName("europa-mapa").Return(domain.GeoDataSource{Name: "europa-mapa"}, true, nil)
		repository.EXPECT().Delete("europa-mapa").Return(wantErr)

		service := application.NewGeoDataService(repository, nil, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

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
		parser := mockdomain.NewMockTrackParser(mockCtrl)
		parser.EXPECT().Parse(gomock.Any()).Return(domain.Track{}, wantErr)

		service := application.NewGeoDataService(nil, nil, nil, parser, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.CheckCoverage(strings.NewReader(""))

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should propagate domain.CleanTrack's error unchanged", func(t *testing.T) {
		// given: domain/cleaning_test.go covers CleanTrack's own rules in
		// detail — this only checks the service does not swallow it.
		track := builddomain.NewTrackBuilder().WithPoints(
			builddomain.NewTrackPointBuilder().Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		parser := mockdomain.NewMockTrackParser(mockCtrl)
		parser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		service := application.NewGeoDataService(nil, nil, nil, parser, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.CheckCoverage(strings.NewReader(""))

		// then
		assert.ErrorIs(t, err, domain.ErrInsufficientPoints)
	})

	t.Run("should propagate the repository's List error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")
		track := builddomain.NewTrackBuilder().WithPoints(
			builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		parser := mockdomain.NewMockTrackParser(mockCtrl)
		parser.EXPECT().Parse(gomock.Any()).Return(track, nil)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().List().Return(nil, wantErr)

		service := application.NewGeoDataService(repository, nil, nil, parser, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.CheckCoverage(strings.NewReader(""))

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should hand the cleaned route and the registered sources to domain.ComputeCoverage", func(t *testing.T) {
		// given: ComputeCoverage's own coverage rules (winner selection,
		// segments, status) are covered in
		// internal/domain/geo_data_coverage_test.go — this only checks the
		// service cleans the track (same pipeline as InspectTrackService —
		// research.md item 9) and calls through with what the repository
		// reports.
		track := builddomain.NewTrackBuilder().WithPoints(
			builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		).Build()

		baseMap := builddomain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).Build()
		elevation := builddomain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).Build()

		mockCtrl := gomock.NewController(t)
		parser := mockdomain.NewMockTrackParser(mockCtrl)
		parser.EXPECT().Parse(gomock.Any()).Return(track, nil)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().List().Return([]domain.GeoDataSource{baseMap, elevation}, nil)
		fileChecker := mockdomain.NewMockFileChecker(mockCtrl)
		fileChecker.EXPECT().Exists(gomock.Any()).Return(true).AnyTimes()

		service := application.NewGeoDataService(repository, nil, fileChecker, parser, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		report, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.CoverageStatusFull, report.Status)
	})

	t.Run("should exclude a registered source whose file no longer exists before checking coverage", func(t *testing.T) {
		// given
		track := builddomain.NewTrackBuilder().WithPoints(
			builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
		).Build()

		baseMap := builddomain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).WithPath("/data/mapa.mbtiles").Build()
		elevation := builddomain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).WithPath("/data/relevo.tif").Build()

		mockCtrl := gomock.NewController(t)
		parser := mockdomain.NewMockTrackParser(mockCtrl)
		parser.EXPECT().Parse(gomock.Any()).Return(track, nil)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().List().Return([]domain.GeoDataSource{baseMap, elevation}, nil)
		fileChecker := mockdomain.NewMockFileChecker(mockCtrl)
		fileChecker.EXPECT().Exists("/data/mapa.mbtiles").Return(false)
		fileChecker.EXPECT().Exists("/data/relevo.tif").Return(true)

		service := application.NewGeoDataService(repository, nil, fileChecker, parser, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		report, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.CoverageStatusPartial, report.Status)
		require.Len(t, report.UncoveredSegments, 1)
		assert.Equal(t, domain.MissingBaseMap, report.UncoveredSegments[0].Missing)
		assert.Empty(t, report.BaseMapSourcesUsed)
	})
}
