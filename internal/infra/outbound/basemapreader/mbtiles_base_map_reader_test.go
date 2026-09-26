package basemapreader_test

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/basemapreader"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

func writeMBTiles(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "map.mbtiles")
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func tile(z, x, y int) helper.MBTile {
	return helper.MBTile{Z: z, X: x, Y: y, Data: helper.TileData(z, x, y)}
}

func Test_MBTiles_Levels(t *testing.T) {
	t.Run("should read the minimum and maximum zoom from the metadata", func(t *testing.T) {
		// given
		spec := helper.MBTilesSpec{
			Bounds:  [4]float64{-47, -24, -46, -23},
			MinZoom: new(3), MaxZoom: new(14),
			Tiles: []helper.MBTile{tile(8, 100, 100)},
		}
		path := writeMBTiles(t, helper.MBTilesWithTiles(spec))

		// when
		levels, err := basemapreader.NewMBTiles().Levels(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.LevelRange{Min: 3, Max: 14}, levels)
	})

	t.Run("should take the levels from the tiles when the metadata does not have them", func(t *testing.T) {
		// given
		spec := helper.MBTilesSpec{
			Bounds: [4]float64{-47, -24, -46, -23},
			Tiles:  []helper.MBTile{tile(5, 10, 10), tile(9, 100, 100), tile(7, 30, 30)},
		}
		path := writeMBTiles(t, helper.MBTilesWithTiles(spec))

		// when
		levels, err := basemapreader.NewMBTiles().Levels(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.LevelRange{Min: 5, Max: 9}, levels)
	})

	t.Run("should fail with an unreadable-content error for a file whose tiles cannot be read", func(t *testing.T) {
		// given
		path := writeMBTiles(t, helper.CorruptMBTiles())

		// when
		_, err := basemapreader.NewMBTiles().Levels(path)

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})
}

func Test_MBTiles_ReadTiles(t *testing.T) {
	spec := helper.MBTilesSpec{
		Bounds: [4]float64{-47, -24, -46, -23},
		Tiles:  []helper.MBTile{tile(12, 1516, 2323), tile(12, 1517, 2323), tile(12, 1516, 2324), tile(11, 758, 1161)},
	}

	t.Run("should return the exact bytes of the requested tiles, identified in XYZ", func(t *testing.T) {
		// given
		path := writeMBTiles(t, helper.MBTilesWithTiles(spec))
		ids := []domain.TileID{{Level: 12, X: 1516, Y: 2323}, {Level: 12, X: 1516, Y: 2324}}

		// when
		read, err := basemapreader.NewMBTiles().ReadTiles(path, 12, ids)

		// then
		require.NoError(t, err)
		assert.Empty(t, read.Missing)
		require.Len(t, read.Tiles, 2)
		assert.Equal(t, ids[0], read.Tiles[0].ID)
		assert.Equal(t, helper.TileData(12, 1516, 2323), read.Tiles[0].Data)
		assert.Equal(t, ids[1], read.Tiles[1].ID)
		assert.Equal(t, helper.TileData(12, 1516, 2324), read.Tiles[1].Data)
	})

	t.Run("should report a tile the file does not contain as missing, without failing", func(t *testing.T) {
		// given
		path := writeMBTiles(t, helper.MBTilesWithTiles(spec))
		ids := []domain.TileID{{Level: 12, X: 1516, Y: 2323}, {Level: 12, X: 9999, Y: 1}, {Level: 12, X: 1517, Y: 2323}}

		// when
		read, err := basemapreader.NewMBTiles().ReadTiles(path, 12, ids)

		// then
		require.NoError(t, err)
		assert.Len(t, read.Tiles, 2)
		assert.Equal(t, []domain.TileID{{Level: 12, X: 9999, Y: 1}}, read.Missing)
	})

	t.Run("should report every tile as missing when the file has none of them", func(t *testing.T) {
		// given
		path := writeMBTiles(t, helper.MBTilesWithTiles(spec))
		ids := []domain.TileID{{Level: 12, X: 1, Y: 1}, {Level: 12, X: 2, Y: 2}}

		// when
		read, err := basemapreader.NewMBTiles().ReadTiles(path, 12, ids)

		// then
		require.NoError(t, err)
		assert.Empty(t, read.Tiles)
		assert.Equal(t, ids, read.Missing)
	})

	t.Run("should only look at the requested level", func(t *testing.T) {
		// given
		path := writeMBTiles(t, helper.MBTilesWithTiles(spec))

		// when
		read, err := basemapreader.NewMBTiles().ReadTiles(path, 11, []domain.TileID{{Level: 11, X: 758, Y: 1161}, {Level: 11, X: 1516, Y: 2323}})

		// then
		require.NoError(t, err)
		assert.Len(t, read.Tiles, 1)
		assert.Len(t, read.Missing, 1)
	})

	t.Run("should report the image format of the metadata, png by default", func(t *testing.T) {
		// given
		jpeg := spec
		jpeg.Format = "jpg"
		jpegPath := writeMBTiles(t, helper.MBTilesWithTiles(jpeg))
		pngPath := writeMBTiles(t, helper.MBTilesWithTiles(spec))
		ids := []domain.TileID{{Level: 12, X: 1516, Y: 2323}}

		// when
		jpegRead, jpegErr := basemapreader.NewMBTiles().ReadTiles(jpegPath, 12, ids)
		pngRead, pngErr := basemapreader.NewMBTiles().ReadTiles(pngPath, 12, ids)

		// then
		require.NoError(t, jpegErr)
		require.NoError(t, pngErr)
		assert.Equal(t, "jpg", jpegRead.Format)
		assert.Equal(t, "png", pngRead.Format)
	})

	t.Run("should not modify the file", func(t *testing.T) {
		// given
		path := writeMBTiles(t, helper.MBTilesWithTiles(spec))
		before, err := os.ReadFile(path)
		require.NoError(t, err)

		// when
		_, readErr := basemapreader.NewMBTiles().ReadTiles(path, 12, []domain.TileID{{Level: 12, X: 1516, Y: 2323}})
		_, levelsErr := basemapreader.NewMBTiles().Levels(path)

		// then
		require.NoError(t, readErr)
		require.NoError(t, levelsErr)
		after, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, sha256.Sum256(before), sha256.Sum256(after))
		entries, _ := os.ReadDir(filepath.Dir(path))
		assert.Len(t, entries, 1, "no journal or other file left next to the map")
	})

	t.Run("should fail with an unreadable-content error for a file whose tiles cannot be read", func(t *testing.T) {
		// given
		path := writeMBTiles(t, helper.CorruptMBTiles())

		// when
		_, err := basemapreader.NewMBTiles().ReadTiles(path, 12, []domain.TileID{{Level: 12, X: 1, Y: 1}})

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})

	t.Run("should fail with an unreadable-content error for a file that is not a database", func(t *testing.T) {
		// given
		path := writeMBTiles(t, helper.NotSQLiteContent())

		// when
		_, err := basemapreader.NewMBTiles().ReadTiles(path, 12, []domain.TileID{{Level: 12, X: 1, Y: 1}})

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})

	t.Run("should fail with an unreadable-content error for a file that has gone", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "gone.mbtiles")

		// when
		_, err := basemapreader.NewMBTiles().ReadTiles(path, 12, []domain.TileID{{Level: 12, X: 1, Y: 1}})

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})
}
