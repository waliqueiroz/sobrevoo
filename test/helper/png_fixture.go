package helper

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
)

// TileSize is the side of the tiles these fixtures draw, in pixels.
const TileSize = 256

// Color is an opaque color of a fixture tile.
type Color struct {
	R, G, B uint8
}

func (c Color) rgba() color.NRGBA { return color.NRGBA{R: c.R, G: c.G, B: c.B, A: 255} }

func encodePNG(img image.Image) []byte {
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		panic("encoding a PNG tile fixture: " + err.Error())
	}
	return out.Bytes()
}

// PNGTile is a 256 × 256 PNG tile of one color.
func PNGTile(c Color) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, TileSize, TileSize))
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			img.SetNRGBA(x, y, c.rgba())
		}
	}
	return encodePNG(img)
}

// CheckerPNGTile is a 256 × 256 PNG tile in a checkerboard of squares of the
// given side, alternating a and b, starting with a at the top left corner.
func CheckerPNGTile(a, b Color, square int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, TileSize, TileSize))
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			c := a
			if (x/square+y/square)%2 == 1 {
				c = b
			}
			img.SetNRGBA(x, y, c.rgba())
		}
	}
	return encodePNG(img)
}

// PositionPNGTile is a 256 × 256 PNG tile a person can recognize in a drawn
// frame: a checkerboard of 32-pixel squares in two tones whose hue depends on
// the position of the tile (z, x and y), with a 4-pixel darker border.
func PositionPNGTile(z, x, y int) []byte {
	hue := uint8((x*37 + y*91 + z*53) % 200)
	light := Color{R: 100 + hue/2, G: 200 - hue/2, B: 120 + hue/3}
	dark := Color{R: light.R - 40, G: light.G - 40, B: light.B - 40}
	border := Color{R: 30, G: 30, B: 30}

	img := image.NewNRGBA(image.Rect(0, 0, TileSize, TileSize))
	for py := 0; py < TileSize; py++ {
		for px := 0; px < TileSize; px++ {
			c := light
			if (px/32+py/32)%2 == 1 {
				c = dark
			}
			if px < 4 || py < 4 || px >= TileSize-4 || py >= TileSize-4 {
				c = border
			}
			img.SetNRGBA(px, py, c.rgba())
		}
	}
	return encodePNG(img)
}

// TransparentPNGTile is a 256 × 256 PNG tile whose left half is fully
// transparent and whose right half is opaque red.
func TransparentPNGTile() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, TileSize, TileSize))
	for y := 0; y < TileSize; y++ {
		for x := TileSize / 2; x < TileSize; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	return encodePNG(img)
}

// JPEGTile is a 256 × 256 JPEG tile of one color, at the best quality.
func JPEGTile(c Color) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, TileSize, TileSize))
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			img.SetNRGBA(x, y, c.rgba())
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 100}); err != nil {
		panic("encoding a JPEG tile fixture: " + err.Error())
	}
	return out.Bytes()
}

// WebPTile is a lossless WebP image of one white pixel (1 × 1): the smallest
// valid one, which golang.org/x/image/webp decodes.
func WebPTile() []byte {
	data, err := base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")
	if err != nil {
		panic("decoding the WebP tile fixture: " + err.Error())
	}
	return data
}

// NotAnImage is bytes that no image decoder reads.
func NotAnImage() []byte {
	return []byte("this is not an image at all")
}

// planMarkPrefix starts the text of the chunk in which a frame of this tool says
// which plan it was drawn from.
const planMarkPrefix = "Sobrevoo\x00plan="

// WithoutPlanMark is the bytes of a PNG frame of this tool without the text
// chunk that says which plan it was drawn from — what a frame drawn before that
// chunk existed looks like. Every other chunk stays as it was, checksums
// included.
func WithoutPlanMark(data []byte) []byte {
	if len(data) < 8 {
		return data
	}

	out := append([]byte(nil), data[:8]...)
	for position := 8; position+12 <= len(data); {
		length := int(binary.BigEndian.Uint32(data[position:]))
		end := position + 12 + length
		if end > len(data) {
			out = append(out, data[position:]...)
			break
		}

		chunk := data[position:end]
		isPlanMark := string(chunk[4:8]) == "tEXt" && bytes.HasPrefix(chunk[8:8+length], []byte(planMarkPrefix))
		if !isPlanMark {
			out = append(out, chunk...)
		}
		position = end
	}
	return out
}

// TruncatedAt is the first size bytes of data: a file cut short.
func TruncatedAt(data []byte, size int) []byte {
	return append([]byte(nil), data[:min(size, len(data))]...)
}
