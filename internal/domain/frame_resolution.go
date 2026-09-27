package domain

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// MinFrameSide and MaxFrameSide bound each side of a frame, in pixels.
	MinFrameSide = 180
	MaxFrameSide = 3840

	// MaxFramePixels bounds the pixels of a frame: those of 3840 × 2160.
	MaxFramePixels = 8_294_400
)

// Resolution is the size, in pixels, of the images the frames are drawn as.
type Resolution struct {
	Width, Height int
}

// NewResolution checks a resolution: width and height are even (the video
// stage that follows needs it), each from MinFrameSide to MaxFrameSide, and
// together at most MaxFramePixels pixels. The error is ErrInvalidResolution.
func NewResolution(width, height int) (Resolution, error) {
	text := fmt.Sprintf("%dx%d", width, height)

	switch {
	case width%2 != 0 || height%2 != 0:
		return Resolution{}, fmt.Errorf("%w: %s, both sides must be even", ErrInvalidResolution, text)
	case width < MinFrameSide || width > MaxFrameSide || height < MinFrameSide || height > MaxFrameSide:
		return Resolution{}, fmt.Errorf("%w: %s, each side must be from %d to %d pixels", ErrInvalidResolution, text, MinFrameSide, MaxFrameSide)
	case width*height > MaxFramePixels:
		return Resolution{}, fmt.Errorf("%w: %s, at most %d pixels in all", ErrInvalidResolution, text, MaxFramePixels)
	}

	return Resolution{Width: width, Height: height}, nil
}

// ParseResolution reads a resolution written WIDTHxHEIGHT (a lowercase or an
// uppercase x, digits only) and checks it as NewResolution does.
func ParseResolution(text string) (Resolution, error) {
	invalid := func() (Resolution, error) {
		return Resolution{}, fmt.Errorf("%w: %q, use WIDTHxHEIGHT in pixels, for example 1920x1080", ErrInvalidResolution, text)
	}

	separator := strings.IndexAny(text, "xX")
	if separator < 0 {
		return invalid()
	}
	widthText, heightText := text[:separator], text[separator+1:]
	if !allDigits(widthText) || !allDigits(heightText) {
		return invalid()
	}

	width, errWidth := strconv.Atoi(widthText)
	height, errHeight := strconv.Atoi(heightText)
	if errWidth != nil || errHeight != nil {
		return Resolution{}, fmt.Errorf("%w: %q, each side must be from %d to %d pixels", ErrInvalidResolution, text, MinFrameSide, MaxFrameSide)
	}

	return NewResolution(width, height)
}

// Pixels is how many pixels an image of the resolution has.
func (r Resolution) Pixels() int {
	return r.Width * r.Height
}

// aspectTolerance is how much narrower than an aspect ratio a resolution may be
// and still count as having that shape.
const aspectTolerance = 0.01

// NarrowerThan reports whether an image of the resolution is narrower than a
// video of the aspect ratio: the camera's vertical field of view is fixed, so
// a narrower image sees less to the sides than the plan framed for, and the
// track may be cut off in the opening and the closing. A wider image only sees
// more. An aspect ratio that is not set is never wider than anything.
func (r Resolution) NarrowerThan(aspect AspectRatio) bool {
	return float64(r.Width)*float64(aspect.Height) < float64(aspect.Width)*float64(r.Height)*(1-aspectTolerance)
}

func allDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, c := range text {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
