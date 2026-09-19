// Package simplifier holds the adapters that implement the
// domain.Simplifier port. DouglasPeucker uses the Douglas-Peucker
// line simplification algorithm.
package simplifier

import (
	"math"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// DouglasPeucker implements domain.Simplifier via the Douglas-Peucker
// algorithm. Latitude/longitude are treated as plane coordinates when
// measuring perpendicular distance — an approximation that is accurate
// enough for reducing point density on the small scale of an individual
// activity's track, and keeps the algorithm simple to implement and test
// directly (research.md item 4).
type DouglasPeucker struct{}

// NewDouglasPeucker creates a DouglasPeucker.
func NewDouglasPeucker() DouglasPeucker {
	return DouglasPeucker{}
}

// Simplify removes points that lie within level's tolerance of the
// straight line connecting their neighbors, recursively, while always
// keeping the first and last point (FR-012, FR-014). Points that are kept
// retain their original Elevation/Time untouched.
func (DouglasPeucker) Simplify(points []domain.TrackPoint, level domain.Level) []domain.TrackPoint {
	if len(points) < 3 {
		return points
	}

	kept := make([]bool, len(points))
	kept[0] = true
	kept[len(points)-1] = true

	simplifySegment(points, 0, len(points)-1, toleranceFor(level), kept)

	result := make([]domain.TrackPoint, 0, len(points))
	for i, isKept := range kept {
		if isKept {
			result = append(result, points[i])
		}
	}

	return result
}

// toleranceFor maps a Level to a perpendicular-distance tolerance in
// degrees: the higher the level, the more aggressive the simplification.
func toleranceFor(level domain.Level) float64 {
	switch level {
	case domain.LevelLow:
		return 0.00002 // ~2m at the equator
	case domain.LevelHigh:
		return 0.00015 // ~15m at the equator
	default:
		return 0.00006 // ~6m at the equator
	}
}

func simplifySegment(points []domain.TrackPoint, start, end int, tolerance float64, kept []bool) {
	if end <= start+1 {
		return
	}

	maxDistance := -1.0
	maxIndex := -1

	for i := start + 1; i < end; i++ {
		d := perpendicularDistance(points[i], points[start], points[end])
		if d > maxDistance {
			maxDistance = d
			maxIndex = i
		}
	}

	if maxDistance > tolerance {
		kept[maxIndex] = true
		simplifySegment(points, start, maxIndex, tolerance, kept)
		simplifySegment(points, maxIndex, end, tolerance, kept)
	}
}

// perpendicularDistance returns the distance from p to the infinite line
// through a and b, treating Longitude/Latitude as plane X/Y coordinates.
func perpendicularDistance(p, a, b domain.TrackPoint) float64 {
	dx := b.Longitude - a.Longitude
	dy := b.Latitude - a.Latitude

	if dx == 0 && dy == 0 {
		return math.Hypot(p.Longitude-a.Longitude, p.Latitude-a.Latitude)
	}

	numerator := math.Abs(dy*p.Longitude - dx*p.Latitude + b.Longitude*a.Latitude - b.Latitude*a.Longitude)
	denominator := math.Hypot(dx, dy)

	return numerator / denominator
}
