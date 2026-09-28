package domain

import (
	"fmt"
	"strconv"
)

const (
	// MinTrailWidthRatio and MaxTrailWidthRatio bound the accepted trail
	// width, as a share of the height of the frame (inclusive).
	MinTrailWidthRatio = 0.0005
	MaxTrailWidthRatio = 0.05

	// MinMarkerRadiusRatio and MaxMarkerRadiusRatio bound the accepted marker
	// radius, as a share of the height of the frame (inclusive).
	MinMarkerRadiusRatio = 0.001
	MaxMarkerRadiusRatio = 0.1
)

// Appearance is what a viewer sees drawn over the terrain of a frame: the
// trail's color and width, the marker's color and radius, and the color of
// the background. It does not include the hatch of "no map" (NoMapColors) or
// the checkerboard of "no elevation" (NoElevationColors), which are fixed and
// are what the image means, not a matter of style.
type Appearance struct {
	// TrailColor is the color of the core of the trail; TrailCasingColor, the
	// dark casing under it, is fixed.
	TrailColor RGB

	// TrailWidthRatio is the width of the trail, as a share of the height of
	// the frame.
	TrailWidthRatio float64

	// MarkerColor is the color of the fill of the marker; MarkerRingColor,
	// the ring around it, is fixed.
	MarkerColor RGB

	// MarkerRadiusRatio is the radius of the marker, as a share of the
	// height of the frame.
	MarkerRadiusRatio float64

	// BackgroundColor is where a ray meets no terrain: outside the slice,
	// above the horizon, and under a partly transparent pixel of the base
	// map.
	BackgroundColor RGB
}

// NewAppearance checks trailWidthRatio and markerRadiusRatio are within the
// documented range. The errors are ErrInvalidTrailWidth and
// ErrInvalidMarkerRadius; the colors are assumed already parsed (ParseColor),
// since no combination of RGB is invalid by itself.
func NewAppearance(trailColor RGB, trailWidthRatio float64, markerColor RGB, markerRadiusRatio float64, backgroundColor RGB) (Appearance, error) {
	if trailWidthRatio < MinTrailWidthRatio || trailWidthRatio > MaxTrailWidthRatio {
		return Appearance{}, fmt.Errorf("%w: %v, must be from %v to %v", ErrInvalidTrailWidth, trailWidthRatio, MinTrailWidthRatio, MaxTrailWidthRatio)
	}
	if markerRadiusRatio < MinMarkerRadiusRatio || markerRadiusRatio > MaxMarkerRadiusRatio {
		return Appearance{}, fmt.Errorf("%w: %v, must be from %v to %v", ErrInvalidMarkerRadius, markerRadiusRatio, MinMarkerRadiusRatio, MaxMarkerRadiusRatio)
	}

	return Appearance{
		TrailColor:        trailColor,
		TrailWidthRatio:   trailWidthRatio,
		MarkerColor:       markerColor,
		MarkerRadiusRatio: markerRadiusRatio,
		BackgroundColor:   backgroundColor,
	}, nil
}

// ParseColor reads a color written "#RRGGBB": "#" followed by exactly 6
// hexadecimal digits, upper or lower case. The error is ErrInvalidColor.
func ParseColor(text string) (RGB, error) {
	invalid := func() (RGB, error) {
		return RGB{}, fmt.Errorf("%w: %q, expected #RRGGBB, for example #FFB000", ErrInvalidColor, text)
	}

	if len(text) != 7 || text[0] != '#' {
		return invalid()
	}

	value, err := strconv.ParseUint(text[1:], 16, 32)
	if err != nil {
		return invalid()
	}

	return RGB{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value)}, nil
}

// Fingerprint is a canonical text of the five fields, which takes part in the
// identification of a set of frames (FrameSetID), the same way
// RenderTuning.Fingerprint already does.
func (a Appearance) Fingerprint() string {
	color := func(c RGB) string { return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B) }
	ratio := func(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

	return color(a.TrailColor) + "|" +
		ratio(a.TrailWidthRatio) + "|" +
		color(a.MarkerColor) + "|" +
		ratio(a.MarkerRadiusRatio) + "|" +
		color(a.BackgroundColor)
}
