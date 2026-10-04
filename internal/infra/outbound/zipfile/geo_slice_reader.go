package zipfile

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// GeoSliceReader implements domain.GeoSliceReader: it reads back the ZIP file
// GeoSliceExporter writes (specs/004-geo-data-slice/contracts/slice-file.md),
// which later stages draw from.
type GeoSliceReader struct{}

// NewGeoSliceReader creates a GeoSliceReader.
func NewGeoSliceReader() GeoSliceReader {
	return GeoSliceReader{}
}

// The manifest as it is in the file; a consumer ignores the fields it does not
// know, so the format can grow.
type manifestFile struct {
	FormatVersion int    `json:"format_version"`
	PlanID        string `json:"plan_id"`
	Area          struct {
		MinLatitude  float64 `json:"min_lat"`
		MaxLatitude  float64 `json:"max_lat"`
		MinLongitude float64 `json:"min_lon"`
		MaxLongitude float64 `json:"max_lon"`
		Crosses      bool    `json:"crosses_antimeridian"`
	} `json:"area"`
	Summary struct {
		TileCount          int `json:"tile_count"`
		MissingTileCount   int `json:"missing_tile_count"`
		SampleCount        int `json:"sample_count"`
		NoValueSampleCount int `json:"no_value_sample_count"`
		Elevation          *struct {
			Min float64 `json:"min"`
			Max float64 `json:"max"`
		} `json:"elevation_m"`
		SizeBytes int64 `json:"size_bytes"`
	} `json:"summary"`
	Sources []struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Format string `json:"format"`
		Path   string `json:"path"`
	} `json:"sources"`
	BaseMap []struct {
		Source     int    `json:"source"`
		TileFormat string `json:"tile_format"`
		Level      struct {
			Ideal  int    `json:"ideal"`
			Chosen int    `json:"chosen"`
			Min    int    `json:"min"`
			Max    int    `json:"max"`
			Reason string `json:"reason"`
		} `json:"level"`
		Tiles []struct {
			X     int    `json:"x"`
			Y     int    `json:"y"`
			Path  string `json:"path"`
			Bytes int    `json:"bytes"`
		} `json:"tiles"`
		Missing []struct {
			X int `json:"x"`
			Y int `json:"y"`
		} `json:"missing"`
	} `json:"base_map"`
	Elevation []struct {
		Source       int     `json:"source"`
		File         string  `json:"file"`
		Rows         int     `json:"rows"`
		Cols         int     `json:"cols"`
		NorthLat     float64 `json:"north_lat"`
		WestLon      float64 `json:"west_lon"`
		CellLat      float64 `json:"cell_lat"`
		CellLon      float64 `json:"cell_lon"`
		NoValueCount int     `json:"no_value_count"`
	} `json:"elevation"`
}

// The format this reader understands, and the largest slice file it accepts: the
// limit of a slice (1.5 GiB of content, config.SliceTuning.MaxSizeBytes) and room
// for its manifest. A ZIP that declares more is not a slice this tool wrote, and
// would take the memory of the computer to read.
const maxManifestBytes = 16 << 20

var maxSliceBytes int64 = 1536<<20 + maxManifestBytes

var planIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// requiredFields are the fields of the manifest without which it is not one.
var requiredFields = []string{"area", "summary", "sources", "base_map", "elevation"}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", domain.ErrSliceFileInvalid, fmt.Sprintf(format, args...))
}

