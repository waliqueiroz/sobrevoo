package domain

import "sort"

// ReorderByTime returns the route with its points sorted chronologically by
// Time, but only when every point carries one. When some points have a
// timestamp and others do not, the route is returned unchanged: a partial
// ordering could interleave timed and untimed points arbitrarily, producing a
// route with no real physical meaning (FR-027, research.md item 8).
func (r Route) ReorderByTime() Route {
	if !r.allHaveTime() {
		return r
	}

	sorted := make([]TrackPoint, len(r.Points))
	copy(sorted, r.Points)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Time.Before(*sorted[j].Time)
	})

	return Route{Points: sorted}
}

// DiscardImpossibleCoordinates removes points whose latitude or longitude
// falls outside the geographically valid range (FR-008).
func (r Route) DiscardImpossibleCoordinates() (kept Route, discarded int) {
	for _, p := range r.Points {
		if p.hasPossibleCoordinate() {
			kept.Points = append(kept.Points, p)
			continue
		}
		discarded++
	}
	return kept, discarded
}

func (p TrackPoint) hasPossibleCoordinate() bool {
	return p.Latitude >= -90 && p.Latitude <= 90 && p.Longitude >= -180 && p.Longitude <= 180
}

// DiscardConsecutiveDuplicates removes a point when its coordinate is
// identical to the previous point kept so far (FR-009).
func (r Route) DiscardConsecutiveDuplicates() (kept Route, discarded int) {
	for _, p := range r.Points {
		if len(kept.Points) > 0 && kept.Points[len(kept.Points)-1].hasSameCoordinateAs(p) {
			discarded++
			continue
		}
		kept.Points = append(kept.Points, p)
	}
	return kept, discarded
}

func (p TrackPoint) hasSameCoordinateAs(other TrackPoint) bool {
	return p.Latitude == other.Latitude && p.Longitude == other.Longitude
}

// DiscardImplausibleJumps removes a point when the speed implied between it
// and the previous point kept so far exceeds maxPlausibleSpeedKmh. A jump is
// only evaluated when both points carry a timestamp; points missing time are
// always kept by this method (FR-010).
func (r Route) DiscardImplausibleJumps(maxPlausibleSpeedKmh float64) (kept Route, discarded int) {
	for _, p := range r.Points {
		if len(kept.Points) > 0 && kept.Points[len(kept.Points)-1].isImplausibleJumpTo(p, maxPlausibleSpeedKmh) {
			discarded++
			continue
		}
		kept.Points = append(kept.Points, p)
	}
	return kept, discarded
}

// isImplausibleJumpTo reports whether getting from p to next takes an
// implausible speed.
func (p TrackPoint) isImplausibleJumpTo(next TrackPoint, maxPlausibleSpeedKmh float64) bool {
	if !p.HasTime() || !next.HasTime() {
		return false
	}

	distanceKm := p.DistanceTo(next) / 1000
	elapsedHours := next.Time.Sub(*p.Time).Hours()

	if elapsedHours <= 0 {
		// Zero or negative elapsed time with any real distance implies an
		// infinite speed, which is always implausible.
		return distanceKm > 0
	}

	return distanceKm/elapsedHours > maxPlausibleSpeedKmh
}

// Clean turns a track's raw parsed points into a trustworthy route,
// composing the Route methods above in the one order that makes sense (FR-006
// through FR-010, FR-027): reorder by time, then discard impossible
// coordinates, consecutive duplicates and implausible jumps. The minimum
// point count is checked both before and after, as two distinguishable
// error cases — shared by every use case that needs a cleaned route from a
// parsed track (TrackService, GeoDataService.CheckCoverage), so this
// composition itself, not just its parts, lives here instead of being
// duplicated or reinvented by each service (Constitution Principle III).
func (t Track) Clean(minPoints int, maxPlausibleSpeedKmh float64) (CleanedTrack, error) {
	if len(t.Points) < minPoints {
		return CleanedTrack{}, ErrInsufficientPoints
	}

	route := Route{Points: t.Points}.ReorderByTime()

	var discarded DiscardStats
	route, discarded.ImpossibleCoordinates = route.DiscardImpossibleCoordinates()
	route, discarded.ConsecutiveDuplicates = route.DiscardConsecutiveDuplicates()
	route, discarded.ImplausibleJumps = route.DiscardImplausibleJumps(maxPlausibleSpeedKmh)

	if len(route.Points) < minPoints {
		return CleanedTrack{}, ErrInsufficientPointsAfterCleaning
	}

	return CleanedTrack{Track: t, Route: route, Discarded: discarded}, nil
}
