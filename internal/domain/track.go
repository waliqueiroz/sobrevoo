package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/track_parser.go -package mockdomain . TrackParser

import "io"

// TrackParser reads a track file's content and produces a Track. Concrete
// implementations (one per supported format, plus any format-detection
// glue) live in internal/infra/outbound/trackparser.
type TrackParser interface {
	Parse(r io.Reader) (Track, error)
}

// Format identifies the track file format a Track was parsed from.
type Format string

const (
	FormatGPX Format = "GPX"
)

// Track is a raw track, exactly as it came from the source file, before any
// treatment (reordering, discarding, simplification, smoothing).
type Track struct {
	Format Format
	Points []TrackPoint
}

// CleanedTrack is a Track after parsing and cleaning (reordering by time and
// discarding invalid, duplicate and implausible points), but before any
// simplification or smoothing. Consumers that must not have points shifted
// (such as geo data coverage checks) use it as is.
type CleanedTrack struct {
	Track     Track
	Points    []TrackPoint
	Discarded DiscardStats
}

// TreatedTrack is a CleanedTrack after simplification and smoothing: the
// route that rendering-oriented consumers (summaries, camera planning) work
// with. It keeps the cleaned points too, because simplification discards the
// points that reveal how the activity unfolded in time (a long stop, for
// instance, collapses into a single straight segment).
type TreatedTrack struct {
	Track         Track
	CleanedPoints []TrackPoint
	Route         Route
	Discarded     DiscardStats
}
