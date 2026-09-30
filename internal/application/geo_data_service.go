package application

import (
	"fmt"
	"io"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -destination mockapplication/geo_data_service.go -package mockapplication . GeoDataService

// GeoDataService manages the local repository of geo data sources — base
// maps and elevation files the user has already downloaded — and checks
// whether a GPS track is covered by them (FR-001 through FR-018). It
// depends only on ports declared in the domain, so it can be reused
// unchanged by any future entrypoint without duplicating any business rule
// (Constitution Principle III).
//
// A single service groups every operation on this one resource (register,
// list, remove, check coverage) — not one interface per use case — the
// same way GroupService groups Create/GetByID/AddUser/... in
// waliqueiroz/mystery-gifter-api. Its methods only orchestrate ports and
// domain entities/functions — the business rules themselves (what a
// GeoDataSource looks like, how coverage is computed) live in
// internal/domain (geo_data_source.go, geo_data_coverage.go), the same way
// GroupService.AddUser delegates the actual rule to domain.Group.AddUser.
type GeoDataService interface {
	// Register registers a local map or elevation data file under name,
	// discovering its type, format and geographic area from its content
	// (FR-001 through FR-008).
	Register(name, path string) (domain.GeoDataSource, error)

	// List returns every registered source, each with its live file
	// availability (FR-009, FR-010).
	List() ([]domain.GeoDataSummary, error)

	// Remove deletes the registered source with the given name, without
	// touching the underlying data file on disk (FR-011, FR-012).
	Remove(name string) error

	// Clear removes every registered source at once (010-geo-data-source-control
	// FR-001, FR-002). Without confirmed, nothing is removed: it reports, via
	// ErrRegistryClearNotConfirmed, how many entries would be removed (FR-003).
	Clear(confirmed bool) (removedCount int, err error)

	// CheckCoverage verifies whether the track read from reader is covered
	// by the registered sources (FR-013 through FR-018), or, when selection
	// names one, exclusively by the requested source of that type
	// (010-geo-data-source-control FR-004 through FR-010).
	CheckCoverage(reader io.Reader, selection domain.SourceSelection) (domain.CoverageReport, error)

	// ElevationAt reads the elevation of a coordinate from the registered
	// elevation data (FR-017, FR-018): the cell that contains it, in the
	// source that wins there. A cell the file has no value for is a reading
	// without value, not an error; a coordinate no source covers fails with
	// ErrElevationNotCovered.
	ElevationAt(latitude, longitude float64) (domain.ElevationReading, error)
}

type geoDataService struct {
	repository   domain.GeoDataRepository
	inspector    domain.GeoDataInspector
	fileChecker  domain.FileChecker
	trackService TrackService

	elevationReader domain.ElevationReader
}

// NewGeoDataService creates a GeoDataService backed by the given ports and
// by the TrackService that provides the cleaned track for coverage checks.
// The ElevationReader reads the elevation of a coordinate.
func NewGeoDataService(
	repository domain.GeoDataRepository,
	inspector domain.GeoDataInspector,
	fileChecker domain.FileChecker,
	trackService TrackService,
	elevationReader domain.ElevationReader,
) GeoDataService {
	return &geoDataService{
		repository:   repository,
		inspector:    inspector,
		fileChecker:  fileChecker,
		trackService: trackService,

		elevationReader: elevationReader,
	}
}

func (s *geoDataService) Register(name, path string) (domain.GeoDataSource, error) {
	_, found, err := s.repository.FindByName(name)
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

	source := domain.NewGeoDataSource(name, path, inspected)

	if err := s.repository.Save(source); err != nil {
		return domain.GeoDataSource{}, err
	}

	return source, nil
}

func (s *geoDataService) List() ([]domain.GeoDataSummary, error) {
	sources, err := s.repository.List()
	if err != nil {
		return nil, err
	}

	summaries := make([]domain.GeoDataSummary, len(sources))
	for i, source := range sources {
		summaries[i] = domain.GeoDataSummary{
			Source:    source,
			Available: s.fileChecker.Exists(source.Path),
		}
	}

	return summaries, nil
}

func (s *geoDataService) Remove(name string) error {
	_, found, err := s.repository.FindByName(name)
	if err != nil {
		return err
	}
	if !found {
		return domain.ErrDataSourceNotRegistered
	}

	return s.repository.Delete(name)
}

func (s *geoDataService) Clear(confirmed bool) (int, error) {
	sources, err := s.repository.List()
	if err != nil {
		return 0, err
	}

	if !confirmed {
		return len(sources), fmt.Errorf("%w: %d entries would be removed; re-run with --confirm", domain.ErrRegistryClearNotConfirmed, len(sources))
	}

	if err := s.repository.Clear(); err != nil {
		return 0, err
	}
	return len(sources), nil
}

func (s *geoDataService) CheckCoverage(reader io.Reader, selection domain.SourceSelection) (domain.CoverageReport, error) {
	// The selection is resolved first, before treating the track, so an
	// invalid requested name is refused before any other work
	// (010-geo-data-source-control FR-006).
	sources, err := s.repository.List()
	if err != nil {
		return domain.CoverageReport{}, err
	}
	baseMaps, elevations := partitionAvailableSources(s.fileChecker, sources)
	baseMaps, elevations, err = selection.Resolve(baseMaps, elevations)
	if err != nil {
		return domain.CoverageReport{}, err
	}

	// The route used for coverage is cleaned but not simplified/smoothed:
	// those two steps are rendering preparation and could shift points,
	// masking a real coverage gap (research.md item 9).
	cleaned, err := s.trackService.Clean(reader)
	if err != nil {
		return domain.CoverageReport{}, err
	}

	return cleaned.Route.Coverage(baseMaps, elevations), nil
}

func (s *geoDataService) ElevationAt(latitude, longitude float64) (domain.ElevationReading, error) {
	coordinate, err := domain.NewCoordinate(latitude, longitude)
	if err != nil {
		return domain.ElevationReading{}, err
	}

	sources, err := s.repository.List()
	if err != nil {
		return domain.ElevationReading{}, err
	}
	_, elevations := partitionAvailableSources(s.fileChecker, sources)

	source, found := domain.SelectSource(elevations, coordinate.Latitude, coordinate.Longitude)
	if !found {
		return domain.ElevationReading{}, fmt.Errorf("%w: latitude %g, longitude %g (\"geodata list\" shows what is registered)", domain.ErrElevationNotCovered, coordinate.Latitude, coordinate.Longitude)
	}

	info, err := s.elevationReader.Describe(source.Path)
	if err != nil {
		return domain.ElevationReading{}, sourceError(source, err)
	}

	// the same cell a slice reads for the point (info.CellAt), so the value
	// of a query and the one in a slice always agree (FR-018)
	row, col := info.CellAt(coordinate.Latitude, coordinate.Longitude)
	read, err := s.elevationReader.ReadWindow(source.Path, domain.GridWindow{FirstRow: row, FirstCol: col, Rows: 1, Cols: 1})
	if err != nil {
		return domain.ElevationReading{}, sourceError(source, err)
	}
	if len(read.Values) != 1 {
		return domain.ElevationReading{}, sourceError(source, fmt.Errorf("%w: the reader gave %d samples for one cell", domain.ErrGeoDataContentUnreadable, len(read.Values)))
	}

	return domain.NewElevationReading(coordinate, source, row, col, read.Values[0]), nil
}

// partitionAvailableSources splits sources into base map and elevation
// candidates, excluding any whose file is no longer found (FR-017) — needs a
// FileChecker, so it stays here rather than in domain.Route.Coverage, which is
// a pure function.
func partitionAvailableSources(fileChecker domain.FileChecker, sources []domain.GeoDataSource) (baseMaps, elevations []domain.GeoDataSource) {
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
