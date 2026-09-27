// Package zipfile implements the outbound adapters that write a ZIP file. It
// writes the geo data slice (specs/004-geo-data-slice/contracts/slice-file.md).
package zipfile

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/atomicfile"
)

// sliceFormatVersion is the version of the exported slice format
// (specs/004-geo-data-slice/contracts/slice-file.md). It changes only when a
// field is removed or changes meaning.
const sliceFormatVersion = 1

// entryTime is the modification date of every entry, fixed so a slice always
// produces the same bytes.
var entryTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

// noValueBits is the quiet NaN that stands for an elevation sample without
// value in the exported grids.
const noValueBits = 0x7FC00000

// GeoSliceExporter implements domain.GeoSliceExporter, writing a geo data
// slice as one ZIP file: a manifest, one grid of elevation samples for each
// region and the tiles as the base maps store them.
type GeoSliceExporter struct{}

// NewGeoSliceExporter creates a GeoSliceExporter.
func NewGeoSliceExporter() GeoSliceExporter {
	return GeoSliceExporter{}
}

// number is a JSON number printed with a fixed maximum number of decimal
// places and no trailing zeros, so the same value always produces the same
// bytes.
type number struct {
	value  float64
	places int
}

func (n number) MarshalJSON() ([]byte, error) {
	text := strconv.FormatFloat(n.value, 'f', n.places, 64)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	}
	if text == "-0" {
		text = "0"
	}
	return []byte(text), nil
}

func coordinate(v float64) number { return number{v, 7} }
func cellSize(v float64) number   { return number{v, 12} }
func meters(v float64) number     { return number{v, 3} }

type areaFile struct {
	MinLatitude         number `json:"min_lat"`
	MaxLatitude         number `json:"max_lat"`
	MinLongitude        number `json:"min_lon"`
	MaxLongitude        number `json:"max_lon"`
	CrossesAntimeridian bool   `json:"crosses_antimeridian"`
}

type elevationRangeFile struct {
	Min number `json:"min"`
	Max number `json:"max"`
}

type summaryFile struct {
	TileCount          int                 `json:"tile_count"`
	MissingTileCount   int                 `json:"missing_tile_count"`
	SampleCount        int                 `json:"sample_count"`
	NoValueSampleCount int                 `json:"no_value_sample_count"`
	Elevation          *elevationRangeFile `json:"elevation_m"`
	SizeBytes          int64               `json:"size_bytes"`
}

type sourceFile struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Format string `json:"format"`
	Path   string `json:"path"`
}

type levelFile struct {
	Ideal  int    `json:"ideal"`
	Chosen int    `json:"chosen"`
	Min    int    `json:"min"`
	Max    int    `json:"max"`
	Reason string `json:"reason"`
}

type tileFile struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

type missingFile struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type gridFile struct {
	Source       int    `json:"source"`
	File         string `json:"file"`
	Rows         int    `json:"rows"`
	Cols         int    `json:"cols"`
	NorthLat     number `json:"north_lat"`
	WestLon      number `json:"west_lon"`
	CellLat      number `json:"cell_lat"`
	CellLon      number `json:"cell_lon"`
	NoValueCount int    `json:"no_value_count"`
}

// Export writes slice to path, atomically: a failure never leaves a partial
// file, and without overwrite an existing path is refused.
func (GeoSliceExporter) Export(slice domain.GeoSlice, path string, overwrite bool) error {
	err := atomicfile.Publish(path, overwrite, func(w io.Writer) error {
		return writeSlice(w, slice)
	})

	switch {
	case err == nil:
		return nil
	case errors.Is(err, atomicfile.ErrExists):
		return fmt.Errorf("%w: %s (use --overwrite to replace it)", domain.ErrSliceDestinationExists, path)
	default:
		return fmt.Errorf("%w: %s: %w", domain.ErrSliceDestinationInvalid, path, err)
	}
}

func writeSlice(w io.Writer, slice domain.GeoSlice) error {
	sourceIndex := make(map[string]int, len(slice.Summary.Sources))
	for i, use := range slice.Summary.Sources {
		sourceIndex[use.Source.Name] = i
	}

	archive := zip.NewWriter(w)

	add := func(name string, data []byte) error {
		entry, err := archive.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store, Modified: entryTime})
		if err != nil {
			return err
		}
		_, err = entry.Write(data)
		return err
	}

	manifest, elevationFiles, tilePaths, err := buildManifest(slice, sourceIndex)
	if err != nil {
		return err
	}

	if err := add("manifest.json", manifest); err != nil {
		return err
	}
	for i, grid := range slice.Elevation {
		if err := add(elevationFiles[i], encodeGrid(grid)); err != nil {
			return err
		}
	}
	for _, tileSet := range slice.TileSets {
		for _, tile := range tileSet.Tiles {
			if err := add(tilePaths[tile.ID][tileSet.Source.Name], tile.Data); err != nil {
				return err
			}
		}
	}

	return archive.Close()
}

// tilePath is where a tile is in the file.
func tilePath(sourceIndex int, format string, id domain.TileID) string {
	return fmt.Sprintf("tiles/%03d/%d/%d/%d.%s", sourceIndex, id.Level, id.X, id.Y, format)
}

func tileExtension(tileSet domain.TileSet) string {
	if tileSet.Format == "" {
		return "png"
	}
	return tileSet.Format
}

func encodeGrid(grid domain.ElevationGrid) []byte {
	data := make([]byte, 0, grid.Rows()*grid.Cols()*domain.BytesPerElevationSample)
	for row := 0; row < grid.Rows(); row++ {
		for col := 0; col < grid.Cols(); col++ {
			bits := uint32(noValueBits)
			if value, ok := grid.At(row, col); ok {
				bits = math.Float32bits(float32(value))
			}
			data = binary.LittleEndian.AppendUint32(data, bits)
		}
	}
	return data
}

