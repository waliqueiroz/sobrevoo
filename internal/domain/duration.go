package domain

import "time"

// Duration returns the elapsed time between the first and the last point
// (FR-020). The second return value reports whether the route had time data
// at all: it is false when at least one point is missing a timestamp, for
// the same reason a partial elevation gain is rejected — a duration
// computed while ignoring some points would misrepresent the real activity
// (FR-021).
func Duration(points []TrackPoint) (duration time.Duration, ok bool) {
	if !allHaveTime(points) {
		return 0, false
	}

	first := *points[0].Time
	last := *points[len(points)-1].Time

	return last.Sub(first), true
}

func allHaveTime(points []TrackPoint) bool {
	if len(points) == 0 {
		return false
	}
	for _, p := range points {
		if !p.HasTime() {
			return false
		}
	}
	return true
}
