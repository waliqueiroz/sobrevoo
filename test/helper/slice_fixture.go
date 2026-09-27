package helper

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// SliceSourceSpec is a registered source a slice file says it was made from.
type SliceSourceSpec struct {
	Name, Type, Format, Path string
}

// SliceBaseMapSpec is the tiles of one base map in a slice file, at one level:
// Source is an index into SliceFileSpec.Sources.
type SliceBaseMapSpec struct {
	Source                  int
	TileFormat              string
	Ideal, Chosen, Min, Max int
	Reason                  string
	Tiles                   []MBTile

	// Missing are the [x, y] of the tiles the slice needed and the source lacks.
	Missing [][2]int
}

// SliceGridSpec is one grid of elevation samples of a slice file; a NaN value is
// a sample with no value.
type SliceGridSpec struct {
	Source            int
	Rows, Cols        int
	NorthLat, WestLon float64
	CellLat, CellLon  float64
	Values            []float32
}

// SliceFileSpec describes a geo data slice file, the way specs/004-geo-data-slice
// /contracts/slice-file.md defines it. The fixture writes it without the
// exporter, so a test of the reader does not depend on the writer.
type SliceFileSpec struct {
	PlanID string

	MinLat, MaxLat, MinLon, MaxLon float64
	CrossesAntimeridian            bool

	Sources  []SliceSourceSpec
	BaseMaps []SliceBaseMapSpec
	Grids    []SliceGridSpec
}

// DefaultSliceFileSpec is a small slice: a base map ("map") with two PNG tiles
// of level 16 and one missing, and a 4 × 4 grid of elevation ("dem") whose
// sixth sample (row 1, column 1) has no value.
func DefaultSliceFileSpec() SliceFileSpec {
	values := make([]float32, 16)
	for i := range values {
		values[i] = float32(100 + i)
	}
	values[5] = float32(math.NaN())

	return SliceFileSpec{
		PlanID: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MinLat: -23.004, MaxLat: -23, MinLon: -47, MaxLon: -46.996,
		Sources: []SliceSourceSpec{
			{Name: "dem", Type: "elevation", Format: "GeoTIFF", Path: "/data/dem.tif"},
			{Name: "map", Type: "base map", Format: "MBTiles", Path: "/data/map.mbtiles"},
		},
		BaseMaps: []SliceBaseMapSpec{{
			Source: 1, TileFormat: "png",
			Ideal: 17, Chosen: 16, Min: 0, Max: 16, Reason: "above the source's maximum level",
			Tiles: []MBTile{
				{Z: 16, X: 24122, Y: 36869, Data: PNGTile(Color{R: 200, G: 30, B: 30})},
				{Z: 16, X: 24123, Y: 36869, Data: PNGTile(Color{R: 30, G: 200, B: 30})},
			},
			Missing: [][2]int{{24124, 36869}},
		}},
		Grids: []SliceGridSpec{{
			Source: 0, Rows: 4, Cols: 4,
			NorthLat: -23, WestLon: -47, CellLat: 0.001, CellLon: 0.001,
			Values: values,
		}},
	}
}

// sliceEntry is one entry of the ZIP of a slice file.
type sliceEntry struct {
	name   string
	data   []byte
	method uint16
}

// slicePieces is a slice file before it is written: the manifest as a tree, so
// a fixture can spoil one field, and the entries in the order of the contract.
type slicePieces struct {
	manifest map[string]any
	entries  []sliceEntry
}

func encodeSliceGrid(values []float32) []byte {
	data := make([]byte, 0, 4*len(values))
	for _, v := range values {
		bits := math.Float32bits(v)
		if v != v {
			bits = 0x7FC00000
		}
		data = binary.LittleEndian.AppendUint32(data, bits)
	}
	return data
}

