package application

import (
	"io"
	"sort"
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// GeoDataSummary is one registered source as reported by
// GeoDataService.List, with its live file availability (FR-010).
type GeoDataSummary struct {
	Source domain.GeoDataSource

	// Available is false when the file is no longer found at Source.Path
	// (FR-010).
	Available bool
}

// CoverageStatus is GeoDataService.CheckCoverage's overall verdict for a
// track (SC-003). See CheckCoverageOutput for the exact rule that decides
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

// CheckCoverageOutput is the verdict produced by GeoDataService.CheckCoverage.
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

//go:generate go run go.uber.org/mock/mockgen -destination mock_application/geo_data_service.go . GeoDataService

// GeoDataService manages the local registry of geo data sources — base
// maps and elevation files the user has already downloaded — and checks
// whether a GPS track is covered by them (FR-001 through FR-018). It
// depends only on ports declared in the domain, so it can be reused
// unchanged by any future entrypoint without duplicating any business rule
// (Constitution Principle III).
//
// A single service groups every operation on this one resource (register,
// list, remove, check coverage) — not one interface per use case — the
// same way GroupService groups Create/GetByID/AddUser/... in
// waliqueiroz/mystery-gifter-api.
type GeoDataService interface {
	// Register registers a local map or elevation data file under name,
	// discovering its type, format and geographic area from its content
	// (FR-001 through FR-008).
	Register(name, path string) (domain.GeoDataSource, error)

	// List returns every registered source, each with its live file
	// availability (FR-009, FR-010).
	List() ([]GeoDataSummary, error)

	// Remove deletes the registered source with the given name, without
	// touching the underlying data file on disk (FR-011, FR-012).
	Remove(name string) error

	// CheckCoverage verifies whether the track read from reader is covered
	// by the registered sources (FR-013 through FR-018).
	CheckCoverage(reader io.Reader) (CheckCoverageOutput, error)
}

type geoDataService struct {
	registry    domain.GeoDataRegistry
	inspector   domain.GeoDataInspector
	fileChecker domain.FileChecker
	parser      domain.TrackParser

	// minPoints and maxPlausibleSpeedKmh are the same track-cleaning
	// thresholds InspectTrackService uses, resolved by an outbound
	// configuration adapter and injected here by whoever assembles the
	// service (Constitution Principle VIII).
	minPoints            int
	maxPlausibleSpeedKmh float64
}

// NewGeoDataService creates a GeoDataService backed by the given ports and
// thresholds.
func NewGeoDataService(
	registry domain.GeoDataRegistry,
	inspector domain.GeoDataInspector,
	fileChecker domain.FileChecker,
	parser domain.TrackParser,
	minPoints int,
	maxPlausibleSpeedKmh float64,
) GeoDataService {
	return &geoDataService{
		registry:             registry,
		inspector:            inspector,
		fileChecker:          fileChecker,
		parser:               parser,
		minPoints:            minPoints,
		maxPlausibleSpeedKmh: maxPlausibleSpeedKmh,
	}
}

func (s *geoDataService) Register(name, path string) (domain.GeoDataSource, error) {
	_, found, err := s.registry.FindByName(name)
	if err != nil {
		return domain.GeoDataSource{}, err
	}
	if found {
		return domain.GeoDataSource{}, domain.ErrDataSourceNameAlreadyUsed
	}

	inspected, err := s.inspector.Inspect(path)
	if err != nil {
		return domain.GeoDataSource{}, err
	}

	source := domain.GeoDataSource{
		Name:         name,
		Path:         path,
		Type:         inspected.Type,
		Format:       inspected.Format,
		BoundingBox:  inspected.BoundingBox,
		RegisteredAt: time.Now(),
	}

	if err := s.registry.Save(source); err != nil {
		return domain.GeoDataSource{}, err
	}

	return source, nil
}

func (s *geoDataService) List() ([]GeoDataSummary, error) {
	sources, err := s.registry.List()
	if err != nil {
		return nil, err
	}

	summaries := make([]GeoDataSummary, len(sources))
	for i, source := range sources {
		summaries[i] = GeoDataSummary{
			Source:    source,
			Available: s.fileChecker.Exists(source.Path),
		}
	}

	return summaries, nil
}

func (s *geoDataService) Remove(name string) error {
	_, found, err := s.registry.FindByName(name)
	if err != nil {
		return err
	}
	if !found {
		return domain.ErrDataSourceNotRegistered
	}

	return s.registry.Delete(name)
}

func (s *geoDataService) CheckCoverage(reader io.Reader) (CheckCoverageOutput, error) {
	// The route used for coverage is cleaned but not simplified/smoothed:
	// those two steps are rendering preparation and could shift points,
	// masking a real coverage gap (research.md item 9).
	_, points, _, err := cleanTrack(s.parser, s.minPoints, s.maxPlausibleSpeedKmh, reader)
	if err != nil {
		return CheckCoverageOutput{}, err
	}

	sources, err := s.registry.List()
	if err != nil {
		return CheckCoverageOutput{}, err
	}

	baseMaps, elevations := s.partitionAvailableSources(sources)

	return s.buildCoverageOutput(points, baseMaps, elevations), nil
}

// partitionAvailableSources splits sources into base map and elevation
// candidates, excluding any whose file is no longer found (FR-017).
func (s *geoDataService) partitionAvailableSources(sources []domain.GeoDataSource) (baseMaps, elevations []domain.GeoDataSource) {
	for _, source := range sources {
		if !s.fileChecker.Exists(source.Path) {
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

func (s *geoDataService) buildCoverageOutput(points []domain.TrackPoint, baseMaps, elevations []domain.GeoDataSource) CheckCoverageOutput {
	baseMapUsed := map[string]domain.GeoDataSource{}
	elevationUsed := map[string]domain.GeoDataSource{}

	var segments []UncoveredSegment
	segmentOpen := false

	for _, point := range points {
		baseMap, hasBaseMap := pickCoverageWinner(baseMaps, point.Latitude, point.Longitude)
		elevation, hasElevation := pickCoverageWinner(elevations, point.Latitude, point.Longitude)

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

// pickCoverageWinner returns the candidate covering (lat, lon) that wins
// the determinism rule (FR-016, Clarification — spec.md): the smallest
// BoundingBox.AreaDegrees (most specific); ties broken by the oldest
// RegisteredAt.
func pickCoverageWinner(candidates []domain.GeoDataSource, lat, lon float64) (domain.GeoDataSource, bool) {
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

// sortedSources returns used's values sorted by Name, so CheckCoverageOutput
// is deterministic regardless of map iteration order (SC-005).
func sortedSources(used map[string]domain.GeoDataSource) []domain.GeoDataSource {
	sources := make([]domain.GeoDataSource, 0, len(used))
	for _, source := range used {
		sources = append(sources, source)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Name < sources[j].Name })
	return sources
}
