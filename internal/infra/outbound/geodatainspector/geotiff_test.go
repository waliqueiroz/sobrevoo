package geodatainspector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/geodatainspector"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

func Test_Inspector_Inspect_GeoTIFF(t *testing.T) {
	t.Run("should identify a valid geographic-CRS GeoTIFF as elevation with the area computed from its georeferencing tags", func(t *testing.T) {
		// given
		path := writeFixture(t, "valid.tif", helper.ValidGeoTIFF())
		inspector := geodatainspector.NewGeoDataInspector()

		// when
		result, err := inspector.Inspect(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.DataFormatGeoTIFF, result.Format)
		assert.Equal(t, domain.DataTypeElevation, result.Type)
		assert.InDelta(t, 50.0, result.BoundingBox.MinLatitude, 0.0001)
		assert.InDelta(t, 51.0, result.BoundingBox.MaxLatitude, 0.0001)
		assert.InDelta(t, 10.0, result.BoundingBox.MinLongitude, 0.0001)
		assert.InDelta(t, 11.0, result.BoundingBox.MaxLongitude, 0.0001)
	})

	t.Run("should compute a wrapped, antimeridian-crossing area for a raster placed near the 180th meridian", func(t *testing.T) {
		// given
		path := writeFixture(t, "antimeridian.tif", helper.GeoTIFFCrossingAntimeridian())
		inspector := geodatainspector.NewGeoDataInspector()

		// when
		result, err := inspector.Inspect(path)

		// then
		require.NoError(t, err)
		assert.True(t, result.BoundingBox.CrossesAntimeridian)
		assert.InDelta(t, 175.0, result.BoundingBox.MinLongitude, 0.0001)
		assert.InDelta(t, -175.0, result.BoundingBox.MaxLongitude, 0.0001)
	})

	t.Run("should reject a GeoTIFF in a projected CRS as an unsupported format", func(t *testing.T) {
		// given
		path := writeFixture(t, "projected.tif", helper.GeoTIFFWithProjectedCRS())
		inspector := geodatainspector.NewGeoDataInspector()

		// when
		_, err := inspector.Inspect(path)

		// then
		assert.ErrorIs(t, err, domain.ErrUnsupportedDataFormat)
	})
}
