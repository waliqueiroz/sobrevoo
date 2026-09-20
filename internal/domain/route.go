package domain

// Route is a sequence of track points: what a track becomes once its points
// are cleaned, and then simplified and smoothed. The statistics of a track
// (length, duration, elevation gain, bounding box) and the checks made over
// it (coverage by geo data) are computed from it.
type Route struct {
	Points []TrackPoint
}
