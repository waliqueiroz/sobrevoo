package domain

// FrameImage is a drawn frame: Resolution.Width × Resolution.Height pixels of
// 8 bits per channel, red, green and blue (no alpha), row by row from the top.
type FrameImage struct {
	Resolution Resolution
	Pix        []uint8
}

// NewFrameImage makes an image of the resolution filled with background.
func NewFrameImage(resolution Resolution, background RGB) FrameImage {
	image := FrameImage{Resolution: resolution, Pix: make([]uint8, 3*resolution.Pixels())}
	for i := 0; i < len(image.Pix); i += 3 {
		image.Pix[i], image.Pix[i+1], image.Pix[i+2] = background.R, background.G, background.B
	}
	return image
}

// Set paints the pixel at column x and row y, from the top left corner.
func (f FrameImage) Set(x, y int, c RGB) {
	offset := 3 * (y*f.Resolution.Width + x)
	f.Pix[offset], f.Pix[offset+1], f.Pix[offset+2] = c.R, c.G, c.B
}

// At is the color of the pixel at column x and row y.
func (f FrameImage) At(x, y int) RGB {
	offset := 3 * (y*f.Resolution.Width + x)
	return RGB{f.Pix[offset], f.Pix[offset+1], f.Pix[offset+2]}
}

// FrameStats says what drawing a frame found in the terrain it shows (the
// pixels a ray of the camera reached, before the trail and the marker).
type FrameStats struct {
	// MapHole is true when some terrain in view has no map image (a tile the
	// slice lacks, or where no tile exists); ElevationHole, when some terrain
	// in view lies over a cell with no elevation value.
	MapHole, ElevationHole bool
}
