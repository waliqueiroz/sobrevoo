package application_test

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mockdomain"
)

// slicePlan is a plan of one frame, 1 km from its marker, in São Paulo: an
// area of about 0.02° on each side.
func slicePlan() domain.CameraPlan {
	frame := builddomain.NewCameraFrameBuilder().
		WithMarkerPosition(-23.55, -46.63).
		WithCameraPosition(-23.554, -46.63).
		WithCameraToMarkerDistance(1000).
		Build()
	parameters := builddomain.NewPlanParametersBuilder().WithDuration(time.Second / 30).WithFrameRate(30).Build()
	return builddomain.NewCameraPlanBuilder().WithParameters(parameters).WithFrames(frame).Build()
}

func sliceBox(minLat, maxLat, minLon, maxLon float64) domain.BoundingBox {
	return domain.BoundingBox{MinLatitude: minLat, MaxLatitude: maxLat, MinLongitude: minLon, MaxLongitude: maxLon}
}

func baseMapSource(name string, area domain.BoundingBox) domain.GeoDataSource {
	return builddomain.NewGeoDataSourceBuilder().
		WithName(name).WithPath("/data/" + name + ".mbtiles").
		WithType(domain.DataTypeBaseMap).WithFormat(domain.DataFormatMBTiles).WithBoundingBox(area).Build()
}

func reliefSource(name string, area domain.BoundingBox) domain.GeoDataSource {
	return builddomain.NewGeoDataSourceBuilder().
		WithName(name).WithPath("/data/" + name + ".tif").
		WithType(domain.DataTypeElevation).WithFormat(domain.DataFormatGeoTIFF).WithBoundingBox(area).Build()
}

var (
	wholeWorldish = sliceBox(-30, -20, -50, -40)

	// a grid of 0.001° cells over the whole of wholeWorldish
	reliefInfo = domain.ElevationGridInfo{
		Rows: 10000, Cols: 10000,
		NorthLatitude: -20, WestLongitude: -50,
		CellLatitude: 0.001, CellLongitude: 0.001,
		UnitToMeters: 1,
	}
)

type sliceMocks struct {
	repository      *mockdomain.MockGeoDataRepository
	fileChecker     *mockdomain.MockFileChecker
	baseMapReader   *mockdomain.MockBaseMapReader
	elevationReader *mockdomain.MockElevationReader
	exporter        *mockdomain.MockGeoSliceExporter
	service         application.GeoSliceService
}

func newSliceMocks(t *testing.T) sliceMocks {
	t.Helper()
	return newSliceMocksWithTuning(t, builddomain.NewSliceTuningBuilder().Build())
}

func newSliceMocksWithTuning(t *testing.T, tuning domain.SliceTuning) sliceMocks {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	m := sliceMocks{
		repository:      mockdomain.NewMockGeoDataRepository(mockCtrl),
		fileChecker:     mockdomain.NewMockFileChecker(mockCtrl),
		baseMapReader:   mockdomain.NewMockBaseMapReader(mockCtrl),
		elevationReader: mockdomain.NewMockElevationReader(mockCtrl),
		exporter:        mockdomain.NewMockGeoSliceExporter(mockCtrl),
	}
	m.service = application.NewGeoSliceService(
		m.repository, m.fileChecker, m.baseMapReader, m.elevationReader, m.exporter,
		tuning, builddomain.NewCameraTuningBuilder().Build(),
	)
	return m
}

// allFilesExist makes every registered file exist.
func (m sliceMocks) allFilesExist() {
	m.fileChecker.EXPECT().Exists(gomock.Any()).Return(true).AnyTimes()
}

// tilesFor answers a request for tiles with a tile of one byte for each of them.
func tilesFor(ids []domain.TileID) domain.TileRead {
	read := domain.TileRead{Format: "png"}
	for _, id := range ids {
		read.Tiles = append(read.Tiles, domain.Tile{ID: id, Data: []byte{byte(id.X), byte(id.Y)}})
	}
	return read
}

