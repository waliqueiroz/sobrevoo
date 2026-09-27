package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_NewGeoSlice_Summary(t *testing.T) {
	t.Run("should count the tiles present and missing over every tile set", func(t *testing.T) {
		// given
		first := builddomain.NewTileSetBuilder().WithMissing(domain.TileID{Level: 16, X: 1, Y: 1}).Build()
		second := builddomain.NewTileSetBuilder().
			WithSource(builddomain.NewGeoDataSourceBuilder().WithName("other-map").Build()).
			WithTiles(domain.Tile{ID: domain.TileID{Level: 12, X: 5, Y: 5}, Data: []byte{1}}).
			WithMissing(domain.TileID{Level: 12, X: 6, Y: 5}, domain.TileID{Level: 12, X: 7, Y: 5}).
			Build()

		// when
		slice := builddomain.NewGeoSliceBuilder().WithTileSets(first, second).Build()

		// then
		assert.Equal(t, 4, slice.Summary.TileCount)
		assert.Equal(t, 3, slice.Summary.MissingTileCount)
	})

	t.Run("should count the samples and those without value", func(t *testing.T) {
		// given
		first := builddomain.NewElevationGridBuilder().WithNoValueAt(0, 0).WithNoValueAt(1, 1).Build()
		second := builddomain.NewElevationGridBuilder().
			WithWindow(domain.GridWindow{Rows: 1, Cols: 2}).WithValues(5, 6).Build()

		// when
		slice := builddomain.NewGeoSliceBuilder().WithElevation(first, second).Build()

		// then
		assert.Equal(t, 11, slice.Summary.SampleCount)
		assert.Equal(t, 2, slice.Summary.NoValueSampleCount)
	})

	t.Run("should give the elevation range over the samples that have a value only", func(t *testing.T) {
		// given
		first := builddomain.NewElevationGridBuilder().WithValues(50, -3, 700, 12, 0, 99, 1, 2, 3).WithNoValueAt(0, 2).Build()
		second := builddomain.NewElevationGridBuilder().WithValues(10, 20, 30, 40, 50, 60, 70, 80, 300).Build()

		// when
		slice := builddomain.NewGeoSliceBuilder().WithElevation(first, second).Build()

		// then
		assert.True(t, slice.Summary.HasElevationRange)
		assert.Equal(t, -3.0, slice.Summary.MinElevation)
		assert.Equal(t, 300.0, slice.Summary.MaxElevation)
	})

	t.Run("should report no elevation range when no sample has a value", func(t *testing.T) {
		// given
		builder := builddomain.NewElevationGridBuilder()
		for row := 0; row < 3; row++ {
			for col := 0; col < 3; col++ {
				builder.WithNoValueAt(row, col)
			}
		}

		// when
		slice := builddomain.NewGeoSliceBuilder().WithElevation(builder.Build()).Build()

		// then
		assert.False(t, slice.Summary.HasElevationRange)
		assert.Equal(t, 9, slice.Summary.NoValueSampleCount)
	})

	t.Run("should report no elevation range for a slice without samples", func(t *testing.T) {
		// when
		slice := builddomain.NewGeoSliceBuilder().WithElevation().Build()

		// then
		assert.False(t, slice.Summary.HasElevationRange)
		assert.Zero(t, slice.Summary.SampleCount)
	})

	t.Run("should size the slice as the bytes of the tiles plus four bytes for each sample", func(t *testing.T) {
		// given: the default tile set has tiles of 3, 4 and 2 bytes; the default grid 9 samples
		slice := builddomain.NewGeoSliceBuilder().Build()

		// then
		assert.Equal(t, int64(3+4+2+4*9), slice.Summary.SizeBytes)
	})

	t.Run("should list each source used once, sorted by name, with the level of detail of the base maps", func(t *testing.T) {
		// given
		zeta := builddomain.NewGeoDataSourceBuilder().WithName("zeta-map").Build()
		alpha := builddomain.NewGeoDataSourceBuilder().WithName("alpha-map").Build()
		relief := builddomain.NewGeoDataSourceBuilder().WithName("dem").WithType(domain.DataTypeElevation).Build()
		sets := []domain.TileSet{
			builddomain.NewTileSetBuilder().WithSource(zeta).WithDetail(domain.DetailLevel{Chosen: 14}).Build(),
			builddomain.NewTileSetBuilder().WithSource(alpha).WithDetail(domain.DetailLevel{Chosen: 16}).Build(),
		}
		grids := []domain.ElevationGrid{
			builddomain.NewElevationGridBuilder().WithSource(relief).Build(),
			builddomain.NewElevationGridBuilder().WithSource(relief).Build(),
		}

		// when
		slice := builddomain.NewGeoSliceBuilder().WithTileSets(sets...).WithElevation(grids...).Build()

		// then
		require.Len(t, slice.Summary.Sources, 3)
		assert.Equal(t, "alpha-map", slice.Summary.Sources[0].Source.Name)
		require.NotNil(t, slice.Summary.Sources[0].Detail)
		assert.Equal(t, 16, slice.Summary.Sources[0].Detail.Chosen)
		assert.Equal(t, "dem", slice.Summary.Sources[1].Source.Name)
		assert.Nil(t, slice.Summary.Sources[1].Detail)
		assert.Equal(t, "zeta-map", slice.Summary.Sources[2].Source.Name)
	})

	t.Run("should keep the area", func(t *testing.T) {
		// given
		area := domain.BoundingBox{MinLatitude: 1, MaxLatitude: 2, MinLongitude: 3, MaxLongitude: 4}

		// when
		slice := builddomain.NewGeoSliceBuilder().WithArea(area).Build()

		// then
		assert.Equal(t, area, slice.Summary.Area)
		assert.Equal(t, area, slice.Area)
	})

	t.Run("should be a valid slice with no tile present at all", func(t *testing.T) {
		// given
		tileSet := builddomain.NewTileSetBuilder().WithTiles().WithMissing(domain.TileID{Level: 16, X: 1, Y: 1}).Build()

		// when
		slice := builddomain.NewGeoSliceBuilder().WithTileSets(tileSet).Build()

		// then
		assert.Zero(t, slice.Summary.TileCount)
		assert.Equal(t, 1, slice.Summary.MissingTileCount)
	})
}

