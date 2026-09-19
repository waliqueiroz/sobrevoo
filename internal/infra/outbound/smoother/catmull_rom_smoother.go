// Package smoother holds the adapters that implement the domain.Smoother
// port. CatmullRomSmoother uses a Catmull-Rom spline as the reference
// "smooth" position for each point.
package smoother

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// CatmullRomSmoother implements domain.Smoother. For each interior point, it computes
// a reference position from a Catmull-Rom spline through its neighbors, and
// blends the point toward that reference by an amount controlled by level
// — this reduces point-to-point jitter (FR-013) without discarding any
// point or touching its Elevation/Time (that stays Simplifier's and the
// discard functions' job). The first and last point are never touched,
// since they have no neighbor on one side to smooth against.
type CatmullRomSmoother struct{}

// NewCatmullRomSmoother creates a CatmullRomSmoother.
func NewCatmullRomSmoother() CatmullRomSmoother {
	return CatmullRomSmoother{}
}

// Smooth returns a new slice of points with the same length as points; only
// Latitude/Longitude may change (FR-013, FR-015).
func (CatmullRomSmoother) Smooth(points []domain.TrackPoint, level domain.Level) []domain.TrackPoint {
	if len(points) < 4 {
		// Not enough neighbors on both sides to compute a meaningful
		// Catmull-Rom reference; leave the route as it is.
		return points
	}

	alpha := blendFactorFor(level)
	last := len(points) - 1

	result := make([]domain.TrackPoint, len(points))
	copy(result, points)

	for i := 1; i < last; i++ {
		p0 := points[clamp(i-2, 0, last)]
		p1 := points[clamp(i-1, 0, last)]
		p2 := points[clamp(i+1, 0, last)]
		p3 := points[clamp(i+2, 0, last)]

		referenceLat := catmullRom(p0.Latitude, p1.Latitude, p2.Latitude, p3.Latitude, 0.5)
		referenceLon := catmullRom(p0.Longitude, p1.Longitude, p2.Longitude, p3.Longitude, 0.5)

		result[i].Latitude = blend(points[i].Latitude, referenceLat, alpha)
		result[i].Longitude = blend(points[i].Longitude, referenceLon, alpha)
	}

	return result
}

// blendFactorFor maps a Level to how strongly each point is pulled toward
// its Catmull-Rom reference position: the higher the level, the stronger
// the smoothing. The three values are all kept below the point where, for
// the worst case of a perfectly alternating (highest-frequency) zig-zag,
// the blended result would overshoot past zero and start growing again —
// which would make a higher level produce more jitter instead of less.
// Keeping every level in the same monotonic region of the blend is what
// guarantees low < medium < high always reduces jitter further (SC-007).
func blendFactorFor(level domain.Level) float64 {
	switch level {
	case domain.LevelLow:
		return 0.15
	case domain.LevelHigh:
		return 0.42
	default:
		return 0.30
	}
}

func blend(original, reference, alpha float64) float64 {
	return original*(1-alpha) + reference*alpha
}

func clamp(i, min, max int) int {
	switch {
	case i < min:
		return min
	case i > max:
		return max
	default:
		return i
	}
}

// catmullRom evaluates the Catmull-Rom spline through control points
// p0, p1, p2, p3 at parameter t in [0, 1], where t=0 is p1 and t=1 is p2.
func catmullRom(p0, p1, p2, p3, t float64) float64 {
	t2 := t * t
	t3 := t2 * t

	return 0.5 * ((2 * p1) +
		(-p0+p2)*t +
		(2*p0-5*p1+4*p2-p3)*t2 +
		(-p0+3*p1-3*p2+p3)*t3)
}