func samplesFor(window domain.GridWindow) domain.ElevationWindow {
	values := make([]float32, window.Rows*window.Cols)
	for i := range values {
		values[i] = 100
	}
	return domain.ElevationWindow{Values: values}
}

func Test_geoSliceService_Generate(t *testing.T) {
	tuning := builddomain.NewSliceTuningBuilder().Build()
	cameraTuning := builddomain.NewCameraTuningBuilder().Build()
	plan := slicePlan()
	area := plan.AreaOfInterest(tuning)

	t.Run("should read the tiles and the samples under the area of the plan, from the sources that cover it", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		m.allFilesExist()
		baseMap := baseMapSource("map", wholeWorldish)
		relief := reliefSource("dem", wholeWorldish)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{baseMap, relief}, nil)

		wantDetail := tuning.DetailLevel(plan.Summary.MinCameraDistance, cameraTuning.OverviewVerticalFOVDegrees, area, domain.LevelRange{Min: 0, Max: 16})
		require.Equal(t, 16, wantDetail.Chosen, "the level asked for is above what the map offers")
		m.baseMapReader.EXPECT().Levels(baseMap.Path).Return(domain.LevelRange{Min: 0, Max: 16}, nil)
		var requested []domain.TileID
		m.baseMapReader.EXPECT().ReadTiles(baseMap.Path, 16, gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) {
				requested = ids
				return tilesFor(ids), nil
			})

		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		wantWindow := reliefInfo.Window(area, area)[0]
		m.elevationReader.EXPECT().ReadWindow(relief.Path, wantWindow).Return(samplesFor(wantWindow), nil)

		// when
		slice, err := m.service.Generate(plan)

		// then
		require.NoError(t, err)
		assert.Equal(t, area, slice.Area)

		require.Len(t, slice.TileSets, 1)
		assert.Equal(t, "map", slice.TileSets[0].Source.Name)
		assert.Equal(t, 16, slice.TileSets[0].Detail.Chosen)
		assert.Equal(t, wantDetail.Ideal, slice.TileSets[0].Detail.Ideal)
		assert.Equal(t, "png", slice.TileSets[0].Format)
		assert.Len(t, slice.TileSets[0].Tiles, len(requested))
		var wantCount int
		for _, r := range area.TileRange(16) {
			wantCount += (r.MaxX - r.MinX + 1) * (r.MaxY - r.MinY + 1)
		}
		assert.Len(t, requested, wantCount)

		require.Len(t, slice.Elevation, 1)
		assert.Equal(t, "dem", slice.Elevation[0].Source.Name)
		assert.Equal(t, wantWindow, slice.Elevation[0].Window)
		meters, hasValue := slice.Elevation[0].At(0, 0)
		assert.True(t, hasValue)
		assert.Equal(t, 100.0, meters)
	})

	t.Run("should ignore a registered source whose file is no longer there", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		gone := baseMapSource("gone", wholeWorldish)
		present := baseMapSource("present", wholeWorldish)
		relief := reliefSource("dem", wholeWorldish)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{gone, present, relief}, nil)
		m.fileChecker.EXPECT().Exists(gone.Path).Return(false).AnyTimes()
		m.fileChecker.EXPECT().Exists(present.Path).Return(true).AnyTimes()
		m.fileChecker.EXPECT().Exists(relief.Path).Return(true).AnyTimes()
		m.baseMapReader.EXPECT().Levels(present.Path).Return(domain.LevelRange{Min: 0, Max: 16}, nil)
		m.baseMapReader.EXPECT().ReadTiles(present.Path, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) { return tilesFor(ids), nil })
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		m.elevationReader.EXPECT().ReadWindow(relief.Path, gomock.Any()).
			DoAndReturn(func(_ string, w domain.GridWindow) (domain.ElevationWindow, error) { return samplesFor(w), nil })

		// when
		slice, err := m.service.Generate(plan)

		// then
		require.NoError(t, err)
		require.Len(t, slice.TileSets, 1)
		assert.Equal(t, "present", slice.TileSets[0].Source.Name)
	})

	t.Run("should refuse an area with no elevation, without reading anything", func(t *testing.T) {
		// given: readers with no expectations fail the test if they are called
		m := newSliceMocks(t)
		m.allFilesExist()
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{baseMapSource("map", wholeWorldish)}, nil)

		// when
		_, err := m.service.Generate(plan)

		// then
		require.ErrorIs(t, err, domain.ErrAreaNotCovered)
		var notCovered *domain.AreaNotCoveredError
		require.ErrorAs(t, err, &notCovered)
		require.Len(t, notCovered.Report.UncoveredSegments, 1)
		assert.Equal(t, domain.MissingElevation, notCovered.Report.UncoveredSegments[0].Missing)
	})

	t.Run("should refuse an area with no base map, without reading anything", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		m.allFilesExist()
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{reliefSource("dem", wholeWorldish)}, nil)

		// when
		_, err := m.service.Generate(plan)

		// then
		var notCovered *domain.AreaNotCoveredError
		require.ErrorAs(t, err, &notCovered)
		assert.Equal(t, domain.MissingBaseMap, notCovered.Report.UncoveredSegments[0].Missing)
	})

	t.Run("should refuse an area that is only partly covered, saying which part", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		m.allFilesExist()
		westOnly := reliefSource("west-only", sliceBox(-30, -20, -50, -46.63))
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{baseMapSource("map", wholeWorldish), westOnly}, nil)

		// when
		_, err := m.service.Generate(plan)

		// then
		var notCovered *domain.AreaNotCoveredError
		require.ErrorAs(t, err, &notCovered)
		assert.Equal(t, domain.CoverageStatusPartial, notCovered.Report.Status)
		require.Len(t, notCovered.Report.UncoveredSegments, 1)
		assert.Greater(t, notCovered.Report.UncoveredSegments[0].StartLongitude, -46.63)
	})

	t.Run("should read the elevation of each region from the source that wins there", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		m.allFilesExist()
		baseMap := baseMapSource("map", wholeWorldish)
		west := reliefSource("west", sliceBox(-30, -20, -50, -46.63))
		east := reliefSource("east", sliceBox(-30, -20, -46.63, -40))
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{baseMap, east, west}, nil)
		m.baseMapReader.EXPECT().Levels(gomock.Any()).Return(domain.LevelRange{Min: 0, Max: 16}, nil)
		m.baseMapReader.EXPECT().ReadTiles(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) { return tilesFor(ids), nil })
		m.elevationReader.EXPECT().Describe(west.Path).Return(reliefInfo, nil).Times(1)
		m.elevationReader.EXPECT().Describe(east.Path).Return(reliefInfo, nil).Times(1)
		m.elevationReader.EXPECT().ReadWindow(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, w domain.GridWindow) (domain.ElevationWindow, error) { return samplesFor(w), nil }).Times(2)

		// when
		slice, err := m.service.Generate(plan)

		// then
		require.NoError(t, err)
		require.Len(t, slice.Elevation, 2)
		assert.Equal(t, "west", slice.Elevation[0].Source.Name)
		assert.Equal(t, "east", slice.Elevation[1].Source.Name)
		assert.Less(t, slice.Elevation[0].Window.FirstCol, slice.Elevation[1].Window.FirstCol)
	})

	t.Run("should limit the level of each base map to what that map offers", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		m.allFilesExist()
		west := baseMapSource("map-a", sliceBox(-30, -20, -50, -46.63))
		east := baseMapSource("map-b", sliceBox(-30, -20, -46.63, -40))
		relief := reliefSource("dem", wholeWorldish)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{east, west, relief}, nil)
		m.baseMapReader.EXPECT().Levels(west.Path).Return(domain.LevelRange{Min: 0, Max: 14}, nil)
		m.baseMapReader.EXPECT().Levels(east.Path).Return(domain.LevelRange{Min: 0, Max: 22}, nil)
		var levelsRead []int
		m.baseMapReader.EXPECT().ReadTiles(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, level int, ids []domain.TileID) (domain.TileRead, error) {
				levelsRead = append(levelsRead, level)
				return tilesFor(ids), nil
			}).Times(2)
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		// the two maps split the area in two regions, and the relief is read for each
		m.elevationReader.EXPECT().ReadWindow(relief.Path, gomock.Any()).
			DoAndReturn(func(_ string, w domain.GridWindow) (domain.ElevationWindow, error) { return samplesFor(w), nil }).Times(2)

		// when
		slice, err := m.service.Generate(plan)

		// then
		require.NoError(t, err)
		require.Len(t, slice.TileSets, 2)
		assert.Equal(t, "map-a", slice.TileSets[0].Source.Name)
		assert.Equal(t, 14, slice.TileSets[0].Detail.Chosen)
		assert.Equal(t, "map-b", slice.TileSets[1].Source.Name)
		assert.Equal(t, slice.TileSets[1].Detail.Ideal, slice.TileSets[1].Detail.Chosen)
		assert.Greater(t, slice.TileSets[1].Detail.Chosen, 14)
		assert.ElementsMatch(t, []int{14, slice.TileSets[1].Detail.Chosen}, levelsRead)
	})

	t.Run("should not ask two base maps for the same tile", func(t *testing.T) {
		// given: both maps end up at level 14, and a tile straddles the limit between them
		m := newSliceMocks(t)
		m.allFilesExist()
		west := baseMapSource("map-a", sliceBox(-30, -20, -50, -46.63))
		east := baseMapSource("map-b", sliceBox(-30, -20, -46.63, -40))
		relief := reliefSource("dem", wholeWorldish)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{west, east, relief}, nil)
		m.baseMapReader.EXPECT().Levels(gomock.Any()).Return(domain.LevelRange{Min: 0, Max: 14}, nil).Times(2)
		asked := map[domain.TileID]int{}
		m.baseMapReader.EXPECT().ReadTiles(gomock.Any(), 14, gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) {
				for _, id := range ids {
					asked[id]++
				}
				return tilesFor(ids), nil
			}).Times(2)
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		m.elevationReader.EXPECT().ReadWindow(relief.Path, gomock.Any()).
			DoAndReturn(func(_ string, w domain.GridWindow) (domain.ElevationWindow, error) { return samplesFor(w), nil }).Times(2)

		// when
		_, err := m.service.Generate(plan)

		// then
		require.NoError(t, err)
		for id, times := range asked {
			assert.Equal(t, 1, times, "tile %+v", id)
		}
	})

	t.Run("should give the same slice every time", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		m.allFilesExist()
		baseMap := baseMapSource("map", wholeWorldish)
		relief := reliefSource("dem", wholeWorldish)
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{baseMap, relief}, nil).AnyTimes()
		m.baseMapReader.EXPECT().Levels(gomock.Any()).Return(domain.LevelRange{Min: 0, Max: 16}, nil).AnyTimes()
		m.baseMapReader.EXPECT().ReadTiles(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) { return tilesFor(ids), nil }).AnyTimes()
		m.elevationReader.EXPECT().Describe(gomock.Any()).Return(reliefInfo, nil).AnyTimes()
		m.elevationReader.EXPECT().ReadWindow(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, w domain.GridWindow) (domain.ElevationWindow, error) { return samplesFor(w), nil }).AnyTimes()
		expected, err := m.service.Generate(plan)
		require.NoError(t, err)

		for i := 0; i < 100; i++ {
			// when
			slice, err := m.service.Generate(plan)

			// then
			require.NoError(t, err)
			assert.Equal(t, expected, slice)
		}
	})
}

