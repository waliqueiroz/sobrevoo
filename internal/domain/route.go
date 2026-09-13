package domain

// Route is a track after all treatment steps (reordering, discarding,
// simplification, smoothing) have been applied. It is what the final
// summary statistics are computed from.
type Route struct {
	Points []TrackPoint
}
