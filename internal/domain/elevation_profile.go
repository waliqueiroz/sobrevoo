package domain

import "sort"

// ElevationProfile is a route's elevation, and the elevation gained so far,
// at every point along it — pre-computed once so it can be queried by
// distance many times (once per frame of a camera plan) without re-walking
// the route's points on every call (009-frame-overlays research.md item 8).
type ElevationProfile struct {
	// Distances is the distance travelled, in meters, at each point — the
	// same array a PlanarRoute of the same route already holds.
	Distances []float64

	Elevations []float64

	// Gains is the elevation gained from the start up to each point: the
	// running sum of every positive elevation delta between consecutive
	// points — the same rule Route.ElevationGain sums into a single total,
	// kept here at every partial step instead. It never decreases.
	Gains []float64
}

// ElevationProfile pre-computes the route's elevation profile against
// distances — the distance travelled at each point, as Route.Distances()
// would give (009-frame-overlays FR-006). The second return value is false
// when !r.allHaveElevation(), the same criterion ElevationGain() uses.
func (r Route) ElevationProfile(distances []float64) (ElevationProfile, bool) {
	if !r.allHaveElevation() {
		return ElevationProfile{}, false
	}

	elevations := make([]float64, len(r.Points))
	gains := make([]float64, len(r.Points))
	for i, p := range r.Points {
		elevations[i] = *p.Elevation
		if i > 0 {
			gains[i] = gains[i-1]
			if delta := elevations[i] - elevations[i-1]; delta > 0 {
				gains[i] += delta
			}
		}
	}

	return ElevationProfile{Distances: distances, Elevations: elevations, Gains: gains}, true
}

// At returns the elevation and the elevation gained so far at distance
// meters along the route (clamped to its ends), interpolating linearly
// between the two points around it — the same bracket-and-lerp technique
// PlanarRoute.PointAt uses. Evaluated exactly at the last point's distance,
// it returns Gains' last value with no interpolation error, which is what
// guarantees it matches Route.ElevationGain() bit for bit at the end of a
// flight (research.md item 7).
func (e ElevationProfile) At(distance float64) (elevation, gain float64) {
	i := min(sort.SearchFloat64s(e.Distances, distance), len(e.Elevations)-1)
	if i == 0 {
		return e.Elevations[0], e.Gains[0]
	}

	// (1-t)*a + t*b, not a+t*(b-a): the latter is not guaranteed to land on
	// exactly `b` at t=1 in floating point, and the plan's last frame must
	// match Route.ElevationGain() bit for bit (research.md item 7).
	t := clamp((distance-e.Distances[i-1])/(e.Distances[i]-e.Distances[i-1]), 0, 1)
	elevation = (1-t)*e.Elevations[i-1] + t*e.Elevations[i]
	gain = (1-t)*e.Gains[i-1] + t*e.Gains[i]
	return elevation, gain
}
