package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/geo_slice_exporter.go -package mockdomain . GeoSliceExporter

import (
	"fmt"
	"math"
	"sort"
)

// GeoSliceExporter writes a geo data slice outside the process, as a file the
// later stages consume. Concrete implementations live in
// internal/infra/outbound.
type GeoSliceExporter interface {
	// Export writes slice to path. Unless overwrite is true it must refuse a
	// path that already exists (ErrSliceDestinationExists), and it must
	// never leave a partial file behind on failure.
	Export(slice GeoSlice, path string, overwrite bool) error
}

const (
	// TilePixels is the width and height of a map tile, in pixels.
	TilePixels = 256

	// BytesPerElevationSample is the size of an elevation sample in a slice.
	BytesPerElevationSample = 4

	// MaxMercatorLatitude is the latitude beyond which Web Mercator has no
	// tiles.
	MaxMercatorLatitude = 85.0511287798

	// MetersPerDegree is the length of a degree of latitude, in meters.
	MetersPerDegree = 111320.0

	// EquatorResolution is the ground resolution of a zoom-0 tile pixel at
	// the equator, in meters.
	EquatorResolution = 156543.03392
)

// SliceTuning holds the heuristic constants of the slice: how much terrain
// around the camera is relevant, the reference for choosing a level of
// detail, and the size limit. They are injected (Constitution Principle
// VIII): the core defines the shape, an outbound configuration adapter
// provides the values. See specs/004-geo-data-slice/research.md.
type SliceTuning struct {
	// MarginFactor is the half-side of the terrain relevant to a frame, in
	// multiples of the camera-to-marker distance of that frame.
	MarginFactor float64

	// ReferenceHeightPixels and TexelScreenRatio choose the level of detail:
	// a tile pixel may cover up to TexelScreenRatio screen pixels of a
	// screen ReferenceHeightPixels tall, at the nearest the camera gets.
	ReferenceHeightPixels float64
	TexelScreenRatio      float64

	// EstimatedTileBytes is the size assumed for a tile before reading it,
	// and MaxSizeBytes is the largest slice accepted.
	EstimatedTileBytes int64
	MaxSizeBytes       int64
}

// Estimate is how many bytes a slice of that many tiles and samples takes,
// before reading them: the tiles at the size assumed for a tile, the samples
// at BytesPerElevationSample each.
func (t SliceTuning) Estimate(tileCount, sampleCount int64) int64 {
	return tileCount*t.EstimatedTileBytes + sampleCount*BytesPerElevationSample
}

// EnsureFits refuses, with ErrSliceTooLarge, a slice of size bytes that is
// bigger than MaxSizeBytes (a slice of exactly that size is accepted). It is
// used with the estimate, before any content is read, and again with the real
// size as the content comes in. The message says what to try: the level and
// the area are what make a slice big.
func (t SliceTuning) EnsureFits(size int64, area BoundingBox, level int) error {
	if size <= t.MaxSizeBytes {
		return nil
	}

	width, height := area.Extent()
	return fmt.Errorf("%w: %s is more than the limit of %s, at level %d over an area of %.1f km × %.1f km; try a higher --distance in the plan, or a shorter track",
		ErrSliceTooLarge, formatSize(size), formatSize(t.MaxSizeBytes), level, width, height)
}

func formatSize(size int64) string {
	const kib = 1024
	switch {
	case size < kib:
		return fmt.Sprintf("%d B", size)
	case size < kib*kib:
		return fmt.Sprintf("%.1f KiB", float64(size)/kib)
	case size < kib*kib*kib:
		return fmt.Sprintf("%.1f MiB", float64(size)/(kib*kib))
	default:
		return fmt.Sprintf("%.1f GiB", float64(size)/(kib*kib*kib))
	}
}

// SliceRegions are the regions of a slice's area, in order (see
// BoundingBox.Regions).
type SliceRegions []SliceRegion

// SliceRegion is a rectangle of the slice's area in which the winning
// registered source of each type is the same everywhere.
type SliceRegion struct {
	Box       BoundingBox
	BaseMap   GeoDataSource
	Elevation GeoDataSource
}

// SliceSourceUse is a registered source the slice was extracted from; Detail
// is set for a base map.
type SliceSourceUse struct {
	Source GeoDataSource
	Detail *DetailLevel
}

// SliceSummary describes a slice at a glance. It is computed from the
// slice's content (FR-012).
type SliceSummary struct {
	Area BoundingBox

	TileCount, MissingTileCount int

	SampleCount, NoValueSampleCount int

	// MinElevation and MaxElevation are in meters, over the samples that
	// have a value; HasElevationRange is false when none has.
	MinElevation, MaxElevation float64
	HasElevationRange          bool

	// Sources are the registered sources used, sorted by name.
	Sources []SliceSourceUse

	// SizeBytes is the real size of the slice: the bytes of its tiles plus
	// BytesPerElevationSample for each sample.
	SizeBytes int64
}

