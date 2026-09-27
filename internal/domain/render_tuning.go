package domain

import "strconv"

// RenderVersion is the version of how a frame is drawn: it goes up whenever
// the algorithm, the colors or the patterns change in a way that shows, or the
// file of a frame changes what it says, so frames drawn by another version are
// never taken for frames of the same set. Version 2 writes, inside each image,
// the plan the frame was drawn from; the pixels are those of version 1.
const RenderVersion = 2

// RGB is a color, 8 bits per channel.
type RGB struct {
	R, G, B uint8
}

// The colors and patterns of a frame (specs/005-frame-rendering/contracts/
// frame-files.md). They are what the image means, not something to tune.
var (
	// BackgroundColor is where a ray meets no terrain: outside the slice and
	// above the horizon.
	BackgroundColor = RGB{0x20, 0x26, 0x2E}

	// NoMapColors are the two tones of the diagonal hatch of terrain with no
	// map tile; NoElevationColors, of the checkerboard of terrain over a cell
	// with no elevation.
	NoMapColors       = [2]RGB{{0xC8, 0xC8, 0xC8}, {0x6E, 0x6E, 0x6E}}
	NoElevationColors = [2]RGB{{0xFF, 0x00, 0xFF}, {0x3A, 0x00, 0x3A}}

	TrailColor       = RGB{0xFF, 0xB0, 0x00}
	TrailCasingColor = RGB{0x10, 0x10, 0x10}
	MarkerColor      = RGB{0xE5, 0x25, 0x2A}
	MarkerRingColor  = RGB{0xFF, 0xFF, 0xFF}
)

const (
	// PatternPeriod is the size, in screen pixels, of the hatch and of the
	// squares of the checkerboard.
	PatternPeriod = 12

	// The size of the trail and of the marker, as a share of the height of the
	// image, and the least each has, in pixels.
	TrailWidthRatio   = 0.005
	TrailMinWidth     = 2.0
	MarkerRadiusRatio = 0.012
	MarkerMinRadius   = 4.0
	MarkerRingRatio   = 0.003
	MarkerRingMin     = 1.5
)

// RenderTuning holds the heuristic constants of drawing a frame. They are
// injected (Constitution Principle VIII): the core defines the shape, an
// outbound configuration adapter provides the values. See
// specs/005-frame-rendering/research.md item 22.
type RenderTuning struct {
	// VerticalFOVDegrees is the vertical field of view of the camera, the same
	// one the camera plan and the level of detail assume.
	VerticalFOVDegrees float64

	// MinCameraClearanceMeters is how far above the terrain under it the
	// camera stays at least.
	MinCameraClearanceMeters float64

	// MinTiltForTargetDegrees is the tilt under which the observed point can no
	// longer be recovered from a frame; the marker is used instead.
	MinTiltForTargetDegrees float64

	// TrailLiftMeters is how far above the ground the trail and the marker are
	// drawn.
	TrailLiftMeters float64

	// DepthBiasMeters and DepthBiasRatio are the slack of the depth test of
	// the trail and the marker against the terrain: a fixed part and a share of
	// the distance.
	DepthBiasMeters float64
	DepthBiasRatio  float64

	// TileCacheBytes is the budget of decoded tiles kept in memory, and
	// Workers how many goroutines draw one frame. Neither changes the frame.
	TileCacheBytes int64
	Workers        int
}

// Fingerprint is a canonical text of the fields that change how a frame looks,
// which takes part in the identification of a set of frames: TileCacheBytes and
// Workers do not.
func (t RenderTuning) Fingerprint() string {
	format := func(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

	return format(t.VerticalFOVDegrees) + "|" +
		format(t.MinCameraClearanceMeters) + "|" +
		format(t.MinTiltForTargetDegrees) + "|" +
		format(t.TrailLiftMeters) + "|" +
		format(t.DepthBiasMeters) + "|" +
		format(t.DepthBiasRatio)
}
