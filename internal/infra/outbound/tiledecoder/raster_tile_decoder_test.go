package tiledecoder_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/tiledecoder"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

func Test_Raster_Decode(t *testing.T) {
	t.Run("should decode a PNG tile to RGBA pixels of its size", func(t *testing.T) {
		// given
		data := helper.PNGTile(helper.Color{R: 200, G: 30, B: 90})

		// when
		image, err := tiledecoder.NewRaster().Decode("png", data)

		// then
		require.NoError(t, err)
		assert.Equal(t, 256, image.Width)
		assert.Equal(t, 256, image.Height)
		assert.Len(t, image.Pix, 4*256*256)
		r, g, b, a := image.At(100, 200)
		assert.Equal(t, [4]uint8{200, 30, 90, 255}, [4]uint8{r, g, b, a})
	})

	t.Run("should keep the layout of a checkerboard", func(t *testing.T) {
		// given
		data := helper.CheckerPNGTile(helper.Color{R: 255, G: 255, B: 255}, helper.Color{R: 0, G: 0, B: 0}, 32)

		// when
		image, err := tiledecoder.NewRaster().Decode("png", data)

		// then
		require.NoError(t, err)
		first, _, _, _ := image.At(0, 0)
		next, _, _, _ := image.At(32, 0)
		below, _, _, _ := image.At(0, 32)
		diagonal, _, _, _ := image.At(32, 32)
		assert.Equal(t, uint8(255), first)
		assert.Equal(t, uint8(0), next)
		assert.Equal(t, uint8(0), below)
		assert.Equal(t, uint8(255), diagonal)
	})

	t.Run("should keep the transparency of a PNG, not premultiplied", func(t *testing.T) {
		// given
		data := helper.TransparentPNGTile()

		// when
		image, err := tiledecoder.NewRaster().Decode("png", data)

		// then
		require.NoError(t, err)
		_, _, _, left := image.At(10, 10)
		r, g, b, right := image.At(200, 10)
		assert.Equal(t, uint8(0), left)
		assert.Equal(t, [4]uint8{255, 0, 0, 255}, [4]uint8{r, g, b, right})
	})

	t.Run("should decode a JPEG tile, to within what the format keeps", func(t *testing.T) {
		// given
		data := helper.JPEGTile(helper.Color{R: 40, G: 160, B: 220})

		// when
		image, err := tiledecoder.NewRaster().Decode("jpg", data)

		// then
		require.NoError(t, err)
		assert.Equal(t, 256, image.Width)
		r, g, b, a := image.At(128, 128)
		assert.InDelta(t, 40, float64(r), 3)
		assert.InDelta(t, 160, float64(g), 3)
		assert.InDelta(t, 220, float64(b), 3)
		assert.Equal(t, uint8(255), a)
	})

	t.Run("should decode a WebP tile", func(t *testing.T) {
		// given
		data := helper.WebPTile()

		// when
		image, err := tiledecoder.NewRaster().Decode("webp", data)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, image.Width)
		assert.Equal(t, 1, image.Height)
		assert.Len(t, image.Pix, 4)
	})

	t.Run("should decode the same bytes to the same pixels", func(t *testing.T) {
		// given
		data := helper.PositionPNGTile(16, 24122, 36869)

		// when
		first, err1 := tiledecoder.NewRaster().Decode("png", data)
		second, err2 := tiledecoder.NewRaster().Decode("png", data)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, first.Pix, second.Pix)
	})

	t.Run("should refuse a format it does not know", func(t *testing.T) {
		// given
		data := helper.PNGTile(helper.Color{})

		for _, format := range []string{"gif", "pbf", ""} {
			// when
			_, err := tiledecoder.NewRaster().Decode(format, data)

			// then
			assert.Error(t, err, "%q", format)
			assert.ErrorContains(t, err, "format")
		}
	})

	t.Run("should refuse bytes that are not an image, and a PNG that is cut short", func(t *testing.T) {
		// given
		png := helper.PNGTile(helper.Color{R: 1})

		// when
		_, notImage := tiledecoder.NewRaster().Decode("png", helper.NotAnImage())
		_, truncated := tiledecoder.NewRaster().Decode("png", png[:len(png)/2])
		_, wrongFormat := tiledecoder.NewRaster().Decode("jpg", png)

		// then
		assert.Error(t, notImage)
		assert.Error(t, truncated)
		assert.Error(t, wrongFormat)
	})
}
