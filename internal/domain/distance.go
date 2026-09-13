package domain

import "math"

// earthRadiusMeters is the mean Earth radius used by the Haversine formula.
const earthRadiusMeters = 6371000.0

// Haversine returns the great-circle distance, in meters, between two
// track points. The formula is periodic in longitude, so it produces a
// correct (short) distance for pairs of points that straddle the
// antimeridian without any special-casing (FR-017).
func Haversine(a, b TrackPoint) float64 {
	lat1 := degreesToRadians(a.Latitude)
	lat2 := degreesToRadians(b.Latitude)
	deltaLat := degreesToRadians(b.Latitude - a.Latitude)
	deltaLon := degreesToRadians(b.Longitude - a.Longitude)

	sinDeltaLat := math.Sin(deltaLat / 2)
	sinDeltaLon := math.Sin(deltaLon / 2)

	h := sinDeltaLat*sinDeltaLat + math.Cos(lat1)*math.Cos(lat2)*sinDeltaLon*sinDeltaLon
	c := 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))

	return earthRadiusMeters * c
}

// TotalDistance sums the Haversine distance between every pair of
// consecutive points in a route (FR-017).
func TotalDistance(points []TrackPoint) float64 {
	var total float64
	for i := 1; i < len(points); i++ {
		total += Haversine(points[i-1], points[i])
	}
	return total
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}