func buildSlicePieces(spec SliceFileSpec) slicePieces {
	var (
		entries                    []sliceEntry
		tileCount, missingCount    int
		sampleCount, noValueCount  int
		sizeBytes                  int64
		minElevation, maxElevation = math.Inf(1), math.Inf(-1)
		sources, baseMaps, grids   = []any{}, []any{}, []any{}
	)

	for _, source := range spec.Sources {
		sources = append(sources, map[string]any{"name": source.Name, "type": source.Type, "format": source.Format, "path": source.Path})
	}

	for i, grid := range spec.Grids {
		file := fmt.Sprintf("elevation/%03d.f32", i)
		entries = append(entries, sliceEntry{name: file, data: encodeSliceGrid(grid.Values), method: zip.Store})

		noValue := 0
		for _, v := range grid.Values {
			if v != v {
				noValue++
				continue
			}
			minElevation, maxElevation = math.Min(minElevation, float64(v)), math.Max(maxElevation, float64(v))
		}
		sampleCount += grid.Rows * grid.Cols
		noValueCount += noValue
		sizeBytes += int64(4 * len(grid.Values))

		grids = append(grids, map[string]any{
			"source": grid.Source, "file": file, "rows": grid.Rows, "cols": grid.Cols,
			"north_lat": grid.NorthLat, "west_lon": grid.WestLon, "cell_lat": grid.CellLat, "cell_lon": grid.CellLon,
			"no_value_count": noValue,
		})
	}

	for i, baseMap := range spec.BaseMaps {
		tiles, missing := []any{}, []any{}
		for _, tile := range baseMap.Tiles {
			path := fmt.Sprintf("tiles/%03d/%d/%d/%d.%s", i, tile.Z, tile.X, tile.Y, baseMap.TileFormat)
			entries = append(entries, sliceEntry{name: path, data: tile.Data, method: zip.Store})
			tiles = append(tiles, map[string]any{"x": tile.X, "y": tile.Y, "path": path, "bytes": len(tile.Data)})
			sizeBytes += int64(len(tile.Data))
		}
		for _, m := range baseMap.Missing {
			missing = append(missing, map[string]any{"x": m[0], "y": m[1]})
		}
		tileCount += len(baseMap.Tiles)
		missingCount += len(baseMap.Missing)

		baseMaps = append(baseMaps, map[string]any{
			"source": baseMap.Source, "tile_format": baseMap.TileFormat,
			"level": map[string]any{"ideal": baseMap.Ideal, "chosen": baseMap.Chosen, "min": baseMap.Min, "max": baseMap.Max, "reason": baseMap.Reason},
			"tiles": tiles, "missing": missing,
		})
	}

	var elevation any
	if !math.IsInf(minElevation, 1) {
		elevation = map[string]any{"min": minElevation, "max": maxElevation}
	}

	manifest := map[string]any{
		"format_version": 1,
		"plan_id":        spec.PlanID,
		"area": map[string]any{
			"min_lat": spec.MinLat, "max_lat": spec.MaxLat, "min_lon": spec.MinLon, "max_lon": spec.MaxLon,
			"crosses_antimeridian": spec.CrossesAntimeridian,
		},
		"summary": map[string]any{
			"tile_count": tileCount, "missing_tile_count": missingCount,
			"sample_count": sampleCount, "no_value_sample_count": noValueCount,
			"elevation_m": elevation, "size_bytes": sizeBytes,
		},
		"sources":   sources,
		"base_map":  baseMaps,
		"elevation": grids,
	}

	return slicePieces{manifest: manifest, entries: entries}
}

var sliceEntryTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

func (p slicePieces) write() []byte {
	manifest, err := json.MarshalIndent(p.manifest, "", "  ")
	if err != nil {
		panic("encoding the manifest of a slice fixture: " + err.Error())
	}

	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	add := func(entry sliceEntry) {
		w, err := archive.CreateHeader(&zip.FileHeader{Name: entry.name, Method: entry.method, Modified: sliceEntryTime})
		if err != nil {
			panic("writing an entry of a slice fixture: " + err.Error())
		}
		if _, err := w.Write(entry.data); err != nil {
			panic("writing an entry of a slice fixture: " + err.Error())
		}
	}
	if p.manifest != nil {
		add(sliceEntry{name: "manifest.json", data: manifest, method: zip.Store})
	}
	for _, entry := range p.entries {
		add(entry)
	}
	if err := archive.Close(); err != nil {
		panic("closing a slice fixture: " + err.Error())
	}
	return out.Bytes()
}

