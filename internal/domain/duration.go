package domain

import (
	"sort"
	"time"
)

// Duration returns the elapsed time between the first and the last point
// (FR-020). The second return value reports whether the route had time data
// at all: it is false when at least one point is missing a timestamp, for
// the same reason a partial elevation gain is rejected — a duration
// computed while ignoring some points would misrepresent the real activity
// (FR-021).
func (r Route) Duration() (duration time.Duration, ok bool) {
	if !r.allHaveTime() {
		return 0, false
	}

	first := *r.Points[0].Time
	last := *r.Points[len(r.Points)-1].Time

	return last.Sub(first), true
}

// TimeAt returns the real time elapsed since the first point when at meters
// have been travelled along the route — distances holding, for each point,
// the distance travelled to reach it (as Route.Distances() would; the plan's
// marker distances already come from a route projected this way). The
// route is clamped to its ends, and the value never decreases as at grows
// (009-frame-overlays FR-006). The second return value is false when
// !r.allHaveTime(), the same criterion Duration() uses.
func (r Route) TimeAt(distances []float64, at float64) (time.Duration, bool) {
	if !r.allHaveTime() {
		return 0, false
	}

	i := min(sort.SearchFloat64s(distances, at), len(r.Points)-1)
	if i == 0 {
		return 0, true
	}

	t := clamp((at-distances[i-1])/(distances[i]-distances[i-1]), 0, 1)
	from := r.Points[i-1].Time.Sub(*r.Points[0].Time)
	to := r.Points[i].Time.Sub(*r.Points[0].Time)
	// (1-t)*from + t*to, not from+t*(to-from): the latter is not guaranteed
	// to land on exactly `to` at t=1 in floating point, and the plan's last
	// frame must match Duration() bit for bit (research.md item 7).
	return time.Duration((1-t)*float64(from) + t*float64(to)), true
}

// DistanceAt is TimeAt's exact inverse (014-speed-overlay-block): given the
// real time elapsed since the first point, it returns the distance travelled
// at (meters) along the route — distances holding, for each point, the
// distance travelled to reach it, as TimeAt's own distances parameter does.
// The route is clamped to its ends (an at before the first point or after the
// last one returns that end's distance), and the second return value is false
// under the same !r.allHaveTime() criterion TimeAt uses.
func (r Route) DistanceAt(distances []float64, at time.Duration) (float64, bool) {
	if !r.allHaveTime() {
		return 0, false
	}

	first := *r.Points[0].Time
	i := min(sort.Search(len(r.Points), func(i int) bool {
		return r.Points[i].Time.Sub(first) >= at
	}), len(r.Points)-1)
	if i == 0 {
		return distances[0], true
	}

	from := r.Points[i-1].Time.Sub(first)
	to := r.Points[i].Time.Sub(first)
	t := clamp(float64(at-from)/float64(to-from), 0, 1)
	// (1-t)*a + t*b, not a+t*(b-a): the latter is not guaranteed to land on
	// exactly `b` at t=1 in floating point (the same reason TimeAt's own
	// lerp takes this form).
	return (1-t)*distances[i-1] + t*distances[i], true
}

func (r Route) allHaveTime() bool {
	if len(r.Points) == 0 {
		return false
	}
	for _, p := range r.Points {
		if !p.HasTime() {
			return false
		}
	}
	return true
}
