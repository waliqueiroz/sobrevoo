// Package trackparser implements the domain.TrackParser port for the track
// file formats Sobrevoo supports.
package trackparser

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/tkrajina/gpxgo/gpx"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// GPX implements domain.TrackParser for the GPX format (FR-002,
// FR-003, FR-004). It validates the content before delegating the actual
// parsing to github.com/tkrajina/gpxgo, so a clear domain.ErrUnsupportedFormat
// is returned for non-GPX content instead of a raw library error.
type GPX struct{}

// NewGPX creates a GPX parser.
func NewGPX() GPX {
	return GPX{}
}

// Parse reads r fully, confirms its root XML element is "gpx", and delegates
// to gpxgo for the actual parsing.
func (GPX) Parse(r io.Reader) (domain.Track, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return domain.Track{}, fmt.Errorf("reading track content: %w", err)
	}

	if len(data) == 0 {
		return domain.Track{}, domain.ErrEmptyFile
	}

	if !isGPXRoot(data) {
		return domain.Track{}, domain.ErrUnsupportedFormat
	}

	parsed, err := gpx.ParseBytes(data)
	if err != nil {
		return domain.Track{}, fmt.Errorf("parsing GPX content: %w", err)
	}

	return domain.Track{
		Format: domain.FormatGPX,
		Points: extractPoints(parsed),
	}, nil
}

// isGPXRoot reports whether data's first XML element is named "gpx",
// regardless of namespace.
func isGPXRoot(data []byte) bool {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		if start, ok := token.(xml.StartElement); ok {
			return start.Name.Local == "gpx"
		}
	}
}

func extractPoints(parsed *gpx.GPX) []domain.TrackPoint {
	var points []domain.TrackPoint

	for _, track := range parsed.Tracks {
		for _, segment := range track.Segments {
			for _, point := range segment.Points {
				points = append(points, toTrackPoint(point))
			}
		}
	}

	return points
}

func toTrackPoint(point gpx.GPXPoint) domain.TrackPoint {
	trackPoint := domain.TrackPoint{
		Latitude:  point.Latitude,
		Longitude: point.Longitude,
	}

	if point.Elevation.NotNull() {
		elevation := point.Elevation.Value()
		trackPoint.Elevation = &elevation
	}

	if !point.Timestamp.IsZero() {
		timestamp := point.Timestamp
		trackPoint.Time = &timestamp
	}

	return trackPoint
}
