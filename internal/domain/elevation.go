package domain

// ElevationGain sums every positive elevation delta between consecutive
// points (FR-018). The second return value reports whether the route had
// elevation data at all: it is false when at least one point is missing
// altitude, since a gain computed from a partial set of points would
// misrepresent the real ascent (FR-019).
func ElevationGain(points []TrackPoint) (gain float64, ok bool) {
	if !allHaveElevation(points) {
		return 0, false
	}

	for i := 1; i < len(points); i++ {
		delta := *points[i].Elevation - *points[i-1].Elevation
		if delta > 0 {
			gain += delta
		}
	}

	return gain, true
}

func allHaveElevation(points []TrackPoint) bool {
	if len(points) == 0 {
		return false
	}
	for _, p := range points {
		if !p.HasElevation() {
			return false
		}
	}
	return true
}
