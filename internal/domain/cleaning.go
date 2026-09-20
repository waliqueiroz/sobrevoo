package domain

import "sort"

// ReorderByTime sorts points chronologically by Time, but only when every
// point in the track carries one. When some points have a timestamp and
// others do not, the track is left unchanged: a partial ordering could
// interleave timed and untimed points arbitrarily, producing a route with
// no real physical meaning (FR-027, research.md item 8).
func ReorderByTime(points []TrackPoint) []TrackPoint {
	if !allHaveTime(points) {
		return points
	}

	sorted := make([]TrackPoint, len(points))
	copy(sorted, points)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Time.Before(*sorted[j].Time)
	})

	return sorted
}

// DiscardImpossibleCoordinates removes points whose latitude or longitude
// falls outside the geographically valid range (FR-008).
func DiscardImpossibleCoordinates(points []TrackPoint) (kept []TrackPoint, discarded int) {
	for _, p := range points {
		if hasPossibleCoordinate(p) {
			kept = append(kept, p)
			continue
		}
		discarded++
	}
	return kept, discarded
}

func hasPossibleCoordinate(p TrackPoint) bool {
	return p.Latitude >= -90 && p.Latitude <= 90 && p.Longitude >= -180 && p.Longitude <= 180
}

// DiscardConsecutiveDuplicates removes a point when its coordinate is
// identical to the previous point kept so far (FR-009).
func DiscardConsecutiveDuplicates(points []TrackPoint) (kept []TrackPoint, discarded int) {
	for _, p := range points {
		if len(kept) > 0 && isSameCoordinate(kept[len(kept)-1], p) {
			discarded++
			continue
		}
		kept = append(kept, p)
	}
	return kept, discarded
}

func isSameCoordinate(a, b TrackPoint) bool {
	return a.Latitude == b.Latitude && a.Longitude == b.Longitude
}

// DiscardImplausibleJumps removes a point when the speed implied between it
// and the previous point kept so far exceeds maxPlausibleSpeedKmh. A jump is
// only evaluated when both points carry a timestamp; points missing time are
// always kept by this function (FR-010).
func DiscardImplausibleJumps(points []TrackPoint, maxPlausibleSpeedKmh float64) (kept []TrackPoint, discarded int) {
	for _, p := range points {
		if len(kept) > 0 && isImplausibleJump(kept[len(kept)-1], p, maxPlausibleSpeedKmh) {
			discarded++
			continue
		}
		kept = append(kept, p)
	}
	return kept, discarded
}

func isImplausibleJump(prev, curr TrackPoint, maxPlausibleSpeedKmh float64) bool {
	if !prev.HasTime() || !curr.HasTime() {
		return false
	}

	distanceKm := Haversine(prev, curr) / 1000
	elapsedHours := curr.Time.Sub(*prev.Time).Hours()

	if elapsedHours <= 0 {
		// Zero or negative elapsed time with any real distance implies an
		// infinite speed, which is always implausible.
		return distanceKm > 0
	}

	return distanceKm/elapsedHours > maxPlausibleSpeedKmh
}

// CleanTrack turns a track's raw parsed points into a trustworthy route,
// composing the functions above in the one order that makes sense (FR-006
// through FR-010, FR-027): reorder by time, then discard impossible
// coordinates, consecutive duplicates and implausible jumps. The minimum
// point count is checked both before and after, as two distinguishable
// error cases — shared by every use case that needs a cleaned route from a
// parsed track (TrackService, GeoDataService.CheckCoverage), so
// this composition itself, not just its parts, lives here instead of being
// duplicated or reinvented by each service (Constitution Principle III).
func CleanTrack(points []TrackPoint, minPoints int, maxPlausibleSpeedKmh float64) ([]TrackPoint, DiscardStats, error) {
	if len(points) < minPoints {
		return nil, DiscardStats{}, ErrInsufficientPoints
	}

	points = ReorderByTime(points)

	var discarded DiscardStats
	points, discarded.ImpossibleCoordinates = DiscardImpossibleCoordinates(points)
	points, discarded.ConsecutiveDuplicates = DiscardConsecutiveDuplicates(points)
	points, discarded.ImplausibleJumps = DiscardImplausibleJumps(points, maxPlausibleSpeedKmh)

	if len(points) < minPoints {
		return nil, DiscardStats{}, ErrInsufficientPointsAfterCleaning
	}

	return points, discarded, nil
}