func Test_NewGeoSlice_Order(t *testing.T) {
	t.Run("should put the tile sets in order of base map name, then level, and the tiles by level, column and row", func(t *testing.T) {
		// given
		id := func(z, x, y int) domain.TileID { return domain.TileID{Level: z, X: x, Y: y} }
		zeta := builddomain.NewTileSetBuilder().
			WithSource(builddomain.NewGeoDataSourceBuilder().WithName("zeta").Build()).
			WithTiles(domain.Tile{ID: id(12, 5, 5)}, domain.Tile{ID: id(12, 3, 9)}, domain.Tile{ID: id(12, 3, 2)}).
			WithMissing(id(12, 9, 1), id(12, 2, 8)).Build()
		alpha := builddomain.NewTileSetBuilder().
			WithSource(builddomain.NewGeoDataSourceBuilder().WithName("alpha").Build()).
			WithTiles(domain.Tile{ID: id(10, 1, 1)}).Build()

		// when
		slice := builddomain.NewGeoSliceBuilder().WithTileSets(zeta, alpha).Build()

		// then
		require.Len(t, slice.TileSets, 2)
		assert.Equal(t, "alpha", slice.TileSets[0].Source.Name)
		assert.Equal(t, "zeta", slice.TileSets[1].Source.Name)
		assert.Equal(t, []domain.TileID{id(12, 3, 2), id(12, 3, 9), id(12, 5, 5)}, tileIDs(slice.TileSets[1].Tiles))
		assert.Equal(t, []domain.TileID{id(12, 2, 8), id(12, 9, 1)}, slice.TileSets[1].Missing)
	})

	t.Run("should not change the tile sets it was given", func(t *testing.T) {
		// given
		zeta := builddomain.NewTileSetBuilder().
			WithSource(builddomain.NewGeoDataSourceBuilder().WithName("zeta").Build()).
			WithTiles(domain.Tile{ID: domain.TileID{Level: 1, X: 5}}, domain.Tile{ID: domain.TileID{Level: 1, X: 2}}).Build()
		alpha := builddomain.NewTileSetBuilder().
			WithSource(builddomain.NewGeoDataSourceBuilder().WithName("alpha").Build()).Build()
		given := []domain.TileSet{zeta, alpha}

		// when
		builddomain.NewGeoSliceBuilder().WithTileSets(given...).Build()

		// then
		assert.Equal(t, "zeta", given[0].Source.Name)
		assert.Equal(t, 5, given[0].Tiles[0].ID.X)
	})

	t.Run("should keep the elevation grids in the order given", func(t *testing.T) {
		// given
		first := builddomain.NewElevationGridBuilder().WithSource(builddomain.NewGeoDataSourceBuilder().WithName("z").Build()).Build()
		second := builddomain.NewElevationGridBuilder().WithSource(builddomain.NewGeoDataSourceBuilder().WithName("a").Build()).Build()

		// when
		slice := builddomain.NewGeoSliceBuilder().WithElevation(first, second).Build()

		// then
		assert.Equal(t, "z", slice.Elevation[0].Source.Name)
		assert.Equal(t, "a", slice.Elevation[1].Source.Name)
	})
}

