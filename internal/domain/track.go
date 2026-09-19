package domain

//go:generate go run go.uber.org/mock/mockgen -destination mock_domain/track_parser.go . TrackParser

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
