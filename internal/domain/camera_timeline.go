package domain

import (
	"sort"
)

const (
	noTimeDataReason       = "no time data"
	inconsistentTimeReason = "time data is inconsistent"

	// maxUntimedJumpMeters is the largest displacement tolerated between two
	// points that share a timestamp before the timestamps are deemed
	// unusable.
	maxUntimedJumpMeters = 1.0
)

// MarkerTimeline maps progress through the video's following phase to
// distance travelled along the track. Progress is linear in "effective
// time": the track's real time with long stops compressed, or the distance
// itself when no usable timestamps exist.
type MarkerTimeline struct {
	// Cumulative is the distance travelled, in meters, at each point. The
	// small movements inside a long stop (GPS jitter around a standstill) do
	// not count: the marker stands still there.
	Cumulative []float64

	// Effective is the effective time, in seconds, at each point. It never
	// decreases.
	Effective []float64

	Reference      TimeReference
	FallbackReason string
}

// NewMarkerTimeline builds the timeline of a route (the cleaned route of a
// track, before simplification, since simplification hides stops). Timestamps are used
// when every point has one, the total duration is positive and no two points
// sharing a timestamp are more than one meter apart; otherwise the distance
// travelled is used, and FallbackReason says why. Long stops (see
// CameraTuning) are compressed so they take a short, capped time in the
// video.
func NewMarkerTimeline(route Route, tuning CameraTuning) MarkerTimeline {
	cumulative := route.Distances()

	timeline := MarkerTimeline{Cumulative: cumulative}

	if reason := route.unusableTimeReason(cumulative); reason != "" {
		timeline.Reference = TimeReferenceDistance
		timeline.FallbackReason = reason
		timeline.Effective = append([]float64{}, cumulative...)
		return timeline
	}

	timeline.Reference = TimeReferenceClock
	timeline.Effective, timeline.Cumulative = route.compressLongStops(cumulative, tuning)
	return timeline
}

// unusableTimeReason returns why the route's timestamps cannot drive the
// marker, or "" when they can.
func (r Route) unusableTimeReason(cumulative []float64) string {
	points := r.Points
	if len(points) == 0 || !r.allHaveTime() {
		return noTimeDataReason
	}

	if points[len(points)-1].Time.Sub(*points[0].Time) <= 0 {
		return inconsistentTimeReason
	}

	for i := 1; i < len(points); i++ {
		dt := points[i].Time.Sub(*points[i-1].Time)
		if dt <= 0 && cumulative[i]-cumulative[i-1] > maxUntimedJumpMeters {
			return inconsistentTimeReason
		}
	}

	return ""
}

// compressLongStops returns the effective time at each point: real elapsed
// seconds, except that every long stop is shrunk (proportionally across its
// segments) to min(real duration, StopCappedDuration, StopMaxShareOfMovingTime
// of the moving time). It also returns the distance travelled at each point
// with the jitter inside long stops removed, so the marker does not hop when
// a stop's few remaining moments are played.
func (r Route) compressLongStops(cumulative []float64, tuning CameraTuning) (effective, distances []float64) {
	points := r.Points
	segments := len(points) - 1
	real := make([]float64, segments)
	stopped := make([]bool, segments)
	for i := 0; i < segments; i++ {
		real[i] = points[i+1].Time.Sub(*points[i].Time).Seconds()
		distance := cumulative[i+1] - cumulative[i]
		stopped[i] = real[i] > 0 && distance/real[i] < tuning.StopSpeedMetersPerSecond
	}

	type run struct{ from, to int } // segments [from, to)
	var longStops []run
	longStopTime := 0.0
	for i := 0; i < segments; {
		if !stopped[i] {
			i++
			continue
		}
		j, duration := i, 0.0
		for j < segments && stopped[j] {
			duration += real[j]
			j++
		}
		if duration > tuning.StopMinDuration.Seconds() {
			longStops = append(longStops, run{i, j})
			longStopTime += duration
		}
		i = j
	}

	totalReal := 0.0
	for _, dt := range real {
		totalReal += dt
	}
	movingTime := totalReal - longStopTime

	scale := make([]float64, segments)
	for i := range scale {
		scale[i] = 1
	}
	for _, r := range longStops {
		duration := 0.0
		for i := r.from; i < r.to; i++ {
			duration += real[i]
		}
		capped := minFloat(duration, tuning.StopCappedDuration.Seconds(), tuning.StopMaxShareOfMovingTime*movingTime)
		for i := r.from; i < r.to; i++ {
			scale[i] = capped / duration
		}
	}

	effective = make([]float64, len(points))
	distances = make([]float64, len(points))
	for i := 0; i < segments; i++ {
		effective[i+1] = effective[i] + real[i]*scale[i]

		moved := cumulative[i+1] - cumulative[i]
		if scale[i] < 1 {
			moved = 0
		}
		distances[i+1] = distances[i] + moved
	}
	return effective, distances
}

// DistanceAt returns the distance travelled, in meters, when a fraction of
// the following phase in [0, 1] has elapsed. It never decreases as the
// fraction grows.
func (m MarkerTimeline) DistanceAt(fraction float64) float64 {
	last := len(m.Effective) - 1
	if last < 0 {
		return 0
	}

	target := clamp(fraction, 0, 1) * m.Effective[last]
	// i is the first point whose effective time reaches target, so
	// Effective[i-1] < target <= Effective[i] and the span below is positive
	// (target never exceeds the last effective time).
	i := sort.Search(len(m.Effective), func(k int) bool { return m.Effective[k] >= target })
	if i == 0 {
		return m.Cumulative[0]
	}

	t := (target - m.Effective[i-1]) / (m.Effective[i] - m.Effective[i-1])
	return m.Cumulative[i-1] + t*(m.Cumulative[i]-m.Cumulative[i-1])
}

// Total returns the total distance travelled, in meters.
func (m MarkerTimeline) Total() float64 {
	if len(m.Cumulative) == 0 {
		return 0
	}
	return m.Cumulative[len(m.Cumulative)-1]
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minFloat(first float64, rest ...float64) float64 {
	m := first
	for _, v := range rest {
		if v < m {
			m = v
		}
	}
	return m
}