func tileIDs(tiles []domain.Tile) []domain.TileID {
	ids := make([]domain.TileID, len(tiles))
	for i, tile := range tiles {
		ids[i] = tile.ID
	}
	return ids
}

func Test_SliceTuning_Estimate(t *testing.T) {
	t.Run("should estimate the size as the tiles at the assumed size plus four bytes for each sample", func(t *testing.T) {
		// given
		tuning := builddomain.NewSliceTuningBuilder().WithEstimatedTileBytes(1000).Build()

		// when
		estimate := tuning.Estimate(30, 500)

		// then
		assert.Equal(t, int64(30*1000+500*4), estimate)
	})

	t.Run("should not overflow for very large counts", func(t *testing.T) {
		// given
		tuning := builddomain.NewSliceTuningBuilder().Build()

		// when
		estimate := tuning.Estimate(1<<40, 1<<40)

		// then
		assert.Greater(t, estimate, tuning.MaxSizeBytes)
	})
}

func Test_SliceTuning_EnsureFits(t *testing.T) {
	area := domain.BoundingBox{MinLatitude: -23.7, MaxLatitude: -23.4, MinLongitude: -46.8, MaxLongitude: -46.4}
	tuning := builddomain.NewSliceTuningBuilder().WithMaxSizeBytes(256 * 1024 * 1024).Build()

	t.Run("should accept a slice of exactly the limit", func(t *testing.T) {
		// when
		err := tuning.EnsureFits(256*1024*1024, area, 16)

		// then
		assert.NoError(t, err)
	})

	t.Run("should refuse a slice one byte over the limit", func(t *testing.T) {
		// when
		err := tuning.EnsureFits(256*1024*1024+1, area, 16)

		// then
		assert.ErrorIs(t, err, domain.ErrSliceTooLarge)
	})

	t.Run("should accept a small slice", func(t *testing.T) {
		// when
		err := tuning.EnsureFits(1024, area, 16)

		// then
		assert.NoError(t, err)
	})

	t.Run("should say the size, the limit, the level, the extent of the area and what to try", func(t *testing.T) {
		// when
		err := tuning.EnsureFits(300*1024*1024, area, 17)

		// then
		require.ErrorIs(t, err, domain.ErrSliceTooLarge)
		assert.ErrorContains(t, err, "300.0 MiB")
		assert.ErrorContains(t, err, "256.0 MiB")
		assert.ErrorContains(t, err, "level 17")
		assert.ErrorContains(t, err, "over an area of 40.8 km × 33.4 km")
		assert.ErrorContains(t, err, "try a higher --distance in the plan, or a shorter track")
	})

	t.Run("should write a size over a gibibyte in GiB", func(t *testing.T) {
		// when
		err := tuning.EnsureFits(3*1024*1024*1024, area, 16)

		// then
		assert.ErrorContains(t, err, "3.0 GiB")
	})
}

