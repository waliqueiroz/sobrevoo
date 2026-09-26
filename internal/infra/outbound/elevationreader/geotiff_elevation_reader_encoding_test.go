package elevationreader_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/elevationreader"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

func Test_GeoTIFF_ReadWindow_Compression(t *testing.T) {
	t.Run("should read Deflate strips", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Compression = helper.TIFFDeflate
		spec.RowsPerStrip = 3

		// when
		values := readWindow(t, spec, wholeGrid)

		// then
		assert.Equal(t, expectedWindow(100, wholeGrid), values)
	})

	t.Run("should read Adobe Deflate strips", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Compression = helper.TIFFAdobeDeflate

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.Equal(t, expectedWindow(100, middleBlock), values)
	})

	t.Run("should read LZW strips", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Compression = helper.TIFFLZW
		spec.RowsPerStrip = 3

		// when
		values := readWindow(t, spec, wholeGrid)

		// then
		assert.Equal(t, expectedWindow(100, wholeGrid), values)
	})

	t.Run("should read LZW tiles", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Compression = helper.TIFFLZW
		spec.TileWidth, spec.TileLength = 4, 3

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.Equal(t, expectedWindow(100, middleBlock), values)
	})

	t.Run("should read Deflate tiles, big-endian", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Compression = helper.TIFFDeflate
		spec.TileWidth, spec.TileLength = 4, 3
		spec.BigEndian = true

		// when
		values := readWindow(t, spec, wholeGrid)

		// then
		assert.Equal(t, expectedWindow(100, wholeGrid), values)
	})

	t.Run("should read a big grid that needs wider LZW codes", func(t *testing.T) {
		// given
		const size = 120
		values := make([][]float64, size)
		for r := range values {
			values[r] = make([]float64, size)
			for c := range values[r] {
				values[r][c] = float64((r*37 + c*11) % 5000)
			}
		}
		spec := helper.GeoTIFFSpec{
			Width: size, Height: size, OriginLon: 0, OriginLat: 10, ScaleX: 0.001, ScaleY: 0.001,
			SampleType: helper.Int16, Values: values, Compression: helper.TIFFLZW, RowsPerStrip: 40,
		}
		window := domain.GridWindow{FirstRow: 10, FirstCol: 5, Rows: 100, Cols: 100}

		// when
		read := readWindow(t, spec, window)

		// then
		require.Len(t, read, 100*100)
		assert.Equal(t, float32(values[10][5]), read[0])
		assert.Equal(t, float32(values[109][104]), read[len(read)-1])
	})
}

func Test_GeoTIFF_ReadWindow_Predictors(t *testing.T) {
	t.Run("should undo the horizontal predictor of int16 samples", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Compression = helper.TIFFDeflate
		spec.Predictor = 2

		// when
		values := readWindow(t, spec, wholeGrid)

		// then
		assert.Equal(t, expectedWindow(100, wholeGrid), values)
	})

	t.Run("should undo the horizontal predictor of big-endian int32 samples in tiles", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Int32
		spec.BigEndian = true
		spec.Compression = helper.TIFFLZW
		spec.Predictor = 2
		spec.TileWidth, spec.TileLength = 4, 3
		spec.Values[2][2] = -70000

		// when
		values := readWindow(t, spec, wholeGrid)

		// then
		expected := expectedWindow(100, wholeGrid)
		expected[2*gridWidth+2] = -70000
		assert.Equal(t, expected, values)
	})

	t.Run("should undo the horizontal predictor of uint8 samples", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Uint8
		spec.Values = gridValues(10)
		spec.Compression = helper.TIFFDeflate
		spec.Predictor = 2

		// when
		values := readWindow(t, spec, wholeGrid)

		// then
		assert.Equal(t, expectedWindow(10, wholeGrid), values)
	})

	t.Run("should undo the floating-point predictor, little-endian", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Float32
		spec.Compression = helper.TIFFDeflate
		spec.Predictor = 3
		spec.Values[4][5] = 1234.5

		// when
		values := readWindow(t, spec, wholeGrid)

		// then
		expected := expectedWindow(100, wholeGrid)
		expected[4*gridWidth+5] = 1234.5
		assert.Equal(t, expected, values)
	})

	t.Run("should undo the floating-point predictor, big-endian, in tiles", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Float32
		spec.BigEndian = true
		spec.Compression = helper.TIFFDeflate
		spec.Predictor = 3
		spec.TileWidth, spec.TileLength = 4, 3

		// when
		values := readWindow(t, spec, wholeGrid)

		// then
		assert.Equal(t, expectedWindow(100, wholeGrid), values)
	})

	t.Run("should undo the floating-point predictor of float64 samples", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Float64
		spec.Compression = helper.TIFFLZW
		spec.Predictor = 3

		// when
		values := readWindow(t, spec, wholeGrid)

		// then
		assert.Equal(t, expectedWindow(100, wholeGrid), values)
	})
}

