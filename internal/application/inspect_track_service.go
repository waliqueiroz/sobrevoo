// Package application holds Sobrevoo's service layer. It orchestrates the
// domain's ports and pure functions, but never talks to a file, a network
// socket, an external process, or a terminal directly — those concerns
// belong to the adapters in internal/infra (Constitution Principles I and
// III).
package application

import (
	"io"
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// InspectTrackInput is the input for InspectTrackService.Inspect.
type InspectTrackInput struct {
	// Reader is the track file's content to process.
	Reader io.Reader

	// SimplificationLevel and SmoothingLevel are the levels chosen by the
	// user, already converted from whatever the inbound adapter received
	// (e.g. a CLI flag value) into the domain's Level type. Applying
	// domain.LevelMedium when the user does not specify a level (FR-016) is
	// the inbound adapter's job (e.g. a flag's default value) — by the time
	// input reaches this service, both levels are already whatever should
	// actually be used.
	SimplificationLevel domain.Level
	SmoothingLevel      domain.Level
}

// InspectTrackOutput is the summary produced by InspectTrackService.Inspect
// (FR-025). It is plain data — no io.Writer field, no formatted text — so
// presentation stays the exclusive responsibility of whichever inbound
// adapter calls this service (Constitution Principle III).
type InspectTrackOutput struct {
	Format              domain.Format
	PointCountOriginal  int
	PointCountTreated   int
	TotalDistanceMeters float64
	ElevationGainMeters *float64       // nil when the track had no elevation data (FR-019)
	Duration            *time.Duration // nil when the track had no time data (FR-021)
	BoundingBox         domain.BoundingBox
	Discarded           domain.DiscardStats
}

//go:generate go run go.uber.org/mock/mockgen -destination mock_application/inspect_track_service.go . InspectTrackService

// InspectTrackService reads, treats and summarizes a GPS track (FR-001
// through FR-027). It depends only on ports declared in the domain, so it
// can be reused unchanged by any future entrypoint — a REST adapter, for
// instance — without duplicating any business rule (Constitution Principle
// III).
type InspectTrackService interface {
	Inspect(input InspectTrackInput) (InspectTrackOutput, error)
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

func (s *inspectTrackService) Inspect(input InspectTrackInput) (InspectTrackOutput, error) {
	track, points, discarded, err := cleanTrack(s.parser, s.minPoints, s.maxPlausibleSpeedKmh, input.Reader)
	if err != nil {
		return InspectTrackOutput{}, err
	}

	points = s.simplifier.Simplify(points, input.SimplificationLevel)
	points = s.smoother.Smooth(points, input.SmoothingLevel)

	route := domain.Route{Points: points}

	return s.buildOutput(track, route, discarded), nil
}

func (s *inspectTrackService) buildOutput(track domain.Track, route domain.Route, discarded domain.DiscardStats) InspectTrackOutput {
	output := InspectTrackOutput{
		Format:              track.Format,
		PointCountOriginal:  len(track.Points),
		PointCountTreated:   len(route.Points),
		TotalDistanceMeters: domain.TotalDistance(route.Points),
		BoundingBox:         domain.ComputeBoundingBox(route.Points),
		Discarded:           discarded,
	}

	if gain, ok := domain.ElevationGain(route.Points); ok {
		output.ElevationGainMeters = &gain
	}

	if duration, ok := domain.Duration(route.Points); ok {
		output.Duration = &duration
	}

	return output
}