func planWith(tileCounts []int, levels []int, windows []domain.GridWindow) domain.SlicePlan {
	plan := domain.SlicePlan{Area: domain.BoundingBox{MinLatitude: -1, MaxLatitude: 1, MinLongitude: -1, MaxLongitude: 1}}
	for i, count := range tileCounts {
		plan.Tiles = append(plan.Tiles, domain.TileRequest{
			Detail: domain.DetailLevel{Chosen: levels[i]},
			IDs:    make([]domain.TileID, count),
		})
	}
	for _, window := range windows {
		plan.Samples = append(plan.Samples, domain.SampleRequest{Window: window})
	}
	return plan
}

func Test_SlicePlan(t *testing.T) {
	t.Run("should count the tiles asked of every base map", func(t *testing.T) {
		// given
		plan := planWith([]int{10, 5}, []int{14, 16}, nil)

		// when
		count := plan.TileCount()

		// then
		assert.Equal(t, int64(15), count)
	})

	t.Run("should count the samples of every window", func(t *testing.T) {
		// given
		plan := planWith(nil, nil, []domain.GridWindow{{Rows: 10, Cols: 20}, {Rows: 3, Cols: 3}})

		// when
		count := plan.SampleCount()

		// then
		assert.Equal(t, int64(209), count)
	})

	t.Run("should report the most detailed level chosen for any base map as the level of the slice", func(t *testing.T) {
		// given
		plan := planWith([]int{1, 1, 1}, []int{12, 16, 14}, nil)

		// when
		level := plan.Level()

		// then
		assert.Equal(t, 16, level)
	})

	t.Run("should report level zero for a plan without base maps", func(t *testing.T) {
		// when
		level := planWith(nil, nil, nil).Level()

		// then
		assert.Equal(t, 0, level)
	})
}

func Test_SliceTuning_EnsurePlanFits(t *testing.T) {
	tuning := builddomain.NewSliceTuningBuilder().WithEstimatedTileBytes(100).WithMaxSizeBytes(1000 + 4*50).Build()

	t.Run("should accept a plan whose estimate is exactly the limit", func(t *testing.T) {
		// given: 10 tiles at 100 bytes and 50 samples at 4 bytes
		plan := planWith([]int{10}, []int{16}, []domain.GridWindow{{Rows: 5, Cols: 10}})

		// when
		err := tuning.EnsurePlanFits(plan)

		// then
		assert.NoError(t, err)
	})

	t.Run("should refuse a plan whose estimate is one tile over the limit, saying its level", func(t *testing.T) {
		// given
		plan := planWith([]int{11}, []int{17}, []domain.GridWindow{{Rows: 5, Cols: 10}})

		// when
		err := tuning.EnsurePlanFits(plan)

		// then
		require.ErrorIs(t, err, domain.ErrSliceTooLarge)
		assert.ErrorContains(t, err, "level 17")
	})
}

