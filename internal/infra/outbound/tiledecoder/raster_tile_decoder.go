// Package tiledecoder implements the outbound adapter that decodes the image of
// a base map tile, so drawing can read its pixels.
package tiledecoder

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/webp"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// Raster implements domain.TileDecoder for the tiles that are images: PNG, JPEG
// and WebP. Vector tiles are not something it decodes.
type Raster struct{}

// NewRaster creates a Raster decoder.
func NewRaster() Raster {
	return Raster{}
}

// Decode decodes the image in data, of the given format ("png", "jpg" or
// "webp"), into pixels of red, green, blue and alpha, not premultiplied. An
// error says why the bytes are not an image of that format.
func (Raster) Decode(format string, data []byte) (domain.TileImage, error) {
	var (
		decoded image.Image
		err     error
	)

	switch format {
	case "png":
		decoded, err = png.Decode(bytes.NewReader(data))
	case "jpg":
		decoded, err = jpeg.Decode(bytes.NewReader(data))
	case "webp":
		decoded, err = webp.Decode(bytes.NewReader(data))
	default:
		return domain.TileImage{}, fmt.Errorf("tile format %q is not an image format this tool decodes", format)
	}
	if err != nil {
		return domain.TileImage{}, fmt.Errorf("decoding a %s tile: %w", format, err)
	}

	bounds := decoded.Bounds()
	pixels := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(pixels, pixels.Bounds(), decoded, bounds.Min, draw.Src)

	return domain.NewTileImage(bounds.Dx(), bounds.Dy(), pixels.Pix), nil
}
