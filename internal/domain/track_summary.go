package domain

import "time"

// TrackSummary is what TrackService.Inspect reports about a GPS
// track (FR-025). It is plain data — no io.Writer field, no formatted text
// — so presentation stays the exclusive responsibility of whichever
// inbound adapter calls the service (Constitution Principle III).
type TrackSummary struct {
	Format              Format
	PointCountOriginal  int
	PointCountTreated   int
	TotalDistanceMeters float64
	ElevationGainMeters *float64       // nil when the track had no elevation data (FR-019)
	Duration            *time.Duration // nil when the track had no time data (FR-021)
	BoundingBox         BoundingBox
	Discarded           DiscardStats
}

// NewTrackSummary builds a TrackSummary from a track's original data and
// the route it became after treatment (cleaning, and — for
// TrackService specifically — simplification and smoothing),
// together with what was discarded while cleaning it (FR-025).
func NewTrackSummary(track Track, route Route, discarded DiscardStats) TrackSummary {
	summary := TrackSummary{
		Format:              track.Format,
		PointCountOriginal:  len(track.Points),
		PointCountTreated:   len(route.Points),
		TotalDistanceMeters: route.Length(),
		BoundingBox:         route.BoundingBox(),
		Discarded:           discarded,
	}

	if gain, ok := route.ElevationGain(); ok {
		summary.ElevationGainMeters = &gain
	}

	if duration, ok := route.Duration(); ok {
		summary.Duration = &duration
	}

	return summary
}
