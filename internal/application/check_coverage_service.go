package application

import (
	"io"
	"sort"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// CheckCoverageInput is the input for CheckCoverageService.Execute.
type CheckCoverageInput struct {
	// Reader is the track file's content to check, in the same format
	// InspectTrackInput accepts.
	Reader io.Reader
}

// CoverageStatus is CheckCoverageService's overall verdict for a track
// (SC-003). See CheckCoverageOutput for the exact rule that decides
// between the three values.
type CoverageStatus string

const (
	CoverageStatusFull    CoverageStatus = "full"
	CoverageStatusPartial CoverageStatus = "partial"
	CoverageStatusNone    CoverageStatus = "none"
)

// MissingDataType names what kind of geo data is missing for an
// UncoveredSegment (FR-015).
type MissingDataType string

const (
	MissingBaseMap   MissingDataType = "base map"
	MissingElevation MissingDataType = "elevation"
	MissingBoth      MissingDataType = "base map and elevation"
)

// UncoveredSegment is one contiguous run of route points not fully covered
// (FR-015, Clarification — spec.md).
type UncoveredSegment struct {
	StartLatitude, StartLongitude float64
	EndLatitude, EndLongitude     float64
	Missing                       MissingDataType
}

// CheckCoverageOutput is the verdict produced by CheckCoverageService.Execute.
//
// Status decision rule (resolves the ambiguity flagged by /speckit-analyze —
// see data-model.md):
//   - CoverageStatusFull: every route point is fully covered (UncoveredSegments is empty).
//   - CoverageStatusNone: no route point has coverage of either type anywhere —
//     equivalently, BaseMapSourcesUsed and ElevationSourcesUsed are both empty.
//   - CoverageStatusPartial: everything else — some real coverage exists
//     (at least one fully covered point, or at least one of the two source
//     sets is non-empty), just not for the whole route.
type CheckCoverageOutput struct {
	Status               CoverageStatus
	UncoveredSegments    []UncoveredSegment
	BaseMapSourcesUsed   []domain.GeoDataSource
	ElevationSourcesUsed []domain.GeoDataSource
}

//go:generate go run go.uber.org/mock/mockgen -destination mock_application/check_coverage_service.go . CheckCoverageService

// CheckCoverageService verifies whether a GPS track is covered by the
// registered geo data sources (FR-013 through FR-018). It depends only on
// ports declared in the domain, so it can be reused unchanged by any
// future entrypoint without duplicating any business rule (Constitution
// Principle III).
type CheckCoverageService interface {
	Execute(input CheckCoverageInput) (CheckCoverageOutput, error)
}

type checkCoverageService struct {
	parser      domain.TrackParser
	registry    domain.GeoDataRegistry
	fileChecker domain.FileChecker

	minPoints            int
	maxPlausibleSpeedKmh float64
}

// NewCheckCoverageService creates a CheckCoverageService backed by the
// given ports and thresholds (the same ones used by InspectTrackService,
// so both services clean a track identically — research.md item 9).
func NewCheckCoverageService(
	parser domain.TrackParser,
	registry domain.GeoDataRegistry,
	fileChecker domain.FileChecker,
	minPoints int,
	maxPlausibleSpeedKmh float64,
) CheckCoverageService {
	return &checkCoverageService{
		parser:               parser,
		registry:             registry,
		fileChecker:          fileChecker,
		minPoints:            minPoints,
		maxPlausibleSpeedKmh: maxPlausibleSpeedKmh,
	}
}

func (s *checkCoverageService) Execute(input CheckCoverageInput) (CheckCoverageOutput, error) {
	// The route used for coverage is cleaned but not simplified/smoothed:
	// those two steps are rendering preparation and could shift points,
	// masking a real coverage gap (research.md item 9).
	_, points, _, err := cleanTrack(s.parser, s.minPoints, s.maxPlausibleSpeedKmh, input.Reader)
	if err != nil {
		return CheckCoverageOutput{}, err
	}

	sources, err := s.registry.List()
	if err != nil {
		return CheckCoverageOutput{}, err
	}

	baseMaps, elevations := partitionAvailableSources(sources, s.fileChecker)

	return buildCoverageOutput(points, baseMaps, elevations), nil
}

// partitionAvailableSources splits sources into base map and elevation
// candidates, excluding any whose file is no longer found (FR-017).
func partitionAvailableSources(sources []domain.GeoDataSource, fileChecker domain.FileChecker) (baseMaps, elevations []domain.GeoDataSource) {
	for _, source := range sources {
		if !fileChecker.Exists(source.Path) {
			continue
		}
		switch source.Type {
		case domain.DataTypeBaseMap:
			baseMaps = append(baseMaps, source)
		case domain.DataTypeElevation:
			elevations = append(elevations, source)
		}
	}
	return baseMaps, elevations
}

func buildCoverageOutput(points []domain.TrackPoint, baseMaps, elevations []domain.GeoDataSource) CheckCoverageOutput {
	baseMapUsed := map[string]domain.GeoDataSource{}
	elevationUsed := map[string]domain.GeoDataSource{}

	var segments []UncoveredSegment
	segmentOpen := false

	for _, point := range points {
		baseMap, hasBaseMap := pickWinner(baseMaps, point.Latitude, point.Longitude)
		elevation, hasElevation := pickWinner(elevations, point.Latitude, point.Longitude)

		if hasBaseMap {
			baseMapUsed[baseMap.Name] = baseMap
		}
		if hasElevation {
			elevationUsed[elevation.Name] = elevation
		}

		missing, fullyCovered := missingDataType(hasBaseMap, hasElevation)
		if fullyCovered {
			segmentOpen = false
			continue
		}

		if segmentOpen && segments[len(segments)-1].Missing == missing {
			segments[len(segments)-1].EndLatitude = point.Latitude
			segments[len(segments)-1].EndLongitude = point.Longitude
			continue
		}

		segments = append(segments, UncoveredSegment{
			StartLatitude:  point.Latitude,
			StartLongitude: point.Longitude,
			EndLatitude:    point.Latitude,
			EndLongitude:   point.Longitude,
			Missing:        missing,
		})
		segmentOpen = true
	}

	baseMapSourcesUsed := sortedSources(baseMapUsed)
	elevationSourcesUsed := sortedSources(elevationUsed)

	status := CoverageStatusPartial
	switch {
	case len(segments) == 0:
		status = CoverageStatusFull
	case len(baseMapSourcesUsed) == 0 && len(elevationSourcesUsed) == 0:
		status = CoverageStatusNone
	}

	return CheckCoverageOutput{
		Status:               status,
		UncoveredSegments:    segments,
		BaseMapSourcesUsed:   baseMapSourcesUsed,
		ElevationSourcesUsed: elevationSourcesUsed,
	}
}

// missingDataType reports what is missing for a point, given whether a
// base map and an elevation source were found to cover it, and whether
// that makes the point fully covered.
func missingDataType(hasBaseMap, hasElevation bool) (missing MissingDataType, fullyCovered bool) {
	switch {
	case hasBaseMap && hasElevation:
		return "", true
	case hasBaseMap:
		return MissingElevation, false
	case hasElevation:
		return MissingBaseMap, false
	default:
		return MissingBoth, false
	}
}

// pickWinner returns the candidate covering (lat, lon) that wins the
// determinism rule (FR-016, Clarification — spec.md): the smallest
// BoundingBox.AreaDegrees (most specific); ties broken by the oldest
// RegisteredAt.
func pickWinner(candidates []domain.GeoDataSource, lat, lon float64) (domain.GeoDataSource, bool) {
	var winner domain.GeoDataSource
	found := false

	for _, candidate := range candidates {
		if !candidate.BoundingBox.Contains(lat, lon) {
			continue
		}
		if !found || isMoreSpecific(candidate, winner) {
			winner = candidate
			found = true
		}
	}

	return winner, found
}

// isMoreSpecific reports whether a should win over b under FR-016's
// desempate rule.
func isMoreSpecific(a, b domain.GeoDataSource) bool {
	areaA, areaB := a.BoundingBox.AreaDegrees(), b.BoundingBox.AreaDegrees()
	if areaA != areaB {
		return areaA < areaB
	}
	return a.RegisteredAt.Before(b.RegisteredAt)
}

// sortedSources returns used's values sorted by Name, so
// CheckCoverageOutput is deterministic regardless of map iteration order
// (SC-005).
func sortedSources(used map[string]domain.GeoDataSource) []domain.GeoDataSource {
	sources := make([]domain.GeoDataSource, 0, len(used))
	for _, source := range used {
		sources = append(sources, source)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Name < sources[j].Name })
	return sources
}