func (p slicePieces) section(name string) map[string]any { return p.manifest[name].(map[string]any) }

// ValidSliceFile writes the ZIP of a slice file for spec: store method, a
// manifest with a summary recomputed from the content, the elevation grids as
// float32 little-endian with 0x7FC00000 for no value, and the tiles.
func ValidSliceFile(spec SliceFileSpec) []byte {
	return buildSlicePieces(spec).write()
}

func defaultSlicePieces() slicePieces { return buildSlicePieces(DefaultSliceFileSpec()) }

// TruncatedSliceFile is a valid slice file cut short.
func TruncatedSliceFile() []byte {
	data := ValidSliceFile(DefaultSliceFileSpec())
	return data[:len(data)*2/5]
}

// SliceFileWithVersion is a valid slice file whose format_version is version.
func SliceFileWithVersion(version int) []byte {
	pieces := defaultSlicePieces()
	pieces.manifest["format_version"] = version
	return pieces.write()
}

// SliceFileWithoutFormatVersion is a slice file whose manifest has no
// format_version.
func SliceFileWithoutFormatVersion() []byte {
	pieces := defaultSlicePieces()
	delete(pieces.manifest, "format_version")
	return pieces.write()
}

// SliceFileWithTextFormatVersion is a slice file whose format_version is a
// text, not a number.
func SliceFileWithTextFormatVersion() []byte {
	pieces := defaultSlicePieces()
	pieces.manifest["format_version"] = "one"
	return pieces.write()
}

// SliceFileWithoutPlanID is a slice file made before plan_id existed.
func SliceFileWithoutPlanID() []byte {
	pieces := defaultSlicePieces()
	delete(pieces.manifest, "plan_id")
	return pieces.write()
}

// SliceFileWithBadPlanID is a slice file whose plan_id is not 64 hexadecimal
// characters.
func SliceFileWithBadPlanID() []byte {
	pieces := defaultSlicePieces()
	pieces.manifest["plan_id"] = "not-a-plan-id"
	return pieces.write()
}

// SliceFileWithoutManifest is a ZIP with the entries of a slice and no manifest.
func SliceFileWithoutManifest() []byte {
	pieces := defaultSlicePieces()
	pieces.manifest = nil
	return pieces.write()
}

// SliceFileWithoutField is a slice file whose manifest lacks a top-level field.
func SliceFileWithoutField(field string) []byte {
	pieces := defaultSlicePieces()
	delete(pieces.manifest, field)
	return pieces.write()
}

// SliceFileWithManifest is a ZIP whose manifest.json holds text.
func SliceFileWithManifest(text string) []byte {
	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	w, _ := archive.CreateHeader(&zip.FileHeader{Name: "manifest.json", Method: zip.Store, Modified: sliceEntryTime})
	_, _ = w.Write([]byte(text))
	_ = archive.Close()
	return out.Bytes()
}

// SliceFileWithWrongCounts is a slice file whose summary has a wrong value for
// one field: tile_count, missing_tile_count, sample_count,
// no_value_sample_count, elevation_m or size_bytes.
func SliceFileWithWrongCounts(field string) []byte {
	pieces := defaultSlicePieces()
	summary := pieces.section("summary")
	switch field {
	case "elevation_m":
		summary[field] = map[string]any{"min": 1.0, "max": 2.0}
	case "size_bytes":
		summary[field] = summary[field].(int64) + 1
	default:
		summary[field] = summary[field].(int) + 1
	}
	return pieces.write()
}

