package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/tile_decoder.go -package mockdomain . TileDecoder

// TileDecoder decodes the image of one base map tile, so drawing can read its
// pixels. Concrete implementations (one per family of formats) live in
// internal/infra/outbound/tiledecoder.
type TileDecoder interface {
	// Decode decodes the image in data. format is the tile format of the slice:
	// "png", "jpg" or "webp". Bytes that are not a readable image fail with an
	// error the caller wraps in ErrSliceFileInvalid, naming the tile: the
	// decoder knows no tile, only bytes.
	Decode(format string, data []byte) (TileImage, error)
}

// TileImage is the decoded image of a tile: Width × Height pixels of 8 bits
// per channel, red, green, blue and alpha (not premultiplied), row by row from
// the top.
type TileImage struct {
	Width, Height int
	Pix           []uint8
}

// NewTileImage builds a tile image out of its pixels.
func NewTileImage(width, height int, pix []uint8) TileImage {
	return TileImage{Width: width, Height: height, Pix: pix}
}

// At is the pixel at column x and row y, from the top left corner.
func (i TileImage) At(x, y int) (r, g, b, a uint8) {
	offset := 4 * (y*i.Width + x)
	return i.Pix[offset], i.Pix[offset+1], i.Pix[offset+2], i.Pix[offset+3]
}
