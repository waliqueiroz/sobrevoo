package domain

import "math"

// BoundingBox is the geographic area covered by a set of points (FR-022,
// FR-023).
type BoundingBox struct {
	MinLatitude  float64
	MaxLatitude  float64
	MinLongitude float64
	MaxLongitude float64

	// CrossesAntimeridian is true when the route crosses the 180th
	// meridian. In that case, the occupied area is the one going from
	// MaxLongitude to MinLongitude "the outside way" (through ±180°), not
	// the direct span between the two values.
	CrossesAntimeridian bool
}

// ComputeBoundingBox returns the geographic area covered by points. Latitude
// never needs special handling (it does not wrap around). Longitude is
// unwrapped by walking the points in order and accumulating each
// consecutive delta, so a route that crosses the antimeridian produces a
// short, correct occupied area instead of one spanning nearly the whole
// planet (FR-024) — no assumption about hemisphere or region is made
// (research.md item 6).
func ComputeBoundingBox(points []TrackPoint) BoundingBox {
	if len(points) == 0 {
		return BoundingBox{}
	}

	minLat, maxLat := points[0].Latitude, points[0].Latitude
	unwrapped := make([]float64, len(points))
	unwrapped[0] = points[0].Longitude

	for i := 1; i < len(points); i++ {
		lat := points[i].Latitude
		minLat = math.Min(minLat, lat)
		maxLat = math.Max(maxLat, lat)

		delta := points[i].Longitude - points[i-1].Longitude
		switch {
		case delta > 180:
			delta -= 360
		case delta < -180:
			delta += 360
		}
		unwrapped[i] = unwrapped[i-1] + delta
	}

	minLon, maxLon := unwrapped[0], unwrapped[0]
	for _, lon := range unwrapped {
		minLon = math.Min(minLon, lon)
		maxLon = math.Max(maxLon, lon)
	}

	return BoundingBox{
		MinLatitude:         minLat,
		MaxLatitude:         maxLat,
		MinLongitude:        normalizeLongitude(minLon),
		MaxLongitude:        normalizeLongitude(maxLon),
		CrossesAntimeridian: minLon < -180 || maxLon > 180,
	}
}

// normalizeLongitude brings a longitude value (potentially outside
// [-180, 180] after unwrapping) back into that range.
func normalizeLongitude(degrees float64) float64 {
	wrapped := math.Mod(degrees+180, 360)
	if wrapped < 0 {
		wrapped += 360
	}
	return wrapped - 180
}