// Read reads the slice file at path. An I/O error opening or reading it comes
// back wrapped and with no sentinel; a file that is not a slice is
// domain.ErrSliceFileInvalid.
func (GeoSliceReader) Read(path string) (domain.GeoSlice, error) {
	file, err := os.Open(path)
	if err != nil {
		return domain.GeoSlice{}, fmt.Errorf("reading the slice file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return domain.GeoSlice{}, fmt.Errorf("reading the slice file: %w", err)
	}

	// The identification of the file is the hash of all of it.
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return domain.GeoSlice{}, fmt.Errorf("reading the slice file: %w", err)
	}

	if info.Size() > maxSliceBytes {
		return domain.GeoSlice{}, invalid("the file is bigger than a slice can be (%d bytes, the most is %d)", info.Size(), maxSliceBytes)
	}

	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return domain.GeoSlice{}, invalid("not a ZIP file (%v)", err)
	}
	entries, err := indexEntries(archive.File)
	if err != nil {
		return domain.GeoSlice{}, err
	}

	manifestEntry, ok := entries["manifest.json"]
	if !ok {
		return domain.GeoSlice{}, invalid("the file has no manifest.json")
	}
	data, err := readEntry(manifestEntry)
	if err != nil {
		return domain.GeoSlice{}, err
	}
	manifest, err := parseManifest(data)
	if err != nil {
		return domain.GeoSlice{}, err
	}
	used := map[string]bool{"manifest.json": true}

	sources := make([]domain.GeoDataSource, len(manifest.Sources))
	for i, s := range manifest.Sources {
		sources[i] = domain.GeoDataSource{Name: s.Name, Path: s.Path, Type: domain.DataType(s.Type), Format: domain.DataFormat(s.Format)}
	}
	sourceAt := func(index int) (domain.GeoDataSource, error) {
		if index < 0 || index >= len(sources) {
			return domain.GeoDataSource{}, invalid("source %d is not in the list of %d sources", index, len(sources))
		}
		return sources[index], nil
	}

	tileSets := make([]domain.TileSet, 0, len(manifest.BaseMap))
	for _, baseMap := range manifest.BaseMap {
		source, err := sourceAt(baseMap.Source)
		if err != nil {
			return domain.GeoSlice{}, err
		}

		if level := baseMap.Level; level.Chosen < level.Min || level.Chosen > level.Max {
			return domain.GeoSlice{}, invalid("level.chosen is %d but the source offers %d to %d", level.Chosen, level.Min, level.Max)
		}

		tileSet := domain.TileSet{
			Source: source,
			Detail: domain.DetailLevel{
				Ideal: baseMap.Level.Ideal, Chosen: baseMap.Level.Chosen, Min: baseMap.Level.Min, Max: baseMap.Level.Max,
				Reason: baseMap.Level.Reason,
			},
			Format: baseMap.TileFormat,
		}
		for _, tile := range baseMap.Tiles {
			entry, ok := entries[tile.Path]
			if !ok {
				return domain.GeoSlice{}, invalid("the file has no %s, which the manifest lists", tile.Path)
			}
			data, err := readEntry(entry)
			if err != nil {
				return domain.GeoSlice{}, err
			}
			if len(data) != tile.Bytes {
				return domain.GeoSlice{}, invalid("%s has %d bytes, and the manifest says bytes is %d", tile.Path, len(data), tile.Bytes)
			}
			used[tile.Path] = true
			tileSet.Tiles = append(tileSet.Tiles, domain.Tile{ID: domain.TileID{Level: baseMap.Level.Chosen, X: tile.X, Y: tile.Y}, Data: data})
		}
		for _, missing := range baseMap.Missing {
			tileSet.Missing = append(tileSet.Missing, domain.TileID{Level: baseMap.Level.Chosen, X: missing.X, Y: missing.Y})
		}
		tileSets = append(tileSets, tileSet)
	}

	grids := make([]domain.ElevationGrid, 0, len(manifest.Elevation))
	for _, g := range manifest.Elevation {
		source, err := sourceAt(g.Source)
		if err != nil {
			return domain.GeoSlice{}, err
		}
		entry, ok := entries[g.File]
		if !ok {
			return domain.GeoSlice{}, invalid("the file has no %s, which the manifest lists", g.File)
		}
		data, err := readEntry(entry)
		if err != nil {
			return domain.GeoSlice{}, err
		}
		if g.Rows < 0 || g.Cols < 0 || len(data) != 4*g.Rows*g.Cols {
			return domain.GeoSlice{}, invalid("%s has %d bytes, and %d rows of %d columns of samples take %d", g.File, len(data), g.Rows, g.Cols, 4*g.Rows*g.Cols)
		}
		used[g.File] = true

		values := make([]float32, g.Rows*g.Cols)
		for i := range values {
			values[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[4*i:]))
		}
		gridInfo := domain.ElevationGridInfo{
			Rows: g.Rows, Cols: g.Cols,
			NorthLatitude: g.NorthLat, WestLongitude: g.WestLon,
			CellLatitude: g.CellLat, CellLongitude: g.CellLon,
			UnitToMeters: 1,
		}
		grids = append(grids, domain.NewElevationGrid(source, domain.GridWindow{Rows: g.Rows, Cols: g.Cols}, gridInfo, values))
	}

	area := domain.BoundingBox{
		MinLatitude: manifest.Area.MinLatitude, MaxLatitude: manifest.Area.MaxLatitude,
		MinLongitude: manifest.Area.MinLongitude, MaxLongitude: manifest.Area.MaxLongitude,
		CrossesAntimeridian: manifest.Area.Crosses,
	}
	for _, entry := range archive.File {
		if !used[entry.Name] {
			return domain.GeoSlice{}, invalid("the file has %s, which the manifest does not list", entry.Name)
		}
	}

	slice := domain.NewGeoSlice(area, tileSets, grids)
	if err := checkSummary(manifest, slice.Summary); err != nil {
		return domain.GeoSlice{}, err
	}
	slice.PlanID = manifest.PlanID
	slice.ContentID = hex.EncodeToString(hash.Sum(nil))
	return slice, nil
}

