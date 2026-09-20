package domain

import "math"

// earthRadiusMeters is the mean Earth radius used by the Haversine formula.
const earthRadiusMeters = 6371000.0

// DistanceTo returns the great-circle distance, in meters, from p to other.
// The Haversine formula is periodic in longitude, so it produces a correct
// (short) distance for pairs of points that straddle the antimeridian without
// any special-casing (FR-017).
func (p TrackPoint) DistanceTo(other TrackPoint) float64 {
	lat1 := degreesToRadians(p.Latitude)
	lat2 := degreesToRadians(other.Latitude)
	deltaLat := degreesToRadians(other.Latitude - p.Latitude)
	deltaLon := degreesToRadians(other.Longitude - p.Longitude)

	sinDeltaLat := math.Sin(deltaLat / 2)
	sinDeltaLon := math.Sin(deltaLon / 2)

	h := sinDeltaLat*sinDeltaLat + math.Cos(lat1)*math.Cos(lat2)*sinDeltaLon*sinDeltaLon
	c := 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))

	return earthRadiusMeters * c
}

// Length sums the distance between every pair of consecutive points of the
// route (FR-017).
func (r Route) Length() float64 {
	var total float64
	for i := 1; i < len(r.Points); i++ {
		total += r.Points[i-1].DistanceTo(r.Points[i])
	}
	return total
}

// Distances returns the distance travelled, in meters, at each point of the
// route: zero at the first point, the running total after that.
func (r Route) Distances() []float64 {
	distances := make([]float64, len(r.Points))
	for i := 1; i < len(r.Points); i++ {
		distances[i] = distances[i-1] + r.Points[i-1].DistanceTo(r.Points[i])
	}
	return distances
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}
