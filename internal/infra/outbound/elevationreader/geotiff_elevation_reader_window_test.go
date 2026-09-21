package elevationreader_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/elevationreader"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

func readWindow(t *testing.T, spec helper.GeoTIFFSpec, window domain.GridWindow) []float32 {
	t.Helper()
	window2, err := elevationreader.NewGeoTIFF().ReadWindow(writeTIFF(t, helper.GeoTIFFWithSamples(spec)), window)
	require.NoError(t, err)
	return window2.Values
}

// expectedWindow is what the samples of window should be, for a grid filled
// by gridValues(step).
func expectedWindow(step float64, window domain.GridWindow) []float32 {
	var values []float32
	for r := window.FirstRow; r < window.FirstRow+window.Rows; r++ {
		for c := window.FirstCol; c < window.FirstCol+window.Cols; c++ {
			values = append(values, float32(float64(r)*step+float64(c)))
		}
	}
	return values
}

var (
	wholeGrid   = domain.GridWindow{FirstRow: 0, FirstCol: 0, Rows: gridHeight, Cols: gridWidth}
	middleBlock = domain.GridWindow{FirstRow: 2, FirstCol: 3, Rows: 4, Cols: 5}
)

func Test_GeoTIFF_ReadWindow_Types(t *testing.T) {
	t.Run("should read float32 samples, little-endian", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Float32

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.Equal(t, expectedWindow(100, middleBlock), values)
	})

	t.Run("should read float32 samples, big-endian", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Float32
		spec.BigEndian = true

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.Equal(t, expectedWindow(100, middleBlock), values)
	})

	t.Run("should read float64 samples", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Float64

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.Equal(t, expectedWindow(100, middleBlock), values)
	})

	t.Run("should read int16 samples, including negative ones", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.Values[3][4] = -250

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		expected := expectedWindow(100, middleBlock)
		expected[1*middleBlock.Cols+1] = -250
		assert.Equal(t, expected, values)
	})

	t.Run("should read uint16 samples", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Uint16
		spec.Values[3][4] = 40000

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		expected := expectedWindow(100, middleBlock)
		expected[1*middleBlock.Cols+1] = 40000
		assert.Equal(t, expected, values)
	})

	t.Run("should read int32 samples", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Int32
		spec.Values[3][4] = -100000

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		expected := expectedWindow(100, middleBlock)
		expected[1*middleBlock.Cols+1] = -100000
		assert.Equal(t, expected, values)
	})

	t.Run("should read uint8 samples", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.SampleType = helper.Uint8
		spec.Values = gridValues(10)

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.Equal(t, expectedWindow(10, middleBlock), values)
	})

	t.Run("should read big-endian int16 samples", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.BigEndian = true

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.Equal(t, expectedWindow(100, middleBlock), values)
	})
}