// readEntry reads all of an entry of the ZIP.
func readEntry(entry *zip.File) ([]byte, error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, invalid("%s cannot be opened (%v)", entry.Name, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, invalid("%s cannot be read (%v)", entry.Name, err)
	}
	return data, nil
}

// indexEntries lists the entries of the ZIP by name, checking each is what the
// exporter writes: a name that stays inside the file, once, stored with no
// compression.
func indexEntries(files []*zip.File) (map[string]*zip.File, error) {
	entries := make(map[string]*zip.File, len(files))
	for _, entry := range files {
		if !safeName(entry.Name) {
			return nil, invalid("the file has an entry with an unsafe name: %q", entry.Name)
		}
		if _, twice := entries[entry.Name]; twice {
			return nil, invalid("the file has %s twice", entry.Name)
		}
		if entry.Method != zip.Store {
			return nil, invalid("%s is compressed, and only entries stored as they are can be read", entry.Name)
		}
		entries[entry.Name] = entry
	}
	return entries, nil
}

// safeName says whether an entry name stays inside the file: relative, with no
// backslash, and with no ".." to climb out.
func safeName(name string) bool {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, `\`) {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return false
		}
	}
	return path.Clean(name) == name
}

// parseManifest reads the manifest, refusing one that lacks what makes it a
// manifest: its version first, so a file of a version this tool does not know is
// told from a broken one.
func parseManifest(data []byte) (manifestFile, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return manifestFile{}, invalid("manifest.json is not valid JSON (%v)", err)
	}

	rawVersion, ok := fields["format_version"]
	if !ok {
		return manifestFile{}, invalid("manifest.json has no \"format_version\"")
	}
	var version int
	if err := json.Unmarshal(rawVersion, &version); err != nil {
		return manifestFile{}, invalid("format_version is not a whole number (%s)", string(rawVersion))
	}
	if version != sliceFormatVersion {
		return manifestFile{}, fmt.Errorf("%w: found %d, accepted: %d", domain.ErrSliceFormatVersionUnsupported, version, sliceFormatVersion)
	}

	for _, field := range requiredFields {
		if _, ok := fields[field]; !ok {
			return manifestFile{}, invalid("manifest.json has no %q", field)
		}
	}

	var manifest manifestFile
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifestFile{}, invalid("manifest.json has a field of the wrong type (%v)", err)
	}

	switch {
	case manifest.PlanID == "":
		return manifestFile{}, invalid(`the slice has no plan identification; generate it again with "geodata slice --export"`)
	case !planIDPattern.MatchString(manifest.PlanID):
		return manifestFile{}, invalid("plan_id is not 64 hexadecimal characters")
	}
	return manifest, nil
}

// checkSummary says whether the summary of the manifest is what the content of the
// file gives, so nothing that was said of the slice is left unchecked.
func checkSummary(manifest manifestFile, content domain.SliceSummary) error {
	said := manifest.Summary
	differs := func(field string, said, found any) error {
		return invalid("summary.%s is %v but the file has %v", field, said, found)
	}

	switch {
	case said.TileCount != content.TileCount:
		return differs("tile_count", said.TileCount, content.TileCount)
	case said.MissingTileCount != content.MissingTileCount:
		return differs("missing_tile_count", said.MissingTileCount, content.MissingTileCount)
	case said.SampleCount != content.SampleCount:
		return differs("sample_count", said.SampleCount, content.SampleCount)
	case said.NoValueSampleCount != content.NoValueSampleCount:
		return differs("no_value_sample_count", said.NoValueSampleCount, content.NoValueSampleCount)
	case said.SizeBytes != content.SizeBytes:
		return differs("size_bytes", said.SizeBytes, content.SizeBytes)
	case (said.Elevation != nil) != content.HasElevationRange:
		return invalid("summary.elevation_m says there is %s range of elevation, and the file has %s", rangeWord(said.Elevation != nil), rangeWord(content.HasElevationRange))
	case said.Elevation != nil && (math.Abs(said.Elevation.Min-content.MinElevation) > 1e-3 || math.Abs(said.Elevation.Max-content.MaxElevation) > 1e-3):
		return invalid("summary.elevation_m is %g to %g but the file has %g to %g", said.Elevation.Min, said.Elevation.Max, content.MinElevation, content.MaxElevation)
	}
	return nil
}

func rangeWord(has bool) string {
	if has {
		return "a"
	}
	return "no"
}