// SliceFileWithoutEntry is a slice file that lacks the entry at path (a tile or
// a grid its manifest lists).
func SliceFileWithoutEntry(path string) []byte {
	pieces := defaultSlicePieces()
	kept := pieces.entries[:0]
	for _, entry := range pieces.entries {
		if entry.name != path {
			kept = append(kept, entry)
		}
	}
	pieces.entries = kept
	return pieces.write()
}

// SliceFileWithExtraEntry is a slice file with an entry its manifest does not
// list.
func SliceFileWithExtraEntry() []byte {
	pieces := defaultSlicePieces()
	pieces.entries = append(pieces.entries, sliceEntry{name: "notes.txt", data: []byte("not in the manifest"), method: zip.Store})
	return pieces.write()
}

// SliceFileWithWrongTileSize is a slice file whose manifest gives a tile the
// wrong number of bytes.
func SliceFileWithWrongTileSize() []byte {
	pieces := defaultSlicePieces()
	tiles := pieces.manifest["base_map"].([]any)[0].(map[string]any)["tiles"].([]any)
	tile := tiles[0].(map[string]any)
	tile["bytes"] = tile["bytes"].(int) + 1
	return pieces.write()
}

// SliceFileWithWrongGridSize is a slice file whose manifest says a grid has one
// row more than its entry holds.
func SliceFileWithWrongGridSize() []byte {
	pieces := defaultSlicePieces()
	grid := pieces.manifest["elevation"].([]any)[0].(map[string]any)
	grid["rows"] = grid["rows"].(int) + 1
	return pieces.write()
}

// SliceFileWithCompressedEntry is a slice file whose first grid is stored with
// the deflate method: only store is accepted.
func SliceFileWithCompressedEntry() []byte {
	pieces := defaultSlicePieces()
	for i := range pieces.entries {
		if pieces.entries[i].name == "elevation/000.f32" {
			pieces.entries[i].method = zip.Deflate
		}
	}
	return pieces.write()
}

// SliceFileWithChosenOutOfRange is a slice file whose chosen level is above
// the maximum the source offers.
func SliceFileWithChosenOutOfRange() []byte {
	pieces := defaultSlicePieces()
	level := pieces.manifest["base_map"].([]any)[0].(map[string]any)["level"].(map[string]any)
	level["chosen"] = level["max"].(int) + 1
	return pieces.write()
}

// SliceFileWithBadSourceIndex is a slice file whose base map points at a source
// that is not listed.
func SliceFileWithBadSourceIndex() []byte {
	pieces := defaultSlicePieces()
	pieces.manifest["base_map"].([]any)[0].(map[string]any)["source"] = 9
	return pieces.write()
}

// SliceFileWithDuplicateEntry is a slice file with two entries of the same name.
func SliceFileWithDuplicateEntry() []byte {
	pieces := defaultSlicePieces()
	pieces.entries = append(pieces.entries, pieces.entries[0])
	return pieces.write()
}

// SliceFileWithUnsafeEntry is a slice file that has, besides its own entries,
// one whose name climbs out of the directory.
func SliceFileWithUnsafeEntry(name string) []byte {
	pieces := defaultSlicePieces()
	pieces.entries = append(pieces.entries, sliceEntry{name: name, data: []byte("x"), method: zip.Store})
	return pieces.write()
}

// SliceFileWithCorruptEntry is a slice file in which one byte of the first grid
// was changed after the CRC was written.
func SliceFileWithCorruptEntry() []byte {
	spec := DefaultSliceFileSpec()
	data := ValidSliceFile(spec)
	grid := encodeSliceGrid(spec.Grids[0].Values)
	at := bytes.Index(data, grid)
	if at < 0 {
		panic("the grid of the slice fixture is not in the file")
	}
	data[at] ^= 0xFF
	return data
}

// NotZipContent is bytes that are not a ZIP file.
func NotZipContent() []byte {
	return []byte("this is not a zip file, just some text that goes on for a while")
}
