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

// The fixed patterns and marks of a frame (specs/005-frame-rendering/contracts/
// frame-files.md). They are what the image means, not a matter of style, so
// they are not part of Appearance and stay the same whatever appearance is
// chosen (008-frame-appearance FR-005). The trail's color and width, the
// marker's fill color and radius, and the background color moved to
// Appearance — only the trail's casing, the marker's ring and the two "no
// data" patterns stay fixed here.
var (
	// NoMapColors are the two tones of the diagonal hatch of terrain with no
	// map tile; NoElevationColors, of the checkerboard of terrain over a cell
	// with no elevation.
	NoMapColors       = [2]RGB{{0xC8, 0xC8, 0xC8}, {0x6E, 0x6E, 0x6E}}
	NoElevationColors = [2]RGB{{0xFF, 0x00, 0xFF}, {0x3A, 0x00, 0x3A}}

	// TrailCasingColor is the dark casing under the trail's core (Appearance.
	// TrailColor); MarkerRingColor, the ring around the marker's fill
	// (Appearance.MarkerColor).
	TrailCasingColor = RGB{0x10, 0x10, 0x10}
	MarkerRingColor  = RGB{0xFF, 0xFF, 0xFF}
)

const (
	// PatternPeriod is the size, in screen pixels, of the hatch and of the
	// squares of the checkerboard.
	PatternPeriod = 12

	// TrailMinWidth and MarkerMinRadius are the least the trail and the
	// marker have, in pixels, whatever ratio Appearance chooses
	// (TrailWidthRatio, MarkerRadiusRatio) — so neither disappears on a small
	// resolution. MarkerRingRatio and MarkerRingMin size the fixed ring the
	// same way.
	TrailMinWidth   = 2.0
	MarkerMinRadius = 4.0
	MarkerRingRatio = 0.003
	MarkerRingMin   = 1.5
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
