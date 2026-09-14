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

//go:generate go run go.uber.org/mock/mockgen -destination mock_application/inspect_track_service.go . InspectTrackService

// InspectTrackService reads, treats and summarizes a GPS track (FR-001
// through FR-027). It depends only on ports declared in the domain, so it
// can be reused unchanged by any future entrypoint — a REST adapter, for
// instance — without duplicating any business rule (Constitution Principle
// III). Its method only orchestrates ports and domain
// functions/constructors — the business rules themselves (what "cleaning"
// a track means, how a summary is built) live in internal/domain
// (cleaning.go, track_summary.go), the same way GeoDataService delegates
// to domain.NewGeoDataSource/domain.ComputeCoverage.
type InspectTrackService interface {
	Inspect(reader io.Reader, simplificationLevel, smoothingLevel domain.Level) (domain.TrackSummary, error)
}

type inspectTrackService struct {
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

// NewInspectTrackService creates an InspectTrackService backed by the given
// ports and thresholds.
func NewInspectTrackService(
	parser domain.TrackParser,
	simplifier domain.Simplifier,
	smoother domain.Smoother,
	minPoints int,
	maxPlausibleSpeedKmh float64,
) InspectTrackService {
	return &inspectTrackService{
		parser:               parser,
		simplifier:           simplifier,
		smoother:             smoother,
		minPoints:            minPoints,
		maxPlausibleSpeedKmh: maxPlausibleSpeedKmh,
	}
}

func (s *inspectTrackService) Inspect(reader io.Reader, simplificationLevel, smoothingLevel domain.Level) (domain.TrackSummary, error) {
	track, err := s.parser.Parse(reader)
	if err != nil {
		return domain.TrackSummary{}, err
	}

	points, discarded, err := domain.CleanTrack(track.Points, s.minPoints, s.maxPlausibleSpeedKmh)
	if err != nil {
		return domain.TrackSummary{}, err
	}

	points = s.simplifier.Simplify(points, simplificationLevel)
	points = s.smoother.Smooth(points, smoothingLevel)

	route := domain.Route{Points: points}

	return domain.SummarizeTrack(track, route, discarded), nil
}