// buildManifest renders manifest.json: the header objects indented by two
// spaces and every item of a list compact, on a line of its own, so the file is
// easy to inspect with head, grep and diff. It also returns where each grid
// and tile goes in the ZIP.
func buildManifest(slice domain.GeoSlice, sourceIndex map[string]int) (manifest []byte, elevationFiles []string, tilePaths map[domain.TileID]map[string]string, err error) {
	summary := slice.Summary

	var elevation *elevationRangeFile
	if summary.HasElevationRange {
		elevation = &elevationRangeFile{meters(summary.MinElevation), meters(summary.MaxElevation)}
	}

	area, err := json.MarshalIndent(areaFile{
		coordinate(summary.Area.MinLatitude), coordinate(summary.Area.MaxLatitude),
		coordinate(summary.Area.MinLongitude), coordinate(summary.Area.MaxLongitude),
		summary.Area.CrossesAntimeridian,
	}, "  ", "  ")
	if err != nil {
		return nil, nil, nil, err
	}
	summaryJSON, err := json.MarshalIndent(summaryFile{
		summary.TileCount, summary.MissingTileCount, summary.SampleCount, summary.NoValueSampleCount,
		elevation, summary.SizeBytes,
	}, "  ", "  ")
	if err != nil {
		return nil, nil, nil, err
	}

	var out bytes.Buffer
	out.WriteString("{\n  \"format_version\": " + strconv.Itoa(sliceFormatVersion) + ",\n")
	out.WriteString("  \"plan_id\": " + strconv.Quote(slice.PlanID) + ",\n")
	out.WriteString("  \"area\": " + string(area) + ",\n")
	out.WriteString("  \"summary\": " + string(summaryJSON) + ",\n")

	sources := make([]sourceFile, len(summary.Sources))
	for i, use := range summary.Sources {
		sources[i] = sourceFile{use.Source.Name, string(use.Source.Type), string(use.Source.Format), use.Source.Path}
	}
	if err := writeList(&out, "  ", "sources", sources); err != nil {
		return nil, nil, nil, err
	}
	out.WriteString(",\n")

	tilePaths = map[domain.TileID]map[string]string{}
	out.WriteString("  \"base_map\": [")
	for i, tileSet := range slice.TileSets {
		if i > 0 {
			out.WriteString(",")
		}
		out.WriteString("\n    {\n")

		index := sourceIndex[tileSet.Source.Name]
		level, err := json.Marshal(levelFile{tileSet.Detail.Ideal, tileSet.Detail.Chosen, tileSet.Detail.Min, tileSet.Detail.Max, tileSet.Detail.Reason})
		if err != nil {
			return nil, nil, nil, err
		}
		fmt.Fprintf(&out, "      \"source\": %d,\n      \"tile_format\": %q,\n      \"level\": %s,\n", index, tileExtension(tileSet), level)

		tiles := make([]tileFile, len(tileSet.Tiles))
		for j, tile := range tileSet.Tiles {
			path := tilePath(index, tileExtension(tileSet), tile.ID)
			if tilePaths[tile.ID] == nil {
				tilePaths[tile.ID] = map[string]string{}
			}
			tilePaths[tile.ID][tileSet.Source.Name] = path
			tiles[j] = tileFile{tile.ID.X, tile.ID.Y, path, len(tile.Data)}
		}
		if err := writeList(&out, "      ", "tiles", tiles); err != nil {
			return nil, nil, nil, err
		}
		out.WriteString(",\n")

		missing := make([]missingFile, len(tileSet.Missing))
		for j, id := range tileSet.Missing {
			missing[j] = missingFile{id.X, id.Y}
		}
		if err := writeList(&out, "      ", "missing", missing); err != nil {
			return nil, nil, nil, err
		}
		out.WriteString("\n    }")
	}
	if len(slice.TileSets) > 0 {
		out.WriteString("\n  ")
	}
	out.WriteString("],\n")

	grids := make([]gridFile, len(slice.Elevation))
	elevationFiles = make([]string, len(slice.Elevation))
	for i, grid := range slice.Elevation {
		elevationFiles[i] = fmt.Sprintf("elevation/%03d.f32", i)
		grids[i] = gridFile{
			Source: sourceIndex[grid.Source.Name], File: elevationFiles[i],
			Rows: grid.Rows(), Cols: grid.Cols(),
			NorthLat: coordinate(grid.NorthLatitude), WestLon: coordinate(grid.WestLongitude),
			CellLat: cellSize(grid.CellLatitude), CellLon: cellSize(grid.CellLongitude),
			NoValueCount: grid.NoValueCount(),
		}
	}
	if err := writeList(&out, "  ", "elevation", grids); err != nil {
		return nil, nil, nil, err
	}
	out.WriteString("\n}\n")

	return out.Bytes(), elevationFiles, tilePaths, nil
}

// writeList writes `"name": [` and the items, each compact on its own line,
// then `]`; an empty list is `[]`. The list is always written, never omitted.
func writeList[T any](out *bytes.Buffer, indent, name string, items []T) error {
	fmt.Fprintf(out, "%s%q: [", indent, name)
	for i, item := range items {
		line, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if i > 0 {
			out.WriteString(",")
		}
		out.WriteString("\n" + indent + "  ")
		out.Write(line)
	}
	if len(items) > 0 {
		out.WriteString("\n" + indent)
	}
	out.WriteString("]")
	return nil
}