func Test_SizeGuard(t *testing.T) {
	area := domain.BoundingBox{MinLatitude: -1, MaxLatitude: 1, MinLongitude: -1, MaxLongitude: 1}
	tuning := builddomain.NewSliceTuningBuilder().WithMaxSizeBytes(100).Build()
	plan := domain.SlicePlan{Area: area, Tiles: []domain.TileRequest{{Detail: domain.DetailLevel{Chosen: 16}}}}
	bytesTile := func(size int) domain.TileSet {
		return builddomain.NewTileSetBuilder().WithTiles(domain.Tile{Data: make([]byte, size)}).Build()
	}

	t.Run("should add up the real bytes of the tiles read", func(t *testing.T) {
		// given
		guard := tuning.NewSizeGuard(plan)

		// when
		first := guard.AddTileSet(bytesTile(60))
		second := guard.AddTileSet(bytesTile(40))

		// then
		assert.NoError(t, first)
		assert.NoError(t, second, "exactly the limit is accepted")
	})

	t.Run("should refuse the tile set that takes the total past the limit, saying the level", func(t *testing.T) {
		// given
		guard := tuning.NewSizeGuard(plan)
		require.NoError(t, guard.AddTileSet(bytesTile(60)))

		// when
		err := guard.AddTileSet(bytesTile(41))

		// then
		require.ErrorIs(t, err, domain.ErrSliceTooLarge)
		assert.ErrorContains(t, err, "level 16")
	})

	t.Run("should count four bytes for each sample of a grid", func(t *testing.T) {
		// given: a grid of 9 samples is 36 bytes
		guard := tuning.NewSizeGuard(plan)
		grid := builddomain.NewElevationGridBuilder().Build()

		// when
		first := guard.AddGrid(grid)
		second := guard.AddGrid(grid)
		third := guard.AddGrid(grid)

		// then: 36, 72, then 108 is past 100
		assert.NoError(t, first)
		assert.NoError(t, second)
		assert.ErrorIs(t, third, domain.ErrSliceTooLarge)
	})

	t.Run("should count tiles and samples together", func(t *testing.T) {
		// given
		guard := tuning.NewSizeGuard(plan)
		require.NoError(t, guard.AddTileSet(bytesTile(70)))

		// when
		err := guard.AddGrid(builddomain.NewElevationGridBuilder().Build())

		// then: 70 + 36 = 106
		assert.ErrorIs(t, err, domain.ErrSliceTooLarge)
	})
}

func Test_NewGeoSlice_Identification(t *testing.T) {
	t.Run("should leave the plan and the content identification empty: they come from elsewhere", func(t *testing.T) {
		// given / when
		slice := builddomain.NewGeoSliceBuilder().Build()

		// then
		assert.Empty(t, slice.PlanID)
		assert.Empty(t, slice.ContentID)
	})
}

// planWithMarkerAt is a plan of one frame whose marker and camera are around the
// given point, 1 km from it.
func planWithMarkerAt(lat, lon float64) domain.CameraPlan {
	frame := builddomain.NewCameraFrameBuilder().
		WithMarkerPosition(lat, lon).WithCameraPosition(lat-0.004, lon).WithCameraToMarkerDistance(1000).Build()
	return builddomain.NewCameraPlanBuilder().WithFrames(frame).Build()
}

func Test_GeoSlice_EnsureMatches(t *testing.T) {
	plan := planWithMarkerAt(-23.55, -46.63)

	t.Run("should accept a slice made from the plan", func(t *testing.T) {
		// given
		slice := builddomain.NewGeoSliceBuilder().WithPlanID(plan.ID()).Build()

		// when / then
		assert.NoError(t, slice.EnsureMatches(plan))
	})

	t.Run("should refuse a slice made from another plan, naming both by their first 12 characters", func(t *testing.T) {
		// given
		other := planWithMarkerAt(-23.56, -46.63)
		slice := builddomain.NewGeoSliceBuilder().WithPlanID(other.ID()).Build()

		// when
		err := slice.EnsureMatches(plan)

		// then
		assert.ErrorIs(t, err, domain.ErrSliceDoesNotMatchPlan)
		assert.ErrorContains(t, err, other.ID()[:12])
		assert.ErrorContains(t, err, plan.ID()[:12])
		assert.NotContains(t, err.Error(), plan.ID()[:13])
	})

	t.Run("should refuse a slice that says nothing of its plan", func(t *testing.T) {
		// given
		slice := builddomain.NewGeoSliceBuilder().Build()

		// when
		err := slice.EnsureMatches(plan)

		// then
		assert.ErrorIs(t, err, domain.ErrSliceDoesNotMatchPlan)
	})
}

