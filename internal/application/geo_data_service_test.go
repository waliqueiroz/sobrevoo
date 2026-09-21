package application_test

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
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

		service := application.NewGeoDataService(repository, nil, nil, nil, nil)

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

		service := application.NewGeoDataService(repository, nil, nil, nil, nil)

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

		service := application.NewGeoDataService(repository, inspector, nil, nil, nil)

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

		service := application.NewGeoDataService(repository, inspector, nil, nil, nil)

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

		service := application.NewGeoDataService(repository, inspector, nil, nil, nil)

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

		service := application.NewGeoDataService(repository, nil, nil, nil, nil)

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

		service := application.NewGeoDataService(repository, nil, nil, nil, nil)

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

		service := application.NewGeoDataService(repository, nil, fileChecker, nil, nil)

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

		service := application.NewGeoDataService(repository, nil, nil, nil, nil)

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

		service := application.NewGeoDataService(repository, nil, nil, nil, nil)

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

		service := application.NewGeoDataService(repository, nil, nil, nil, nil)

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

		service := application.NewGeoDataService(repository, nil, nil, nil, nil)

		// when
		err := service.Remove("europa-mapa")

		// then
		assert.ErrorIs(t, err, wantErr)
	})
}

func Test_geoDataService_CheckCoverage(t *testing.T) {
	t.Run("should propagate TrackService.Clean's error unchanged", func(t *testing.T) {
		// given
		wantErr := domain.ErrEmptyFile

		mockCtrl := gomock.NewController(t)
		trackService := mockapplication.NewMockTrackService(mockCtrl)
		trackService.EXPECT().Clean(gomock.Any()).Return(domain.CleanedTrack{}, wantErr)

		service := application.NewGeoDataService(nil, nil, nil, trackService, nil)

		// when
		_, err := service.CheckCoverage(strings.NewReader(""))

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should propagate the repository's List error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		trackService := mockapplication.NewMockTrackService(mockCtrl)
		trackService.EXPECT().Clean(gomock.Any()).Return(cleanedCoverageTrack(), nil)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().List().Return(nil, wantErr)

		service := application.NewGeoDataService(repository, nil, nil, trackService, nil)

		// when
		_, err := service.CheckCoverage(strings.NewReader(""))

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should hand the cleaned route and the registered sources to domain.ComputeCoverage", func(t *testing.T) {
		// given: ComputeCoverage's own coverage rules (winner selection,
		// segments, status) are covered in
		// internal/domain/geo_data_coverage_test.go — this only checks the
		// service asks TrackService for the cleaned (not simplified) track
		// — research.md item 9 — and calls through with what the repository
		// reports.
		baseMap := builddomain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).Build()
		elevation := builddomain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).Build()

		mockCtrl := gomock.NewController(t)
		trackService := mockapplication.NewMockTrackService(mockCtrl)
		trackService.EXPECT().Clean(gomock.Any()).Return(cleanedCoverageTrack(), nil)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().List().Return([]domain.GeoDataSource{baseMap, elevation}, nil)
		fileChecker := mockdomain.NewMockFileChecker(mockCtrl)
		fileChecker.EXPECT().Exists(gomock.Any()).Return(true).AnyTimes()

		service := application.NewGeoDataService(repository, nil, fileChecker, trackService, nil)

		// when
		report, err := service.CheckCoverage(strings.NewReader(""))

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.CoverageStatusFull, report.Status)
	})

	t.Run("should exclude a registered source whose file no longer exists before checking coverage", func(t *testing.T) {
		// given
		baseMap := builddomain.NewGeoDataSourceBuilder().WithName("europa-mapa").WithType(domain.DataTypeBaseMap).WithPath("/data/mapa.mbtiles").Build()
		elevation := builddomain.NewGeoDataSourceBuilder().WithName("europa-relevo").WithType(domain.DataTypeElevation).WithPath("/data/relevo.tif").Build()

		mockCtrl := gomock.NewController(t)
		trackService := mockapplication.NewMockTrackService(mockCtrl)
		trackService.EXPECT().Clean(gomock.Any()).Return(cleanedCoverageTrack(), nil)
		repository := mockdomain.NewMockGeoDataRepository(mockCtrl)
		repository.EXPECT().List().Return([]domain.GeoDataSource{baseMap, elevation}, nil)
		fileChecker := mockdomain.NewMockFileChecker(mockCtrl)
		fileChecker.EXPECT().Exists("/data/mapa.mbtiles").Return(false)
		fileChecker.EXPECT().Exists("/data/relevo.tif").Return(true)

		service := application.NewGeoDataService(repository, nil, fileChecker, trackService, nil)

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

// cleanedCoverageTrack is a short cleaned track around (45,15)-(46,16),
// shared by the CheckCoverage scenarios.
func cleanedCoverageTrack() domain.CleanedTrack {
	points := []domain.TrackPoint{
		builddomain.NewTrackPointBuilder().WithLatitude(45).WithLongitude(15).Build(),
		builddomain.NewTrackPointBuilder().WithLatitude(46).WithLongitude(16).Build(),
	}
	return domain.CleanedTrack{Track: builddomain.NewTrackBuilder().WithPoints(points...).Build(), Route: domain.Route{Points: points}}
}

// reliefWorld is the area a relief in these tests covers.
var reliefWorld = domain.BoundingBox{MinLatitude: 40, MaxLatitude: 41, MinLongitude: 10, MaxLongitude: 11}

// an 8 × 8 grid of 0.125° cells over reliefWorld, exactly representable
var reliefWorldInfo = domain.ElevationGridInfo{
	Rows: 8, Cols: 8,
	NorthLatitude: 41, WestLongitude: 10,
	CellLatitude: 0.125, CellLongitude: 0.125,
	UnitToMeters: 1,
}

func Test_geoDataService_ElevationAt(t *testing.T) {
	type mocks struct {
		repository  *mockdomain.MockGeoDataRepository
		fileChecker *mockdomain.MockFileChecker
		reader      *mockdomain.MockElevationReader
		service     application.GeoDataService
	}
	newMocks := func(t *testing.T) mocks {
		mockCtrl := gomock.NewController(t)
		m := mocks{
			repository:  mockdomain.NewMockGeoDataRepository(mockCtrl),
			fileChecker: mockdomain.NewMockFileChecker(mockCtrl),
			reader:      mockdomain.NewMockElevationReader(mockCtrl),
		}
		m.service = application.NewGeoDataService(m.repository, nil, m.fileChecker, nil, m.reader)
		return m
	}
	relief := func(name string, area domain.BoundingBox) domain.GeoDataSource {
		return builddomain.NewGeoDataSourceBuilder().WithName(name).WithPath("/data/" + name + ".tif").
			WithType(domain.DataTypeElevation).WithFormat(domain.DataFormatGeoTIFF).WithBoundingBox(area).Build()
	}

	t.Run("should read the cell that contains the point, in meters, and say which source and cell it is", func(t *testing.T) {
		// given
		m := newMocks(t)
		dem := relief("dem", reliefWorld)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{dem}, nil)
		m.fileChecker.EXPECT().Exists(dem.Path).Return(true)
		m.reader.EXPECT().Describe(dem.Path).Return(reliefWorldInfo, nil)
		m.reader.EXPECT().ReadWindow(dem.Path, domain.GridWindow{FirstRow: 3, FirstCol: 2, Rows: 1, Cols: 1}).
			Return(domain.ElevationWindow{Values: []float32{760}}, nil)

		// when
		reading, err := m.service.ElevationAt(40.55, 10.35)

		// then
		require.NoError(t, err)
		assert.True(t, reading.HasValue)
		assert.Equal(t, 760.0, reading.Meters)
		assert.Equal(t, "dem", reading.Source.Name)
		assert.Equal(t, 3, reading.Row)
		assert.Equal(t, 2, reading.Col)
		assert.Equal(t, 40.55, reading.Latitude)
		assert.Equal(t, 10.35, reading.Longitude)
	})

	t.Run("should say the file has no value for the cell, rather than fail or give zero", func(t *testing.T) {
		// given
		m := newMocks(t)
		dem := relief("dem", reliefWorld)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{dem}, nil)
		m.fileChecker.EXPECT().Exists(gomock.Any()).Return(true)
		m.reader.EXPECT().Describe(gomock.Any()).Return(reliefWorldInfo, nil)
		m.reader.EXPECT().ReadWindow(gomock.Any(), gomock.Any()).
			Return(domain.ElevationWindow{Values: []float32{float32(math.NaN())}}, nil)

		// when
		reading, err := m.service.ElevationAt(40.55, 10.35)

		// then
		require.NoError(t, err)
		assert.False(t, reading.HasValue)
		assert.Equal(t, 0.0, reading.Meters)
	})

	t.Run("should pick the smaller of two reliefs that cover the point", func(t *testing.T) {
		// given
		m := newMocks(t)
		big := relief("big", domain.BoundingBox{MinLatitude: 30, MaxLatitude: 50, MinLongitude: 0, MaxLongitude: 20})
		small := relief("small", reliefWorld)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{big, small}, nil)
		m.fileChecker.EXPECT().Exists(gomock.Any()).Return(true).AnyTimes()
		m.reader.EXPECT().Describe(small.Path).Return(reliefWorldInfo, nil)
		m.reader.EXPECT().ReadWindow(small.Path, gomock.Any()).Return(domain.ElevationWindow{Values: []float32{5}}, nil)

		// when
		reading, err := m.service.ElevationAt(40.55, 10.35)

		// then
		require.NoError(t, err)
		assert.Equal(t, "small", reading.Source.Name)
	})

	t.Run("should ignore a relief whose file is no longer there", func(t *testing.T) {
		// given
		m := newMocks(t)
		gone := relief("gone", reliefWorld)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{gone}, nil)
		m.fileChecker.EXPECT().Exists(gone.Path).Return(false)

		// when
		_, err := m.service.ElevationAt(40.55, 10.35)

		// then
		assert.ErrorIs(t, err, domain.ErrElevationNotCovered)
	})

	t.Run("should not consider base maps", func(t *testing.T) {
		// given
		m := newMocks(t)
		baseMap := builddomain.NewGeoDataSourceBuilder().WithType(domain.DataTypeBaseMap).WithBoundingBox(reliefWorld).Build()
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{baseMap}, nil)
		m.fileChecker.EXPECT().Exists(gomock.Any()).Return(true)

		// when
		_, err := m.service.ElevationAt(40.55, 10.35)

		// then
		assert.ErrorIs(t, err, domain.ErrElevationNotCovered)
	})

	t.Run("should say that no relief covers a point outside every relief", func(t *testing.T) {
		// given
		m := newMocks(t)
		dem := relief("dem", reliefWorld)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{dem}, nil)
		m.fileChecker.EXPECT().Exists(gomock.Any()).Return(true)

		// when
		_, err := m.service.ElevationAt(0, 0)

		// then
		assert.ErrorIs(t, err, domain.ErrElevationNotCovered)
	})

	t.Run("should refuse a coordinate out of range without looking at the registry", func(t *testing.T) {
		// given: mocks with no expectations fail the test if they are called
		m := newMocks(t)

		// when
		_, err := m.service.ElevationAt(91, 0)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidCoordinate)
	})

	t.Run("should give the same answer for longitude 180 and -180", func(t *testing.T) {
		// given
		crossing := relief("crossing", domain.BoundingBox{MinLatitude: -1, MaxLatitude: 1, MinLongitude: 170, MaxLongitude: -170, CrossesAntimeridian: true})
		info := domain.ElevationGridInfo{Rows: 8, Cols: 160, NorthLatitude: 1, WestLongitude: 170, CellLatitude: 0.25, CellLongitude: 0.125, UnitToMeters: 1}
		m := newMocks(t)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{crossing}, nil).Times(2)
		m.fileChecker.EXPECT().Exists(gomock.Any()).Return(true).Times(2)
		m.reader.EXPECT().Describe(gomock.Any()).Return(info, nil).Times(2)
		m.reader.EXPECT().ReadWindow(gomock.Any(), gomock.Any()).Return(domain.ElevationWindow{Values: []float32{42}}, nil).Times(2)

		// when
		east, eastErr := m.service.ElevationAt(0.3, 180)
		west, westErr := m.service.ElevationAt(0.3, -180)

		// then
		require.NoError(t, eastErr)
		require.NoError(t, westErr)
		assert.Equal(t, east.Row, west.Row)
		assert.Equal(t, east.Col, west.Col)
		assert.Equal(t, east.Meters, west.Meters)
		assert.Equal(t, -180.0, east.Longitude)
	})

	t.Run("should read the same cell a slice would, so the values agree", func(t *testing.T) {
		// given
		m := newMocks(t)
		dem := relief("dem", reliefWorld)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{dem}, nil)
		m.fileChecker.EXPECT().Exists(gomock.Any()).Return(true)
		m.reader.EXPECT().Describe(gomock.Any()).Return(reliefWorldInfo, nil)
		var asked domain.GridWindow
		m.reader.EXPECT().ReadWindow(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, w domain.GridWindow) (domain.ElevationWindow, error) {
				asked = w
				return domain.ElevationWindow{Values: []float32{321}}, nil
			})

		// when
		reading, err := m.service.ElevationAt(40.55, 10.35)

		// then: the cell a slice of a region holding the point would give for it
		require.NoError(t, err)
		area := domain.BoundingBox{MinLatitude: 40.2, MaxLatitude: 40.9, MinLongitude: 10.1, MaxLongitude: 10.9}
		window := reliefWorldInfo.Window(area, area)[0]
		grid := domain.NewElevationGrid(dem, window, reliefWorldInfo, make([]float32, window.Rows*window.Cols))
		assert.Equal(t, 1, asked.Rows*asked.Cols)
		assert.GreaterOrEqual(t, asked.FirstRow, window.FirstRow)
		assert.Less(t, asked.FirstRow, window.FirstRow+grid.Rows())
		assert.GreaterOrEqual(t, asked.FirstCol, window.FirstCol)
		assert.Less(t, asked.FirstCol, window.FirstCol+grid.Cols())
		assert.Equal(t, asked.FirstRow, reading.Row)
		assert.Equal(t, asked.FirstCol, reading.Col)
	})

	t.Run("should propagate the repository's error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")
		m := newMocks(t)
		m.repository.EXPECT().List().Return(nil, wantErr)

		// when
		_, err := m.service.ElevationAt(40.55, 10.35)

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should name the relief whose content cannot be read", func(t *testing.T) {
		// given
		m := newMocks(t)
		dem := relief("dem", reliefWorld)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{dem}, nil)
		m.fileChecker.EXPECT().Exists(gomock.Any()).Return(true)
		m.reader.EXPECT().Describe(gomock.Any()).Return(reliefWorldInfo, nil)
		m.reader.EXPECT().ReadWindow(gomock.Any(), gomock.Any()).
			Return(domain.ElevationWindow{}, fmt.Errorf("%w: cut off", domain.ErrGeoDataContentUnreadable))

		// when
		_, err := m.service.ElevationAt(40.55, 10.35)

		// then
		require.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
		assert.ErrorContains(t, err, `"dem"`)
	})

	t.Run("should name the relief whose unit is not supported", func(t *testing.T) {
		// given
		m := newMocks(t)
		dem := relief("dem", reliefWorld)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{dem}, nil)
		m.fileChecker.EXPECT().Exists(gomock.Any()).Return(true)
		m.reader.EXPECT().Describe(gomock.Any()).Return(domain.ElevationGridInfo{}, fmt.Errorf("%w: unit 9999", domain.ErrElevationUnitUnsupported))

		// when
		_, err := m.service.ElevationAt(40.55, 10.35)

		// then
		require.ErrorIs(t, err, domain.ErrElevationUnitUnsupported)
		assert.ErrorContains(t, err, `"dem"`)
	})
}