func Test_geoSliceService_Generate_Errors(t *testing.T) {
	plan := slicePlan()
	baseMap := baseMapSource("map", wholeWorldish)
	relief := reliefSource("dem", wholeWorldish)
	boom := errors.New("boom")

	givenSources := func(m sliceMocks) {
		m.allFilesExist()
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{baseMap, relief}, nil)
	}

	t.Run("should propagate the repository's error unchanged", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		m.repository.EXPECT().List().Return(nil, boom)

		// when
		_, err := m.service.Generate(plan)

		// then
		assert.ErrorIs(t, err, boom)
	})

	t.Run("should propagate the error of reading the levels of a base map", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		givenSources(m)
		m.baseMapReader.EXPECT().Levels(baseMap.Path).Return(domain.LevelRange{}, boom)

		// when
		_, err := m.service.Generate(plan)

		// then
		assert.ErrorIs(t, err, boom)
	})

	t.Run("should propagate the error of reading the tiles", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		givenSources(m)
		m.baseMapReader.EXPECT().Levels(baseMap.Path).Return(domain.LevelRange{Min: 0, Max: 16}, nil)
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		m.baseMapReader.EXPECT().ReadTiles(baseMap.Path, gomock.Any(), gomock.Any()).Return(domain.TileRead{}, boom)

		// when
		_, err := m.service.Generate(plan)

		// then
		assert.ErrorIs(t, err, boom)
	})

	t.Run("should propagate the error of describing an elevation grid", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		givenSources(m)
		m.baseMapReader.EXPECT().Levels(baseMap.Path).Return(domain.LevelRange{Min: 0, Max: 16}, nil)
		m.elevationReader.EXPECT().Describe(relief.Path).Return(domain.ElevationGridInfo{}, boom)

		// when
		_, err := m.service.Generate(plan)

		// then
		assert.ErrorIs(t, err, boom)
	})

	t.Run("should propagate the error of reading the samples", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		givenSources(m)
		m.baseMapReader.EXPECT().Levels(baseMap.Path).Return(domain.LevelRange{Min: 0, Max: 16}, nil)
		m.baseMapReader.EXPECT().ReadTiles(baseMap.Path, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) { return tilesFor(ids), nil })
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		m.elevationReader.EXPECT().ReadWindow(relief.Path, gomock.Any()).Return(domain.ElevationWindow{}, boom)

		// when
		_, err := m.service.Generate(plan)

		// then
		assert.ErrorIs(t, err, boom)
	})
}