func Test_GeoSlice_EnsureCovers(t *testing.T) {
	tuning := builddomain.NewSliceTuningBuilder().Build()
	plan := planWithMarkerAt(-23.55, -46.63)
	needed := plan.AreaOfInterest(tuning)

	t.Run("should accept a slice whose area is the one the plan needs, or bigger", func(t *testing.T) {
		// given
		bigger := needed
		bigger.MinLatitude -= 0.1
		bigger.MaxLongitude += 0.1

		// when
		exact := builddomain.NewGeoSliceBuilder().WithArea(needed).Build().EnsureCovers(plan, tuning)
		wider := builddomain.NewGeoSliceBuilder().WithArea(bigger).Build().EnsureCovers(plan, tuning)

		// then
		assert.NoError(t, exact)
		assert.NoError(t, wider)
	})

	t.Run("should refuse a slice smaller than what the plan needs, on any side", func(t *testing.T) {
		for _, side := range []func(a *domain.BoundingBox){
			func(a *domain.BoundingBox) { a.MinLatitude += 0.001 },
			func(a *domain.BoundingBox) { a.MaxLatitude -= 0.001 },
			func(a *domain.BoundingBox) { a.MinLongitude += 0.001 },
			func(a *domain.BoundingBox) { a.MaxLongitude -= 0.001 },
		} {
			// given
			area := needed
			side(&area)

			// when
			err := builddomain.NewGeoSliceBuilder().WithArea(area).Build().EnsureCovers(plan, tuning)

			// then
			assert.ErrorIs(t, err, domain.ErrSliceDoesNotCoverPlan)
		}
	})

	t.Run("should say the area the plan needs and the one the slice has, as the slice summary writes an area", func(t *testing.T) {
		// given
		small := domain.BoundingBox{MinLatitude: -23.5600, MaxLatitude: -23.5500, MinLongitude: -46.6400, MaxLongitude: -46.6200}

		// when
		err := builddomain.NewGeoSliceBuilder().WithArea(small).Build().EnsureCovers(plan, tuning)

		// then
		assert.ErrorContains(t, err, "the plan needs lat -23.5")
		assert.ErrorContains(t, err, "the slice has lat -23.5600 to -23.5500, lon -46.6400 to -46.6200")
	})

	t.Run("should cover an area that crosses the antimeridian only with one that does", func(t *testing.T) {
		// given
		crossing := planWithMarkerAt(-16.5, 179.999)
		crossingNeed := crossing.AreaOfInterest(tuning)
		require.True(t, crossingNeed.CrossesAntimeridian)
		notCrossing := crossingNeed
		notCrossing.CrossesAntimeridian = false

		// when
		ok := builddomain.NewGeoSliceBuilder().WithArea(crossingNeed).Build().EnsureCovers(crossing, tuning)
		bad := builddomain.NewGeoSliceBuilder().WithArea(notCrossing).Build().EnsureCovers(crossing, tuning)

		// then
		assert.NoError(t, ok)
		assert.ErrorIs(t, bad, domain.ErrSliceDoesNotCoverPlan)
	})

	t.Run("should mark in the message an area that crosses the antimeridian", func(t *testing.T) {
		// given
		crossing := planWithMarkerAt(-16.5, 179.999)
		small := domain.BoundingBox{MinLatitude: -16.51, MaxLatitude: -16.49, MinLongitude: 179.9995, MaxLongitude: -179.9995, CrossesAntimeridian: true}

		// when
		err := builddomain.NewGeoSliceBuilder().WithArea(small).Build().EnsureCovers(crossing, tuning)

		// then
		assert.ErrorContains(t, err, "(crosses the antimeridian)")
	})
}

