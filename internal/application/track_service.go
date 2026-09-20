// Package application holds Sobrevoo's service layer. It orchestrates the
// domain's ports and pure functions, but never talks to a file, a network
// socket, an external process, or a terminal directly — those concerns
// belong to the adapters in internal/infra (Constitution Principles I and
// III).
package application

import (
	"io"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -destination mockapplication/track_service.go -package mockapplication . TrackService

// TrackService reads and treats GPS tracks (FR-001 through FR-027 of the
// first stage). It is the single place that knows how a raw track becomes a
// clean or a treated one, so every other service that needs a track
// (GeoDataService, CameraPlanService) depends on it instead of repeating the
// same parse/clean/simplify/smooth sequence.
//
// One service groups every operation on this resource — Clean, Treat and
// Inspect — and each method only orchestrates ports and domain
// functions/constructors: the business rules themselves (what "cleaning" a
// track means, how a summary is built) live in internal/domain
// (cleaning.go, track_summary.go).
type TrackService interface {
	// Clean parses the track read from reader and cleans it (reordering,
	// discarding invalid points), without simplifying or smoothing it.
	Clean(reader io.Reader) (domain.CleanedTrack, error)

	// Treat cleans the track and then simplifies and smooths it with the
	// given levels.
	Treat(reader io.Reader, simplificationLevel, smoothingLevel domain.Level) (domain.TreatedTrack, error)

	// Inspect treats the track and summarizes it.
	Inspect(reader io.Reader, simplificationLevel, smoothingLevel domain.Level) (domain.TrackSummary, error)
}

type trackService struct {
	parser     domain.TrackParser
	simplifier domain.Simplifier
	smoother   domain.Smoother

	// minPoints and maxPlausibleSpeedKmh are internal thresholds resolved by
	// an outbound configuration adapter and injected here by whoever
	// assembles the service (Constitution Principle VIII) — the service
	// itself never reads configuration directly.
	minPoints            int
	maxPlausibleSpeedKmh float64
}

// NewTrackService creates a TrackService backed by the given ports and
// thresholds.
func NewTrackService(
	parser domain.TrackParser,
	simplifier domain.Simplifier,
	smoother domain.Smoother,
	minPoints int,
	maxPlausibleSpeedKmh float64,
) TrackService {
	return &trackService{
		parser:               parser,
		simplifier:           simplifier,
		smoother:             smoother,
		minPoints:            minPoints,
		maxPlausibleSpeedKmh: maxPlausibleSpeedKmh,
	}
}

func (s *trackService) Clean(reader io.Reader) (domain.CleanedTrack, error) {
	track, err := s.parser.Parse(reader)
	if err != nil {
		return domain.CleanedTrack{}, err
	}

	points, discarded, err := domain.CleanTrack(track.Points, s.minPoints, s.maxPlausibleSpeedKmh)
	if err != nil {
		return domain.CleanedTrack{}, err
	}

	return domain.CleanedTrack{Track: track, Points: points, Discarded: discarded}, nil
}

func (s *trackService) Treat(reader io.Reader, simplificationLevel, smoothingLevel domain.Level) (domain.TreatedTrack, error) {
	cleaned, err := s.Clean(reader)
	if err != nil {
		return domain.TreatedTrack{}, err
	}

	points := s.simplifier.Simplify(cleaned.Points, simplificationLevel)
	points = s.smoother.Smooth(points, smoothingLevel)

	return domain.TreatedTrack{
		Track:         cleaned.Track,
		CleanedPoints: cleaned.Points,
		Route:         domain.Route{Points: points},
		Discarded:     cleaned.Discarded,
	}, nil
}

func (s *trackService) Inspect(reader io.Reader, simplificationLevel, smoothingLevel domain.Level) (domain.TrackSummary, error) {
	treated, err := s.Treat(reader, simplificationLevel, smoothingLevel)
	if err != nil {
		return domain.TrackSummary{}, err
	}

	return domain.SummarizeTrack(treated.Track, treated.Route, treated.Discarded), nil
}
