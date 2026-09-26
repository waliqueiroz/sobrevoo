package elevationreader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/elevationreader"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

const (
	gridWidth  = 10
	gridHeight = 8
)

// gridValues fills a gridWidth × gridHeight grid with row × step + column, so
// every sample says where it is.
func gridValues(step float64) [][]float64 {
	values := make([][]float64, gridHeight)
	for r := range values {
		values[r] = make([]float64, gridWidth)
		for c := range values[r] {
			values[r][c] = float64(r)*step + float64(c)
		}
	}
	return values
}

// baseSpec is a grid of int16 samples, in strips, anchored at (-47, -23) with
// cells of 0.01°.
func baseSpec() helper.GeoTIFFSpec {
	return helper.GeoTIFFSpec{
		Width: gridWidth, Height: gridHeight,
		OriginLon: -47, OriginLat: -23,
		ScaleX: 0.01, ScaleY: 0.01,
		SampleType: helper.Int16,
		Values:     gridValues(100),
	}
}

func writeTIFF(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "relief.tif")
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func describe(t *testing.T, spec helper.GeoTIFFSpec) (domain.ElevationGridInfo, error) {
	t.Helper()
	return elevationreader.NewGeoTIFF().Describe(writeTIFF(t, helper.GeoTIFFWithSamples(spec)))
}

func Test_GeoTIFF_Describe(t *testing.T) {
	t.Run("should describe the grid from the georeferencing tags", func(t *testing.T) {
		// when
		info, err := describe(t, baseSpec())

		// then
		require.NoError(t, err)
		assert.Equal(t, gridHeight, info.Rows)
		assert.Equal(t, gridWidth, info.Cols)
		assert.InDelta(t, -23.0, info.NorthLatitude, 1e-9)
		assert.InDelta(t, -47.0, info.WestLongitude, 1e-9)
		assert.InDelta(t, 0.01, info.CellLatitude, 1e-12)
		assert.InDelta(t, 0.01, info.CellLongitude, 1e-12)
		assert.Equal(t, 1.0, info.UnitToMeters)
	})

	t.Run("should describe a big-endian file the same way", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.BigEndian = true

		// when
		info, err := describe(t, spec)

		// then
		require.NoError(t, err)
		assert.Equal(t, gridHeight, info.Rows)
		assert.InDelta(t, -47.0, info.WestLongitude, 1e-9)
	})

	t.Run("should only need the tags: a file whose samples were cut off can still be described", func(t *testing.T) {
		// given
		data := helper.GeoTIFFWithSamples(baseSpec())
		path := writeTIFF(t, helper.Truncated(data, len(data)-10))

		// when
		info, err := elevationreader.NewGeoTIFF().Describe(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, gridHeight, info.Rows)
	})

	t.Run("should shift the grid by half a cell when the tie point is the center of a cell", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.PixelIsPoint = true

		// when
		info, err := describe(t, spec)

		// then
		require.NoError(t, err)
		assert.InDelta(t, -47.005, info.WestLongitude, 1e-9)
		assert.InDelta(t, -22.995, info.NorthLatitude, 1e-9)
	})

	t.Run("should keep a western limit past 180 within [-180, 180)", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.OriginLon = 190

		// when
		info, err := describe(t, spec)

		// then
		require.NoError(t, err)
		assert.InDelta(t, -170.0, info.WestLongitude, 1e-9)
	})

	t.Run("should take meters as the unit when the file does not say", func(t *testing.T) {
		// when
		info, err := describe(t, baseSpec())

		// then
		require.NoError(t, err)
		assert.Equal(t, 1.0, info.UnitToMeters)
	})

	t.Run("should take meters as the unit when the file says so", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.VerticalUnit = new(helper.UnitMeter)

		// when
		info, err := describe(t, spec)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1.0, info.UnitToMeters)
	})

	t.Run("should convert international feet to meters", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.VerticalUnit = new(helper.UnitFoot)

		// when
		info, err := describe(t, spec)

		// then
		require.NoError(t, err)
		assert.InDelta(t, 0.3048, info.UnitToMeters, 1e-12)
	})

	t.Run("should convert US survey feet to meters", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.VerticalUnit = new(helper.UnitUSFoot)

		// when
		info, err := describe(t, spec)

		// then
		require.NoError(t, err)
		assert.InDelta(t, 1200.0/3937.0, info.UnitToMeters, 1e-12)
	})

	t.Run("should refuse a unit that is neither meters nor feet", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.VerticalUnit = new(helper.UnitOtherRaw)

		// when
		_, err := describe(t, spec)

		// then
		require.ErrorIs(t, err, domain.ErrElevationUnitUnsupported)
		assert.ErrorContains(t, err, "9999")
	})

	t.Run("should refuse a raster in a projected coordinate system", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.ProjectedCRS = true

		// when
		_, err := describe(t, spec)

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})

	t.Run("should refuse a raster with several bands", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SamplesPerPixel = 3

		// when
		_, err := describe(t, spec)

		// then
		require.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
		assert.ErrorContains(t, err, "unsupported encoding")
	})

	t.Run("should refuse content that is not a TIFF file", func(t *testing.T) {
		// given
		path := writeTIFF(t, helper.NotTIFFContent())

		// when
		_, err := elevationreader.NewGeoTIFF().Describe(path)

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})

	t.Run("should refuse a file that has gone", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "gone.tif")

		// when
		_, err := elevationreader.NewGeoTIFF().Describe(path)

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})
}
