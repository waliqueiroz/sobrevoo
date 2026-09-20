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

// SummarizeTrack builds a TrackSummary from a track's original data and
// the route it became after treatment (cleaning, and — for
// TrackService specifically — simplification and smoothing),
// together with what was discarded while cleaning it (FR-025).
func SummarizeTrack(track Track, route Route, discarded DiscardStats) TrackSummary {
	summary := TrackSummary{
		Format:              track.Format,
		PointCountOriginal:  len(track.Points),
		PointCountTreated:   len(route.Points),
		TotalDistanceMeters: TotalDistance(route.Points),
		BoundingBox:         ComputeBoundingBox(route.Points),
		Discarded:           discarded,
	}

	if gain, ok := ElevationGain(route.Points); ok {
		summary.ElevationGainMeters = &gain
	}

	if duration, ok := Duration(route.Points); ok {
		summary.Duration = &duration
	}

	return summary
}
