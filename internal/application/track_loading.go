package application

import (
	"io"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// cleanTrack reads a track via parser and applies the reordering and
// discarding steps shared by InspectTrackService and CheckCoverageService
// (research.md item 9): reorder by time, then discard impossible
// coordinates, consecutive duplicates and implausible jumps. Extracted out
// of inspectTrackService.Execute so the two services never duplicate this
// business rule (Constitution Principle III).
//
// It returns the original parsed domain.Track (InspectTrackService still
// needs its Format and original point count) alongside the cleaned points
// and the discard statistics. Simplification and smoothing are
// deliberately not part of this helper — CheckCoverageService needs the
// cleaned-but-untreated route, not the simplified/smoothed one (research.md
// item 9).
func cleanTrack(parser domain.TrackParser, minPoints int, maxPlausibleSpeedKmh float64, reader io.Reader) (domain.Track, []domain.TrackPoint, domain.DiscardStats, error) {
	track, err := parser.Parse(reader)
	if err != nil {
		return domain.Track{}, nil, domain.DiscardStats{}, err
	}

	if len(track.Points) < minPoints {
		return domain.Track{}, nil, domain.DiscardStats{}, domain.ErrInsufficientPoints
	}

	points := domain.ReorderByTime(track.Points)

	var discarded domain.DiscardStats
	points, discarded.ImpossibleCoordinates = domain.DiscardImpossibleCoordinates(points)
	points, discarded.ConsecutiveDuplicates = domain.DiscardConsecutiveDuplicates(points)
	points, discarded.ImplausibleJumps = domain.DiscardImplausibleJumps(points, maxPlausibleSpeedKmh)

	if len(points) < minPoints {
		return domain.Track{}, nil, domain.DiscardStats{}, domain.ErrInsufficientPointsAfterCleaning
	}

	return track, points, discarded, nil
}