func Test_GeoSlice_EnsureDrawable(t *testing.T) {
	tileSetOf := func(name, format string) domain.TileSet {
		return builddomain.NewTileSetBuilder().WithSource(builddomain.NewGeoDataSourceBuilder().WithName(name).Build()).WithFormat(format).Build()
	}

	t.Run("should accept tiles that are images", func(t *testing.T) {
		for _, format := range []string{"png", "jpg", "webp"} {
			// given
			slice := builddomain.NewGeoSliceBuilder().WithTileSets(tileSetOf("map", format)).Build()

			// when / then
			assert.NoError(t, slice.EnsureDrawable(), format)
		}
	})

	t.Run("should refuse vector tiles, naming the base map and the format, and saying it is not supported yet", func(t *testing.T) {
		// given
		slice := builddomain.NewGeoSliceBuilder().WithTileSets(tileSetOf("bbbike", "pbf")).Build()

		// when
		err := slice.EnsureDrawable()

		// then
		assert.ErrorIs(t, err, domain.ErrTileFormatUnsupported)
		assert.ErrorContains(t, err, `base map "bbbike" has vector tiles (pbf); drawing vector tiles is not supported yet, use a base map of image tiles (PNG, JPG or WebP)`)
	})

	t.Run("should refuse the vector tiles of the other name they go by", func(t *testing.T) {
		// given
		slice := builddomain.NewGeoSliceBuilder().WithTileSets(tileSetOf("map", "mvt")).Build()

		// when / then
		assert.ErrorContains(t, slice.EnsureDrawable(), "has vector tiles (mvt)")
	})

	t.Run("should refuse any other format, naming it", func(t *testing.T) {
		// given
		gif := builddomain.NewGeoSliceBuilder().WithTileSets(tileSetOf("map", "gif")).Build()
		none := builddomain.NewGeoSliceBuilder().WithTileSets(tileSetOf("map", "")).Build()

		// when
		errGIF := gif.EnsureDrawable()
		errNone := none.EnsureDrawable()

		// then
		assert.ErrorIs(t, errGIF, domain.ErrTileFormatUnsupported)
		assert.ErrorContains(t, errGIF, `has tiles of format "gif", which is not supported for drawing`)
		assert.ErrorIs(t, errNone, domain.ErrTileFormatUnsupported)
	})

	t.Run("should refuse a slice in which one base map has vector tiles even when another has images", func(t *testing.T) {
		// given
		slice := builddomain.NewGeoSliceBuilder().WithTileSets(tileSetOf("a-images", "png"), tileSetOf("b-vectors", "pbf")).Build()

		// when
		err := slice.EnsureDrawable()

		// then
		assert.ErrorIs(t, err, domain.ErrTileFormatUnsupported)
		assert.ErrorContains(t, err, `"b-vectors"`)
	})

	t.Run("should accept a slice with no tiles at all: it is drawn with the marks of a missing map", func(t *testing.T) {
		// given
		slice := builddomain.NewGeoSliceBuilder().WithTileSets().Build()

		// when / then
		assert.NoError(t, slice.EnsureDrawable())
	})

	t.Run("should refuse a slice in which no elevation sample has a value", func(t *testing.T) {
		// given
		nan := float32(math.NaN())
		grid := builddomain.NewElevationGridBuilder().WithValues(nan, nan, nan, nan, nan, nan, nan, nan, nan).Build()
		slice := builddomain.NewGeoSliceBuilder().WithElevation(grid).Build()

		// when
		err := slice.EnsureDrawable()

		// then
		assert.ErrorIs(t, err, domain.ErrNoElevationData)
		assert.ErrorContains(t, err, "no elevation sample has a value")
	})

	t.Run("should refuse a slice with no elevation grid at all", func(t *testing.T) {
		// given
		slice := builddomain.NewGeoSliceBuilder().WithElevation().Build()

		// when / then
		assert.ErrorIs(t, slice.EnsureDrawable(), domain.ErrNoElevationData)
	})

	t.Run("should accept a slice in which one sample has a value", func(t *testing.T) {
		// given
		nan := float32(math.NaN())
		grid := builddomain.NewElevationGridBuilder().WithValues(nan, nan, nan, nan, 10, nan, nan, nan, nan).Build()
		slice := builddomain.NewGeoSliceBuilder().WithElevation(grid).Build()

		// when / then
		assert.NoError(t, slice.EnsureDrawable())
	})

	t.Run("should check the format of the tiles before the elevation", func(t *testing.T) {
		// given
		nan := float32(math.NaN())
		grid := builddomain.NewElevationGridBuilder().WithValues(nan, nan, nan, nan, nan, nan, nan, nan, nan).Build()
		slice := builddomain.NewGeoSliceBuilder().WithElevation(grid).WithTileSets(tileSetOf("map", "pbf")).Build()

		// when
		err := slice.EnsureDrawable()

		// then
		assert.ErrorIs(t, err, domain.ErrTileFormatUnsupported)
	})
}
