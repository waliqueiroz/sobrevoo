package domain

// ElevationGain sums every positive elevation delta between consecutive
// points (FR-018). The second return value reports whether the route had
// elevation data at all: it is false when at least one point is missing
// altitude, since a gain computed from a partial set of points would
// misrepresent the real ascent (FR-019).
func (r Route) ElevationGain() (gain float64, ok bool) {
	if !r.allHaveElevation() {
		return 0, false
	}

	for i := 1; i < len(r.Points); i++ {
		delta := *r.Points[i].Elevation - *r.Points[i-1].Elevation
		if delta > 0 {
			gain += delta
		}
	}

	return gain, true
}

func (r Route) allHaveElevation() bool {
	if len(r.Points) == 0 {
		return false
	}
	for _, p := range r.Points {
		if !p.HasElevation() {
			return false
		}
	}
	return true
}