func Test_GeoTIFF_ReadWindow_Layouts(t *testing.T) {
	t.Run("should read the whole grid", func(t *testing.T) {
		// when
		values := readWindow(t, baseSpec(), wholeGrid)

		// then
		assert.Equal(t, expectedWindow(100, wholeGrid), values)
	})

	t.Run("should read a single sample", func(t *testing.T) {
		// given
		window := domain.GridWindow{FirstRow: 7, FirstCol: 9, Rows: 1, Cols: 1}

		// when
		values := readWindow(t, baseSpec(), window)

		// then
		assert.Equal(t, []float32{709}, values)
	})

	t.Run("should read a window that touches the north-west corner", func(t *testing.T) {
		// given
		window := domain.GridWindow{FirstRow: 0, FirstCol: 0, Rows: 2, Cols: 3}

		// when
		values := readWindow(t, baseSpec(), window)

		// then
		assert.Equal(t, expectedWindow(100, window), values)
	})

	t.Run("should read a window across several strips", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.RowsPerStrip = 3

		// when
		values := readWindow(t, spec, domain.GridWindow{FirstRow: 1, FirstCol: 2, Rows: 6, Cols: 7})

		// then
		assert.Equal(t, expectedWindow(100, domain.GridWindow{FirstRow: 1, FirstCol: 2, Rows: 6, Cols: 7}), values)
	})

	t.Run("should read the last, shorter strip", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.RowsPerStrip = 3 // strips of 3, 3 and 2 rows
		window := domain.GridWindow{FirstRow: 6, FirstCol: 0, Rows: 2, Cols: gridWidth}

		// when
		values := readWindow(t, spec, window)

		// then
		assert.Equal(t, expectedWindow(100, window), values)
	})

	t.Run("should read a window across tiles that do not divide the grid evenly", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.TileWidth, spec.TileLength = 4, 3

		// when
		values := readWindow(t, spec, wholeGrid)

		// then
		assert.Equal(t, expectedWindow(100, wholeGrid), values)
	})

	t.Run("should read a window inside a single tile", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.TileWidth, spec.TileLength = 4, 3
		window := domain.GridWindow{FirstRow: 3, FirstCol: 4, Rows: 2, Cols: 2}

		// when
		values := readWindow(t, spec, window)

		// then
		assert.Equal(t, expectedWindow(100, window), values)
	})

	t.Run("should read big-endian tiles", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.TileWidth, spec.TileLength = 4, 3
		spec.BigEndian = true
		spec.SampleType = helper.Float32

		// when
		values := readWindow(t, spec, middleBlock)

		// then
		assert.Equal(t, expectedWindow(100, middleBlock), values)
	})

	t.Run("should give no samples for an empty window", func(t *testing.T) {
		// when
		values := readWindow(t, baseSpec(), domain.GridWindow{FirstRow: 2, FirstCol: 2})

		// then
		assert.Empty(t, values)
	})

	t.Run("should not read strips outside the window: a file cut after the first strips still gives them", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.RowsPerStrip = 2 // four strips
		data := helper.GeoTIFFWithSamples(spec)
		path := writeTIFF(t, helper.Truncated(data, len(data)-1)) // cuts into the last strip only
		window := domain.GridWindow{FirstRow: 0, FirstCol: 0, Rows: 4, Cols: gridWidth}

		// when
		read, err := elevationreader.NewGeoTIFF().ReadWindow(path, window)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedWindow(100, window), read.Values)
	})
}

func Test_GeoTIFF_ReadWindow_Corruption(t *testing.T) {
	t.Run("should refuse a file cut off inside the samples the window needs", func(t *testing.T) {
		// given
		spec := baseSpec()
		spec.RowsPerStrip = 2
		data := helper.GeoTIFFWithSamples(spec)
		path := writeTIFF(t, helper.Truncated(data, len(data)-1))

		// when
		_, err := elevationreader.NewGeoTIFF().ReadWindow(path, wholeGrid)

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})

	t.Run("should refuse a window that is not inside the grid", func(t *testing.T) {
		// when
		_, err := elevationreader.NewGeoTIFF().ReadWindow(
			writeTIFF(t, helper.GeoTIFFWithSamples(baseSpec())),
			domain.GridWindow{FirstRow: 5, FirstCol: 0, Rows: 9, Cols: 1},
		)

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})

	t.Run("should refuse a file that has gone", func(t *testing.T) {
		// when
		_, err := elevationreader.NewGeoTIFF().ReadWindow(writeTIFF(t, nil)+".gone", wholeGrid)

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})

	t.Run("should refuse content that is not a TIFF file", func(t *testing.T) {
		// when
		_, err := elevationreader.NewGeoTIFF().ReadWindow(writeTIFF(t, helper.NotTIFFContent()), wholeGrid)

		// then
		assert.ErrorIs(t, err, domain.ErrGeoDataContentUnreadable)
	})

	t.Run("should never give a partial window: the error comes with no samples", func(t *testing.T) {
		// given
		data := helper.GeoTIFFWithSamples(baseSpec())
		path := writeTIFF(t, helper.Truncated(data, len(data)-5))

		// when
		read, err := elevationreader.NewGeoTIFF().ReadWindow(path, wholeGrid)

		// then
		require.Error(t, err)
		assert.Empty(t, read.Values)
	})
}
