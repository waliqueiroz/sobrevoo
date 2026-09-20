package domain

import (
	"math"
	"sort"
)

// PlanarRoute is a route projected onto a LocalPlane: the positions of its
// points and the distance travelled, in meters, at each of them.
type PlanarRoute struct {
	Points    []PlanePoint
	Distances []float64
}

// Length is the total distance travelled along the route, in meters.
func (p PlanarRoute) Length() float64 {
	return p.Distances[len(p.Distances)-1]
}

// Span is twice the largest distance from the plane's center to a point of
// the route: roughly the largest distance between two of its points.
func (p PlanarRoute) Span() float64 {
	var radius float64
	for _, point := range p.Points {
		radius = math.Max(radius, math.Hypot(point.X, point.Y))
	}
	return 2 * radius
}

// PointAt returns the position after travelling distance meters along the
// route (clamped to its ends).
func (p PlanarRoute) PointAt(distance float64) PlanePoint {
	// i is the first point at or after distance, so
	// Distances[i-1] < distance <= Distances[i] and the segment has positive
	// length (i is clamped for a distance beyond the end).
	i := min(sort.SearchFloat64s(p.Distances, distance), len(p.Points)-1)
	if i == 0 {
		return p.Points[0]
	}

	t := clamp((distance-p.Distances[i-1])/(p.Distances[i]-p.Distances[i-1]), 0, 1)
	return PlanePoint{
		X: p.Points[i-1].X + t*(p.Points[i].X-p.Points[i-1].X),
		Y: p.Points[i-1].Y + t*(p.Points[i].Y-p.Points[i-1].Y),
	}
}

// ChordHeading is the direction, in degrees clockwise from north in [0, 360),
// from the first to the last point of the route.
func (p PlanarRoute) ChordHeading() float64 {
	first, last := p.Points[0], p.Points[len(p.Points)-1]
	return normalizeDegrees(radiansToDegrees(math.Atan2(last.X-first.X, last.Y-first.Y)))
}

// headingHoldRatio is the resultant/total-weight ratio of the tangent window
// below which the route has no clear direction (a turn back, or a loop
// tighter than the window), so the camera keeps its heading instead of
// guessing.
const headingHoldRatio = 0.05

// HeadingAt returns the direction, in degrees clockwise from north in
// [0, 360), the camera should point at distance meters along the route: the
// direction of the Gaussian-weighted sum of the route's tangents around it,
// with standard deviation sigma meters. ok is false when the tangents cancel
// out (a turn back, a tight loop) and there is no clear direction.
func (p PlanarRoute) HeadingAt(distance, sigma float64) (heading float64, ok bool) {
	if len(p.Points) < 2 || sigma <= 0 {
		return 0, false
	}

	lo := sort.SearchFloat64s(p.Distances, distance-3*sigma)
	hi := sort.SearchFloat64s(p.Distances, distance+3*sigma)
	lo = max(lo-1, 0)
	hi = min(hi+1, len(p.Points)-1)

	var sumX, sumY, sumWeight float64
	for i := lo; i < hi; i++ {
		dx, dy := p.Points[i+1].X-p.Points[i].X, p.Points[i+1].Y-p.Points[i].Y
		length := math.Hypot(dx, dy)
		if length < 1e-9 {
			continue
		}

		offset := (p.Distances[i]+p.Distances[i+1])/2 - distance
		weight := length * math.Exp(-offset*offset/(2*sigma*sigma))
		sumX += weight * dx / length
		sumY += weight * dy / length
		sumWeight += weight
	}

	if sumWeight == 0 || math.Hypot(sumX, sumY) < headingHoldRatio*sumWeight {
		return 0, false
	}

	return normalizeDegrees(radiansToDegrees(math.Atan2(sumX, sumY))), true
}

// OverviewView returns the view that frames the whole route: it looks at the
// center of the route's bounding box, tilted OverviewTiltDegrees, pointing
// along heading (the opening and closing pass no rotation: they keep the
// heading of the following phase), from far enough away that every point of
// the route is within half the vertical field of view of the optical axis,
// plus a margin. The distance is never below minDistance, so that the
// overview is always farther than the following distance.
func (p PlanarRoute) OverviewView(heading, minDistance float64, tuning CameraTuning) CameraView {
	minX, maxX, minY, maxY := p.Points[0].X, p.Points[0].X, p.Points[0].Y, p.Points[0].Y
	for _, point := range p.Points {
		minX, maxX = math.Min(minX, point.X), math.Max(maxX, point.X)
		minY, maxY = math.Min(minY, point.Y), math.Max(maxY, point.Y)
	}
	center := PlanePoint{X: (minX + maxX) / 2, Y: (minY + maxY) / 2}

	var radius float64
	for _, point := range p.Points {
		radius = math.Max(radius, math.Hypot(point.X-center.X, point.Y-center.Y))
	}

	halfFOV := degreesToRadians(tuning.OverviewVerticalFOVDegrees) / 2
	framing := tuning.OverviewMargin * radius / math.Tan(halfFOV)

	return CameraView{
		Target:      center,
		Heading:     heading,
		TiltDegrees: tuning.OverviewTiltDegrees,
		Distance:    math.Max(framing, minDistance),
	}
}
