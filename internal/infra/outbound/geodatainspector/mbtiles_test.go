package geodatainspector_test

import (
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
		inspector := geodatainspector.NewGeoDataInspector()

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
		inspector := geodatainspector.NewGeoDataInspector()

		// when
		_, err := inspector.Inspect(path)

		// then
		assert.ErrorIs(t, err, domain.ErrUnsupportedDataFormat)
	})
}
