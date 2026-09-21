package application

import (
	"fmt"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -destination mockapplication/geo_slice_service.go -package mockapplication . GeoSliceService

// GeoSliceService reads the content of the geo data the user registered and
// gathers what a flight needs of it (the fourth stage): the elevation samples
// and the base map tiles under the area a camera plan covers. Like every
// service, it groups all the operations on one resource and only orchestrates:
// the rules — the area of a plan, the regions of an area, the level of detail,
// the tiles to ask for, the coverage check — live in internal/domain; the
// reading of files behind the domain.BaseMapReader and domain.ElevationReader
// ports.
type GeoSliceService interface {
	// Generate gathers the geo data slice of plan from the registered sources
	// whose files are still there. It refuses, before reading any content, an
	// area the registered data does not fully cover (ErrAreaNotCovered).
	Generate(plan domain.CameraPlan) (domain.GeoSlice, error)

	// Export writes slice to path; unless overwrite is true it refuses a path
	// that already exists.
	Export(slice domain.GeoSlice, path string, overwrite bool) error
}

type geoSliceService struct {
	repository      domain.GeoDataRepository
	fileChecker     domain.FileChecker
	baseMapReader   domain.BaseMapReader
	elevationReader domain.ElevationReader
	exporter        domain.GeoSliceExporter

	// sliceTuning holds the slice's constants and cameraTuning the camera's
	// field of view, which the level of detail depends on. Both are resolved
	// by an outbound configuration adapter and injected by whoever assembles
	// the service (Constitution Principle VIII).
	sliceTuning  domain.SliceTuning
	cameraTuning domain.CameraTuning
}

// NewGeoSliceService creates a GeoSliceService backed by the given ports.
func NewGeoSliceService(
	repository domain.GeoDataRepository,
	fileChecker domain.FileChecker,
	baseMapReader domain.BaseMapReader,
	elevationReader domain.ElevationReader,
	exporter domain.GeoSliceExporter,
	sliceTuning domain.SliceTuning,
	cameraTuning domain.CameraTuning,
) GeoSliceService {
	return &geoSliceService{
		repository:      repository,
		fileChecker:     fileChecker,
		baseMapReader:   baseMapReader,
		elevationReader: elevationReader,
		exporter:        exporter,
		sliceTuning:     sliceTuning,
		cameraTuning:    cameraTuning,
	}
}

func (s *geoSliceService) Generate(plan domain.CameraPlan) (domain.GeoSlice, error) {
	area := plan.AreaOfInterest(s.sliceTuning)

	sources, err := s.repository.List()
	if err != nil {
		return domain.GeoSlice{}, err
	}
	baseMaps, elevations := partitionAvailableSources(s.fileChecker, sources)

	regions, route := area.Regions(baseMaps, elevations)
	if report := route.Coverage(baseMaps, elevations); report.Status != domain.CoverageStatusFull {
		return domain.GeoSlice{}, &domain.AreaNotCoveredError{Report: report}
	}

	// What is needed is worked out from the metadata of the files alone, so a
	// slice that is too big is refused before any content is read.
	tilesToRead, err := s.planTiles(plan, area, regions)
	if err != nil {
		return domain.GeoSlice{}, err
	}
	windowsToRead, err := s.planElevation(area, regions)
	if err != nil {
		return domain.GeoSlice{}, err
	}

	var tileCount, sampleCount int64
	level := 0
	for _, read := range tilesToRead {
		tileCount += int64(len(read.ids))
		level = max(level, read.detail.Chosen)
	}
	for _, read := range windowsToRead {
		sampleCount += int64(read.window.Rows * read.window.Cols)
	}
	if err := s.sliceTuning.EnsureFits(s.sliceTuning.Estimate(tileCount, sampleCount), area, level); err != nil {
		return domain.GeoSlice{}, err
	}

	var size int64
	tileSets := make([]domain.TileSet, 0, len(tilesToRead))
	for _, read := range tilesToRead {
		tileSet, err := s.readTileSet(read)
		if err != nil {
			return domain.GeoSlice{}, err
		}
		for _, tile := range tileSet.Tiles {
			size += int64(len(tile.Data))
		}
		if err := s.sliceTuning.EnsureFits(size, area, level); err != nil {
			return domain.GeoSlice{}, err
		}
		tileSets = append(tileSets, tileSet)
	}

	grids := make([]domain.ElevationGrid, 0, len(windowsToRead))
	for _, read := range windowsToRead {
		grid, err := s.readGrid(read)
		if err != nil {
			return domain.GeoSlice{}, err
		}
		size += int64(read.window.Rows*read.window.Cols) * domain.BytesPerElevationSample
		if err := s.sliceTuning.EnsureFits(size, area, level); err != nil {
			return domain.GeoSlice{}, err
		}
		grids = append(grids, grid)
	}

	return domain.NewGeoSlice(area, tileSets, grids), nil
}

// tileRead is what to ask one base map for.
type tileRead struct {
	source domain.GeoDataSource
	detail domain.DetailLevel
	ids    []domain.TileID
}

// windowRead is what to read from one elevation source.
type windowRead struct {
	source domain.GeoDataSource
	info   domain.ElevationGridInfo
	window domain.GridWindow
}

// planTiles chooses the level of detail of each base map that wins in some
// region and works out the tiles to ask it for.
func (s *geoSliceService) planTiles(plan domain.CameraPlan, area domain.BoundingBox, regions domain.SliceRegions) ([]tileRead, error) {
	baseMaps := regions.BaseMaps()

	details := make(map[string]domain.DetailLevel, len(baseMaps))
	levels := make(map[string]int, len(baseMaps))
	for _, baseMap := range baseMaps {
		offered, err := s.baseMapReader.Levels(baseMap.Path)
		if err != nil {
			return nil, sourceError(baseMap, err)
		}

		detail := s.sliceTuning.DetailLevel(plan.Summary.MinCameraDistance, s.cameraTuning.OverviewVerticalFOVDegrees, area, offered)
		details[baseMap.Name], levels[baseMap.Name] = detail, detail.Chosen
	}

	wanted := regions.TilesFor(levels)

	reads := make([]tileRead, len(baseMaps))
	for i, baseMap := range baseMaps {
		reads[i] = tileRead{source: baseMap, detail: details[baseMap.Name], ids: wanted[baseMap.Name]}
	}
	return reads, nil
}

// planElevation works out, for each region, the window of samples to read from
// the elevation source that wins there.
func (s *geoSliceService) planElevation(area domain.BoundingBox, regions domain.SliceRegions) ([]windowRead, error) {
	infos := map[string]domain.ElevationGridInfo{}

	var reads []windowRead
	for _, region := range regions {
		source := region.Elevation

		info, described := infos[source.Name]
		if !described {
			var err error
			if info, err = s.elevationReader.Describe(source.Path); err != nil {
				return nil, sourceError(source, err)
			}
			infos[source.Name] = info
		}

		for _, window := range info.Window(region.Box, area) {
			reads = append(reads, windowRead{source: source, info: info, window: window})
		}
	}
	return reads, nil
}

func (s *geoSliceService) readTileSet(read tileRead) (domain.TileSet, error) {
	tiles, err := s.baseMapReader.ReadTiles(read.source.Path, read.detail.Chosen, read.ids)
	if err != nil {
		return domain.TileSet{}, sourceError(read.source, err)
	}

	return domain.TileSet{
		Source:  read.source,
		Detail:  read.detail,
		Format:  tiles.Format,
		Tiles:   tiles.Tiles,
		Missing: tiles.Missing,
	}, nil
}

func (s *geoSliceService) readGrid(read windowRead) (domain.ElevationGrid, error) {
	samples, err := s.elevationReader.ReadWindow(read.source.Path, read.window)
	if err != nil {
		return domain.ElevationGrid{}, sourceError(read.source, err)
	}

	return domain.NewElevationGrid(read.source, read.window, read.info, samples.Values), nil
}

// sourceError says which registered source an error comes from, keeping the
// error itself for errors.Is.
func sourceError(source domain.GeoDataSource, err error) error {
	return fmt.Errorf("source %q: %w", source.Name, err)
}

func (s *geoSliceService) Export(slice domain.GeoSlice, path string, overwrite bool) error {
	return s.exporter.Export(slice, path, overwrite)
}
