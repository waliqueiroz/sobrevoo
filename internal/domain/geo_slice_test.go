package domain_test

import (
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