func Test_geoSliceService_Export(t *testing.T) {
	slice := builddomain.NewGeoSliceBuilder().Build()

	t.Run("should hand the slice, the path and the overwrite choice to the exporter", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		m.exporter.EXPECT().Export(slice, "out.zip", true).Return(nil)

		// when
		err := m.service.Export(slice, "out.zip", true)

		// then
		assert.NoError(t, err)
	})

	t.Run("should not overwrite unless asked to", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		m.exporter.EXPECT().Export(slice, "out.zip", false).Return(nil)

		// when
		err := m.service.Export(slice, "out.zip", false)

		// then
		assert.NoError(t, err)
	})

	t.Run("should return the exporter's error unchanged", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		m.exporter.EXPECT().Export(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.ErrSliceDestinationExists)

		// when
		err := m.service.Export(slice, "out.zip", false)

		// then
		assert.ErrorIs(t, err, domain.ErrSliceDestinationExists)
	})
}

func Test_geoSliceService_Generate_Robustness(t *testing.T) {
	plan := slicePlan()
	baseMap := baseMapSource("map", wholeWorldish)
	relief := reliefSource("dem", wholeWorldish)

	givenSources := func(m sliceMocks) {
		m.allFilesExist()
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{baseMap, relief}, nil)
		m.baseMapReader.EXPECT().Levels(baseMap.Path).Return(domain.LevelRange{Min: 0, Max: 16}, nil)
	}

	t.Run("should carry on when tiles are missing from the base map, recording which", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		givenSources(m)
		gone := domain.TileID{Level: 16, X: 1, Y: 2}
		m.baseMapReader.EXPECT().ReadTiles(baseMap.Path, 16, gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) {
				read := tilesFor(ids[1:])
				read.Missing = []domain.TileID{gone}
				return read, nil
			})
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		m.elevationReader.EXPECT().ReadWindow(relief.Path, gomock.Any()).
			DoAndReturn(func(_ string, w domain.GridWindow) (domain.ElevationWindow, error) { return samplesFor(w), nil })

		// when
		slice, err := m.service.Generate(plan)

		// then
		require.NoError(t, err)
		assert.Equal(t, []domain.TileID{gone}, slice.TileSets[0].Missing)
		assert.Equal(t, 1, slice.Summary.MissingTileCount)
		assert.Greater(t, slice.Summary.TileCount, 0)
	})

	t.Run("should still give a slice when every tile is missing", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		givenSources(m)
		m.baseMapReader.EXPECT().ReadTiles(baseMap.Path, 16, gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) {
				return domain.TileRead{Format: "png", Missing: ids}, nil
			})
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		m.elevationReader.EXPECT().ReadWindow(relief.Path, gomock.Any()).
			DoAndReturn(func(_ string, w domain.GridWindow) (domain.ElevationWindow, error) { return samplesFor(w), nil })

		// when
		slice, err := m.service.Generate(plan)

		// then
		require.NoError(t, err)
		assert.Zero(t, slice.Summary.TileCount)
		assert.Greater(t, slice.Summary.MissingTileCount, 0)
	})

	t.Run("should keep the samples without value as such", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		givenSources(m)
		m.baseMapReader.EXPECT().ReadTiles(baseMap.Path, 16, gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) { return tilesFor(ids), nil })
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		m.elevationReader.EXPECT().ReadWindow(relief.Path, gomock.Any()).
			DoAndReturn(func(_ string, w domain.GridWindow) (domain.ElevationWindow, error) {
				window := samplesFor(w)
				window.Values[0], window.Values[1] = float32(math.NaN()), float32(math.NaN())
				return window, nil
			})

		// when
		slice, err := m.service.Generate(plan)

		// then
		require.NoError(t, err)
		assert.Equal(t, 2, slice.Summary.NoValueSampleCount)
	})

	t.Run("should refuse a slice whose estimate is over the limit before reading any content", func(t *testing.T) {
		// given: a limit that even the estimate of the tiles alone goes past; the readers have no ReadTiles or ReadWindow expectations
		m := newSliceMocksWithTuning(t, builddomain.NewSliceTuningBuilder().WithMaxSizeBytes(1024).Build())
		givenSources(m)
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)

		// when
		_, err := m.service.Generate(plan)

		// then
		require.ErrorIs(t, err, domain.ErrSliceTooLarge)
		assert.ErrorContains(t, err, "level 16")
	})

	t.Run("should count the samples in the estimate", func(t *testing.T) {
		// given: tiles are estimated at one byte, so only the samples can go past the limit
		tuning := builddomain.NewSliceTuningBuilder().WithEstimatedTileBytes(1).WithMaxSizeBytes(1000).Build()
		m := newSliceMocksWithTuning(t, tuning)
		givenSources(m)
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)

		// when
		_, err := m.service.Generate(plan)

		// then
		assert.ErrorIs(t, err, domain.ErrSliceTooLarge)
	})

	t.Run("should refuse a slice whose real size goes past the limit while reading", func(t *testing.T) {
		// given: the estimate fits (tiles assumed at one byte) but the tiles read are big
		tuning := builddomain.NewSliceTuningBuilder().WithEstimatedTileBytes(1).WithMaxSizeBytes(1_000_000).Build()
		m := newSliceMocksWithTuning(t, tuning)
		givenSources(m)
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		m.baseMapReader.EXPECT().ReadTiles(baseMap.Path, 16, gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) {
				return domain.TileRead{Format: "png", Tiles: []domain.Tile{{ID: ids[0], Data: make([]byte, 2_000_000)}}}, nil
			})

		// when
		_, err := m.service.Generate(plan)

		// then
		assert.ErrorIs(t, err, domain.ErrSliceTooLarge)
	})

	t.Run("should accept a slice that fits the limit exactly", func(t *testing.T) {
		// given: tiles of one byte, estimated at one byte, so the estimate and the real size are the same
		area := plan.AreaOfInterest(builddomain.NewSliceTuningBuilder().Build())
		var tiles int64
		for _, r := range area.TileRange(16) {
			tiles += int64((r.MaxX - r.MinX + 1) * (r.MaxY - r.MinY + 1))
		}
		window := reliefInfo.Window(area, area)[0]
		samples := int64(window.Rows * window.Cols)

		exact := builddomain.NewSliceTuningBuilder().WithEstimatedTileBytes(1).
			WithMaxSizeBytes(tiles + samples*domain.BytesPerElevationSample).Build()
		m := newSliceMocksWithTuning(t, exact)
		givenSources(m)
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		m.baseMapReader.EXPECT().ReadTiles(baseMap.Path, 16, gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) {
				read := domain.TileRead{Format: "png"}
				for _, id := range ids {
					read.Tiles = append(read.Tiles, domain.Tile{ID: id, Data: []byte{1}})
				}
				return read, nil
			})
		m.elevationReader.EXPECT().ReadWindow(relief.Path, gomock.Any()).
			DoAndReturn(func(_ string, w domain.GridWindow) (domain.ElevationWindow, error) { return samplesFor(w), nil })

		// when
		slice, err := m.service.Generate(plan)

		// then
		require.NoError(t, err)
		assert.Equal(t, exact.MaxSizeBytes, slice.Summary.SizeBytes)
	})

	t.Run("should name the base map whose content cannot be read", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		givenSources(m)
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		m.baseMapReader.EXPECT().ReadTiles(baseMap.Path, gomock.Any(), gomock.Any()).
			Return(domain.TileRead{}, fmt.Errorf("%w: cut off", domain.ErrGeoDataContentUnreadable))

		// when
		_, err := m.service.Generate(plan)

		// then
		require.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
		assert.ErrorContains(t, err, `"map"`)
	})

	t.Run("should name the base map whose levels cannot be read", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		m.allFilesExist()
		m.repository.EXPECT().List().Return([]domain.GeoDataSource{baseMap, relief}, nil)
		m.baseMapReader.EXPECT().Levels(baseMap.Path).Return(domain.LevelRange{}, fmt.Errorf("%w: no tiles", domain.ErrGeoDataContentUnreadable))

		// when
		_, err := m.service.Generate(plan)

		// then
		require.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
		assert.ErrorContains(t, err, `"map"`)
	})

	t.Run("should name the elevation source whose unit is not supported", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		givenSources(m)
		m.elevationReader.EXPECT().Describe(relief.Path).Return(domain.ElevationGridInfo{}, fmt.Errorf("%w: unit 9999", domain.ErrElevationUnitUnsupported))

		// when
		_, err := m.service.Generate(plan)

		// then
		require.ErrorIs(t, err, domain.ErrElevationUnitUnsupported)
		assert.ErrorContains(t, err, `"dem"`)
	})

	t.Run("should name the elevation source whose samples cannot be read, giving no partial slice", func(t *testing.T) {
		// given
		m := newSliceMocks(t)
		givenSources(m)
		m.baseMapReader.EXPECT().ReadTiles(baseMap.Path, 16, gomock.Any()).
			DoAndReturn(func(_ string, _ int, ids []domain.TileID) (domain.TileRead, error) { return tilesFor(ids), nil })
		m.elevationReader.EXPECT().Describe(relief.Path).Return(reliefInfo, nil)
		m.elevationReader.EXPECT().ReadWindow(relief.Path, gomock.Any()).
			Return(domain.ElevationWindow{}, fmt.Errorf("%w: cut off", domain.ErrGeoDataContentUnreadable))

		// when
		slice, err := m.service.Generate(plan)

		// then
		require.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
		assert.ErrorContains(t, err, `"dem"`)
		assert.Equal(t, domain.GeoSlice{}, slice)
	})
}