// GeoSlice is the geo data a flight needs: the elevation samples and the base
// map tiles under the area the camera covers.
type GeoSlice struct {
	Area      BoundingBox
	TileSets  []TileSet
	Elevation []ElevationGrid
	Summary   SliceSummary
}

// NewGeoSlice assembles a slice from its content and computes its summary
// from it, so the summary can never disagree with the content. The tile sets
// are put in order — by the name of the base map, then the level — and so are
// the tiles and the missing tiles of each — by level, column and row — so a
// slice is always the same whatever the order it was gathered in. The
// elevation grids keep the order of the regions they came from.
func NewGeoSlice(area BoundingBox, tileSets []TileSet, grids []ElevationGrid) GeoSlice {
	tileSets = sortedTileSets(tileSets)
	summary := SliceSummary{Area: area}
	used := map[string]SliceSourceUse{}

	for _, tileSet := range tileSets {
		summary.TileCount += len(tileSet.Tiles)
		summary.MissingTileCount += len(tileSet.Missing)
		for _, tile := range tileSet.Tiles {
			summary.SizeBytes += int64(len(tile.Data))
		}

		detail := tileSet.Detail
		if _, ok := used[tileSet.Source.Name]; !ok {
			used[tileSet.Source.Name] = SliceSourceUse{Source: tileSet.Source, Detail: &detail}
		}
	}

	for _, grid := range grids {
		summary.SampleCount += grid.Rows() * grid.Cols()
		summary.NoValueSampleCount += grid.NoValueCount()
		if _, ok := used[grid.Source.Name]; !ok {
			used[grid.Source.Name] = SliceSourceUse{Source: grid.Source}
		}

		if minimum, maximum, ok := grid.Range(); ok {
			if !summary.HasElevationRange {
				summary.MinElevation, summary.MaxElevation, summary.HasElevationRange = minimum, maximum, true
			} else {
				summary.MinElevation = math.Min(summary.MinElevation, minimum)
				summary.MaxElevation = math.Max(summary.MaxElevation, maximum)
			}
		}
	}
	summary.SizeBytes += BytesPerElevationSample * int64(summary.SampleCount)

	for _, use := range used {
		summary.Sources = append(summary.Sources, use)
	}
	sort.Slice(summary.Sources, func(i, j int) bool { return summary.Sources[i].Source.Name < summary.Sources[j].Source.Name })

	return GeoSlice{
		Area:      area,
		TileSets:  tileSets,
		Elevation: grids,
		Summary:   summary,
	}
}

// BaseMaps are the base maps that win in some region, each once, in the
// order of their first region.
func (r SliceRegions) BaseMaps() []GeoDataSource {
	var maps []GeoDataSource
	seen := map[string]bool{}
	for _, region := range r {
		if !seen[region.BaseMap.Name] {
			seen[region.BaseMap.Name] = true
			maps = append(maps, region.BaseMap)
		}
	}
	return maps
}

// TilesFor says which tiles to ask each base map for, by the map's name:
// for each region, the tiles that cover it at the level chosen for the
// region's base map. A tile that covers two regions is asked of the base map
// of the first of them, so no tile is asked twice; each list is sorted by
// column, then row.
func (r SliceRegions) TilesFor(levels map[string]int) map[string][]TileID {
	tiles := map[string][]TileID{}
	seen := map[TileID]bool{}

	for _, region := range r {
		for _, span := range region.Box.TileRange(levels[region.BaseMap.Name]) {
			for x := span.MinX; x <= span.MaxX; x++ {
				for y := span.MinY; y <= span.MaxY; y++ {
					id := TileID{Level: span.Level, X: x, Y: y}
					if !seen[id] {
						seen[id] = true
						tiles[region.BaseMap.Name] = append(tiles[region.BaseMap.Name], id)
					}
				}
			}
		}
	}

	for _, ids := range tiles {
		sort.Slice(ids, func(i, j int) bool {
			if ids[i].X != ids[j].X {
				return ids[i].X < ids[j].X
			}
			return ids[i].Y < ids[j].Y
		})
	}
	return tiles
}

func sortedTileSets(tileSets []TileSet) []TileSet {
	sorted := make([]TileSet, len(tileSets))
	for i, tileSet := range tileSets {
		tileSet.Tiles = append([]Tile(nil), tileSet.Tiles...)
		sort.Slice(tileSet.Tiles, func(a, b int) bool { return tileSet.Tiles[a].ID.before(tileSet.Tiles[b].ID) })

		tileSet.Missing = append([]TileID(nil), tileSet.Missing...)
		sort.Slice(tileSet.Missing, func(a, b int) bool { return tileSet.Missing[a].before(tileSet.Missing[b]) })

		sorted[i] = tileSet
	}

	sort.SliceStable(sorted, func(a, b int) bool {
		if sorted[a].Source.Name != sorted[b].Source.Name {
			return sorted[a].Source.Name < sorted[b].Source.Name
		}
		return sorted[a].Detail.Chosen < sorted[b].Detail.Chosen
	})
	return sorted
}
