package geodatainspector_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/geodatainspector"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

func writeFixture(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func Test_Inspector_Inspect_MBTiles(t *testing.T) {
	t.Run("should identify a valid MBTiles file as a base map with the bounds from its metadata table", func(t *testing.T) {
		// given
		path := writeFixture(t, "valid.mbtiles", helper.ValidMBTiles())
		inspector := geodatainspector.New()

		// when
		result, err := inspector.Inspect(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.DataFormatMBTiles, result.Format)
		assert.Equal(t, domain.DataTypeBaseMap, result.Type)
		assert.InDelta(t, 40.0, result.BoundingBox.MinLatitude, 0.0001)
		assert.InDelta(t, 50.0, result.BoundingBox.MaxLatitude, 0.0001)
		assert.InDelta(t, 10.0, result.BoundingBox.MinLongitude, 0.0001)
		assert.InDelta(t, 20.0, result.BoundingBox.MaxLongitude, 0.0001)
	})

	t.Run("should reject a SQLite file without a bounds entry as an unsupported format", func(t *testing.T) {
		// given
		path := writeFixture(t, "no-bounds.mbtiles", helper.MBTilesWithoutBounds())
		inspector := geodatainspector.New()

		// when
		_, err := inspector.Inspect(path)

		// then
		assert.ErrorIs(t, err, domain.ErrUnsupportedDataFormat)
	})

	t.Run("should correct declared bounds that are larger than where the tiles are", func(t *testing.T) {
		// given: the north-east corner of the bounds was written as 0,0, as some tilemaker versions do
		tiles := []helper.MBTile{}
		for x := 379; x <= 380; x++ {
			for y := 580; y <= 581; y++ {
				tiles = append(tiles, helper.MBTile{Z: 10, X: x, Y: y, Data: []byte{1}})
			}
		}
		spec := helper.MBTilesSpec{Bounds: [4]float64{-47, -24, 0, 0}, Tiles: tiles}
		path := writeFixture(t, "broken-bounds.mbtiles", helper.MBTilesWithTiles(spec))
		inspector := geodatainspector.New()

		result, err := inspector.Inspect(path)

		require.NoError(t, err)
		expected := domain.TileRange{Level: 10, MinX: 379, MaxX: 380, MinY: 580, MaxY: 581}.Bounds()
		assert.InDelta(t, expected.MaxLatitude, result.BoundingBox.MaxLatitude, 1e-9)
		assert.InDelta(t, expected.MaxLongitude, result.BoundingBox.MaxLongitude, 1e-9)
		assert.InDelta(t, math.Max(-24, expected.MinLatitude), result.BoundingBox.MinLatitude, 1e-9)
		assert.InDelta(t, math.Max(-47, expected.MinLongitude), result.BoundingBox.MinLongitude, 1e-9)
		assert.Less(t, result.BoundingBox.MaxLatitude, 0.0, "no longer reaches the 0,0 corner the file declared")
	})

	t.Run("should keep declared bounds that are tighter than the tiles", func(t *testing.T) {
		// given
		spec := helper.MBTilesSpec{
			Bounds: [4]float64{-46.7, -23.6, -46.6, -23.5},
			Tiles:  []helper.MBTile{{Z: 10, X: 379, Y: 580, Data: []byte{1}}, {Z: 10, X: 380, Y: 581, Data: []byte{1}}},
		}
		path := writeFixture(t, "tight.mbtiles", helper.MBTilesWithTiles(spec))
		inspector := geodatainspector.New()

		result, err := inspector.Inspect(path)

		require.NoError(t, err)
		assert.InDelta(t, -23.6, result.BoundingBox.MinLatitude, 1e-9)
		assert.InDelta(t, -23.5, result.BoundingBox.MaxLatitude, 1e-9)
		assert.InDelta(t, -46.7, result.BoundingBox.MinLongitude, 1e-9)
		assert.InDelta(t, -46.6, result.BoundingBox.MaxLongitude, 1e-9)
	})

	t.Run("should use the declared bounds as they are when the file has no tiles to check them against", func(t *testing.T) {
		// given
		path := writeFixture(t, "no-tiles.mbtiles", helper.ValidMBTiles())
		inspector := geodatainspector.New()

		result, err := inspector.Inspect(path)

		require.NoError(t, err)
		assert.InDelta(t, 50.0, result.BoundingBox.MaxLatitude, 1e-9)
		assert.InDelta(t, 20.0, result.BoundingBox.MaxLongitude, 1e-9)
	})
}