func Test_GeoTIFF_ReadWindow_NoValueAndUnit(t *testing.T) {
	isNoValue := func(v float32) bool { return math.IsNaN(float64(v)) }

	t.Run("should mark a sample equal to the file's no-data value as having no value", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Values[3][4] = -9999
		spec.NoData = new("-9999")

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.True(t, isNoValue(values[1*middleBlock.Cols+1]))
		assert.False(t, isNoValue(values[0]))
	})

	t.Run("should never turn a sample without value into zero", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Values[3][4] = -9999
		spec.NoData = new("-9999")

		// when
		values := readWindow(t, spec, wholeGrid)

		// then
		noValueCount := 0
		for _, v := range values {
			if isNoValue(v) {
				noValueCount++
			}
		}
		assert.Equal(t, 1, noValueCount)
		assert.Equal(t, float32(0), values[0], "a real zero is a value")
	})

	t.Run("should read a float no-data value written in scientific notation", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Float32
		spec.Values[3][4] = -3.4028234663852886e+38
		spec.NoData = new("-3.4028234663852886e+38")

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.True(t, isNoValue(values[1*middleBlock.Cols+1]))
	})

	t.Run("should mark a NaN float sample as having no value", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Float32
		spec.Values[3][4] = math.NaN()

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.True(t, isNoValue(values[1*middleBlock.Cols+1]))
	})

	t.Run("should treat every sample as a value when the file has no no-data tag", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Values[3][4] = -9999

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.Equal(t, float32(-9999), values[1*middleBlock.Cols+1])
	})

	t.Run("should convert feet to meters", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.VerticalUnit = new(helper.UnitFoot)
		spec.Values[3][4] = 1000

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.InDelta(t, 304.8, values[1*middleBlock.Cols+1], 1e-3)
	})

	t.Run("should compare a sample with the no-data value before converting the unit", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.VerticalUnit = new(helper.UnitFoot)
		spec.Values[3][4] = -9999
		spec.NoData = new("-9999")

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.True(t, isNoValue(values[1*middleBlock.Cols+1]))
	})

	t.Run("should not let PixelIsPoint change the samples", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.PixelIsPoint = true

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.Equal(t, expectedWindow(100, middleBlock), values)
	})
}

func Test_GeoTIFF_ReadWindow_UnsupportedEncodings(t *testing.T) {
	t.Run("should refuse JPEG compression, saying what is unsupported", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Compression = helper.TIFFJPEG

		// when
		_, err := elevationreader.NewGeoTIFF().ReadWindow(writeTIFF(t, helper.GeoTIFFWithSamples(spec)), wholeGrid)

		// then
		require.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
		assert.ErrorContains(t, err, "unsupported encoding: compression 7")
	})

	t.Run("should refuse a raster with several bands", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SamplesPerPixel = 2

		// when
		_, err := elevationreader.NewGeoTIFF().ReadWindow(writeTIFF(t, helper.GeoTIFFWithSamples(spec)), wholeGrid)

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})

	t.Run("should refuse the floating-point predictor on integer samples", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Compression = helper.TIFFDeflate
		spec.Predictor = 3

		// when
		_, err := elevationreader.NewGeoTIFF().ReadWindow(writeTIFF(t, helper.GeoTIFFWithSamples(spec)), wholeGrid)

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})

	t.Run("should refuse compressed data that is not valid", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Compression = helper.TIFFDeflate
		data := helper.GeoTIFFWithSamples(spec)
		for i := len(data) - 20; i < len(data); i++ {
			data[i] = 0xFF
		}

		// when
		_, err := elevationreader.NewGeoTIFF().ReadWindow(writeTIFF(t, data), wholeGrid)

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})
}
