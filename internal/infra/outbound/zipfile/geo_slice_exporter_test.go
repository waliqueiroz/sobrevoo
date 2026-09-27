package zipfile_test

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/zipfile"
)

type manifest struct {
	FormatVersion int    `json:"format_version"`
	PlanID        string `json:"plan_id"`
	Area          struct {
		MinLatitude float64 `json:"min_lat"`
		MaxLatitude float64 `json:"max_lat"`
		MinLon      float64 `json:"min_lon"`
		MaxLon      float64 `json:"max_lon"`
		Crosses     bool    `json:"crosses_antimeridian"`
	} `json:"area"`
	Summary struct {
		TileCount      int `json:"tile_count"`
		MissingCount   int `json:"missing_tile_count"`
		SampleCount    int `json:"sample_count"`
		NoValueCount   int `json:"no_value_sample_count"`
		ElevationRange *struct {
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
			X, Y  int
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

func exportSlice(t *testing.T, slice domain.GeoSlice) (path string, content []byte) {
	t.Helper()
	path = filepath.Join(t.TempDir(), "slice.zip")
	require.NoError(t, zipfile.NewGeoSliceExporter().Export(slice, path, false))
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return path, content
}

func openZip(t *testing.T, content []byte) *zip.Reader {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)
	return reader
}

func readEntry(t *testing.T, reader *zip.Reader, name string) []byte {
	t.Helper()
	for _, file := range reader.File {
		if file.Name == name {
			opened, err := file.Open()
			require.NoError(t, err)
			defer opened.Close()
			data, err := io.ReadAll(opened)
			require.NoError(t, err)
			return data
		}
	}
	require.Failf(t, "entry not found", "%s", name)
	return nil
}

func readManifest(t *testing.T, reader *zip.Reader) manifest {
	t.Helper()
	var m manifest
	require.NoError(t, json.Unmarshal(readEntry(t, reader, "manifest.json"), &m))
	return m
}

func entryNames(reader *zip.Reader) []string {
	names := make([]string, len(reader.File))
	for i, file := range reader.File {
		names[i] = file.Name
	}
	return names
}

func twoSourceSlice() domain.GeoSlice {
	mapB := builddomain.NewGeoDataSourceBuilder().WithName("b-map").WithPath("/data/b.mbtiles").Build()
	mapA := builddomain.NewGeoDataSourceBuilder().WithName("a-map").WithPath("/data/a.mbtiles").Build()
	dem := builddomain.NewGeoDataSourceBuilder().WithName("dem").WithPath("/data/dem.tif").
		WithType(domain.DataTypeElevation).WithFormat(domain.DataFormatGeoTIFF).Build()

	tiles := func(level int, xs ...int) []domain.Tile {
		var list []domain.Tile
		for _, x := range xs {
			list = append(list, domain.Tile{ID: domain.TileID{Level: level, X: x, Y: 7}, Data: []byte{byte(level), byte(x)}})
		}
		return list
	}

	return builddomain.NewGeoSliceBuilder().
		WithArea(domain.BoundingBox{MinLatitude: -23.61, MaxLatitude: -23.48, MinLongitude: -46.72, MaxLongitude: -46.53}).
		WithTileSets(
			builddomain.NewTileSetBuilder().WithSource(mapB).WithFormat("jpg").
				WithDetail(domain.DetailLevel{Ideal: 15, Chosen: 12, Min: 0, Max: 12, Reason: "above the source's maximum level"}).
				WithTiles(tiles(12, 30, 5)...).Build(),
			builddomain.NewTileSetBuilder().WithSource(mapA).
				WithDetail(domain.DetailLevel{Ideal: 15, Chosen: 15, Min: 0, Max: 18, Reason: "within the source's range"}).
				WithTiles(tiles(15, 9, 8)...).
				WithMissing(domain.TileID{Level: 15, X: 10, Y: 7}, domain.TileID{Level: 15, X: 11, Y: 7}).Build(),
		).
		WithElevation(
			builddomain.NewElevationGridBuilder().WithSource(dem).WithNoValueAt(1, 1).Build(),
			builddomain.NewElevationGridBuilder().WithSource(dem).
				WithWindow(domain.GridWindow{Rows: 1, Cols: 2}).WithValues(7, 8).Build(),
		).Build()
}

func Test_GeoSliceExporter_Export(t *testing.T) {
	t.Run("should write the manifest first, then the elevation grids, then the tiles in order", func(t *testing.T) {
		// given
		slice := twoSourceSlice()

		// when
		_, content := exportSlice(t, slice)

		// then: sources sorted by name are a-map (0), b-map (1), dem (2); tiles by (source, level, x, y)
		assert.Equal(t, []string{
			"manifest.json",
			"elevation/000.f32", "elevation/001.f32",
			"tiles/000/15/8/7.png", "tiles/000/15/9/7.png",
			"tiles/001/12/5/7.jpg", "tiles/001/12/30/7.jpg",
		}, entryNames(openZip(t, content)))
	})

	t.Run("should store every entry uncompressed with a fixed date and no comment", func(t *testing.T) {
		// given
		_, content := exportSlice(t, twoSourceSlice())

		// when
		reader := openZip(t, content)

		// then
		fixed := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
		assert.Empty(t, reader.Comment)
		for _, file := range reader.File {
			assert.Equal(t, zip.Store, file.Method, file.Name)
			assert.True(t, file.Modified.UTC().Equal(fixed), "%s: %s", file.Name, file.Modified)
		}
	})

	t.Run("should describe the slice in the manifest", func(t *testing.T) {
		// given
		slice := twoSourceSlice()
		_, content := exportSlice(t, slice)

		// when
		m := readManifest(t, openZip(t, content))

		// then
		assert.Equal(t, 1, m.FormatVersion)
		assert.Equal(t, -23.61, m.Area.MinLatitude)
		assert.Equal(t, -23.48, m.Area.MaxLatitude)
		assert.Equal(t, -46.72, m.Area.MinLon)
		assert.Equal(t, -46.53, m.Area.MaxLon)
		assert.False(t, m.Area.Crosses)

		assert.Equal(t, 4, m.Summary.TileCount)
		assert.Equal(t, 2, m.Summary.MissingCount)
		assert.Equal(t, 11, m.Summary.SampleCount)
		assert.Equal(t, 1, m.Summary.NoValueCount)
		require.NotNil(t, m.Summary.ElevationRange)
		assert.Equal(t, 7.0, m.Summary.ElevationRange.Min)
		assert.Equal(t, 108.0, m.Summary.ElevationRange.Max)
		assert.Equal(t, slice.Summary.SizeBytes, m.Summary.SizeBytes)

		require.Len(t, m.Sources, 3)
		assert.Equal(t, "a-map", m.Sources[0].Name)
		assert.Equal(t, "base map", m.Sources[0].Type)
		assert.Equal(t, "MBTiles", m.Sources[0].Format)
		assert.Equal(t, "/data/a.mbtiles", m.Sources[0].Path)
		assert.Equal(t, "dem", m.Sources[2].Name)
		assert.Equal(t, "elevation", m.Sources[2].Type)
		assert.Equal(t, "GeoTIFF", m.Sources[2].Format)
	})

	t.Run("should write the identification of the plan right after the format version", func(t *testing.T) {
		// given
		planID := strings.Repeat("ab", 32)
		slice := twoSourceSlice()
		slice.PlanID = planID
		_, content := exportSlice(t, slice)

		// when
		raw := string(readEntry(t, openZip(t, content), "manifest.json"))
		m := readManifest(t, openZip(t, content))

		// then
		assert.Equal(t, planID, m.PlanID)
		assert.Equal(t, 1, m.FormatVersion, "adding the field does not change the version")
		assert.Contains(t, raw, "{\n  \"format_version\": 1,\n  \"plan_id\": \""+planID+"\",\n  \"area\": {")
	})

	t.Run("should list the base maps with their level, tiles and missing tiles", func(t *testing.T) {
		// given
		_, content := exportSlice(t, twoSourceSlice())

		// when
		m := readManifest(t, openZip(t, content))

		// then
		require.Len(t, m.BaseMap, 2)
		first, second := m.BaseMap[0], m.BaseMap[1]
		assert.Equal(t, 0, first.Source)
		assert.Equal(t, "png", first.TileFormat)
		assert.Equal(t, 15, first.Level.Ideal)
		assert.Equal(t, 15, first.Level.Chosen)
		assert.Equal(t, 18, first.Level.Max)
		assert.Equal(t, "within the source's range", first.Level.Reason)
		require.Len(t, first.Tiles, 2)
		assert.Equal(t, 8, first.Tiles[0].X)
		assert.Equal(t, "tiles/000/15/8/7.png", first.Tiles[0].Path)
		assert.Equal(t, 2, first.Tiles[0].Bytes)
		require.Len(t, first.Missing, 2)
		assert.Equal(t, 10, first.Missing[0].X)

		assert.Equal(t, 1, second.Source)
		assert.Equal(t, "jpg", second.TileFormat)
		assert.Equal(t, 12, second.Level.Chosen)
		assert.Equal(t, "above the source's maximum level", second.Level.Reason)
		assert.Equal(t, 5, second.Tiles[0].X, "tiles are sorted by x")
	})

	t.Run("should always write the list of missing tiles, empty when none is missing", func(t *testing.T) {
		// given
		_, content := exportSlice(t, builddomain.NewGeoSliceBuilder().Build())

		// when
		raw := readEntry(t, openZip(t, content), "manifest.json")

		// then
		assert.Contains(t, string(raw), `"missing": []`)
	})

	t.Run("should describe each elevation grid", func(t *testing.T) {
		// given
		_, content := exportSlice(t, twoSourceSlice())

		// when
		m := readManifest(t, openZip(t, content))

		// then
		require.Len(t, m.Elevation, 2)
		assert.Equal(t, 2, m.Elevation[0].Source)
		assert.Equal(t, "elevation/000.f32", m.Elevation[0].File)
		assert.Equal(t, 3, m.Elevation[0].Rows)
		assert.Equal(t, 3, m.Elevation[0].Cols)
		assert.Equal(t, -23.0, m.Elevation[0].NorthLat)
		assert.Equal(t, -47.0, m.Elevation[0].WestLon)
		assert.Equal(t, 0.001, m.Elevation[0].CellLat)
		assert.Equal(t, 0.001, m.Elevation[0].CellLon)
		assert.Equal(t, 1, m.Elevation[0].NoValueCount)
		assert.Equal(t, 1, m.Elevation[1].Rows)
		assert.Equal(t, 2, m.Elevation[1].Cols)
	})

	t.Run("should write null for the elevation range when no sample has a value", func(t *testing.T) {
		// given
		builder := builddomain.NewElevationGridBuilder()
		for row := 0; row < 3; row++ {
			for col := 0; col < 3; col++ {
				builder.WithNoValueAt(row, col)
			}
		}
		_, content := exportSlice(t, builddomain.NewGeoSliceBuilder().WithElevation(builder.Build()).Build())

		// when
		m := readManifest(t, openZip(t, content))

		// then
		assert.Nil(t, m.Summary.ElevationRange)
		assert.Contains(t, string(readEntry(t, openZip(t, content), "manifest.json")), `"elevation_m": null`)
	})

	t.Run("should write the original bytes of every tile", func(t *testing.T) {
		// given
		_, content := exportSlice(t, twoSourceSlice())

		// when
		reader := openZip(t, content)

		// then
		assert.Equal(t, []byte{15, 8}, readEntry(t, reader, "tiles/000/15/8/7.png"))
		assert.Equal(t, []byte{12, 30}, readEntry(t, reader, "tiles/001/12/30/7.jpg"))
	})

	t.Run("should write the samples as little-endian float32, north to south, with a NaN for no value", func(t *testing.T) {
		// given
		_, content := exportSlice(t, twoSourceSlice())
		reader := openZip(t, content)

		// when
		first := readEntry(t, reader, "elevation/000.f32")
		second := readEntry(t, reader, "elevation/001.f32")

		// then
		require.Len(t, first, 9*4)
		sample := func(data []byte, i int) uint32 { return binary.LittleEndian.Uint32(data[4*i:]) }
		assert.Equal(t, math.Float32bits(100), sample(first, 0))
		assert.Equal(t, math.Float32bits(103), sample(first, 3))
		assert.Equal(t, uint32(0x7FC00000), sample(first, 4), "the sample without value is the quiet NaN 0x7FC00000")
		assert.Equal(t, math.Float32bits(108), sample(first, 8))
		require.Len(t, second, 2*4)
		assert.Equal(t, math.Float32bits(8), sample(second, 1))
	})

	t.Run("should never write a NaN other than the one for no value", func(t *testing.T) {
		// given
		_, content := exportSlice(t, twoSourceSlice())
		data := readEntry(t, openZip(t, content), "elevation/000.f32")

		// when
		var nans int
		for i := 0; i < len(data); i += 4 {
			if math.IsNaN(float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i:])))) {
				nans++
				assert.Equal(t, uint32(0x7FC00000), binary.LittleEndian.Uint32(data[i:]))
			}
		}

		// then
		assert.Equal(t, 1, nans)
	})

	t.Run("should write a slice that reads back with the same counts as its summary", func(t *testing.T) {
		// given
		slice := twoSourceSlice()
		_, content := exportSlice(t, slice)
		reader := openZip(t, content)

		// when
		m := readManifest(t, reader)

		// then
		var tiles, missing, samples int
		var size int64
		for _, baseMap := range m.BaseMap {
			tiles += len(baseMap.Tiles)
			missing += len(baseMap.Missing)
			for _, tile := range baseMap.Tiles {
				data := readEntry(t, reader, tile.Path)
				assert.Len(t, data, tile.Bytes)
				size += int64(len(data))
			}
		}
		for _, grid := range m.Elevation {
			data := readEntry(t, reader, grid.File)
			assert.Len(t, data, grid.Rows*grid.Cols*4)
			samples += grid.Rows * grid.Cols
			size += int64(len(data))
		}
		assert.Equal(t, slice.Summary.TileCount, tiles)
		assert.Equal(t, slice.Summary.MissingTileCount, missing)
		assert.Equal(t, slice.Summary.SampleCount, samples)
		assert.Equal(t, slice.Summary.SizeBytes, size)
	})

	t.Run("should say when the area crosses the antimeridian", func(t *testing.T) {
		// given
		area := domain.BoundingBox{MinLatitude: -1, MaxLatitude: 1, MinLongitude: 170, MaxLongitude: -170, CrossesAntimeridian: true}
		_, content := exportSlice(t, builddomain.NewGeoSliceBuilder().WithArea(area).Build())

		// when
		m := readManifest(t, openZip(t, content))

		// then
		assert.True(t, m.Area.Crosses)
		assert.Equal(t, 170.0, m.Area.MinLon)
		assert.Equal(t, -170.0, m.Area.MaxLon)
	})

	t.Run("should produce identical bytes for a hundred exports of the same slice", func(t *testing.T) {
		// given
		slice := twoSourceSlice()
		_, expected := exportSlice(t, slice)

		for i := 0; i < 100; i++ {
			// when
			_, content := exportSlice(t, slice)

			// then
			assert.Equal(t, expected, content)
		}
	})

	t.Run("should refuse an existing destination without overwrite, leaving it intact", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "slice.zip")
		require.NoError(t, os.WriteFile(path, []byte("precious"), 0o644))

		// when
		err := zipfile.NewGeoSliceExporter().Export(twoSourceSlice(), path, false)

		// then
		require.ErrorIs(t, err, domain.ErrSliceDestinationExists)
		assert.ErrorContains(t, err, "use --overwrite to replace it")
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "precious", string(content))
	})

	t.Run("should replace an existing destination entirely with overwrite", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "slice.zip")
		require.NoError(t, os.WriteFile(path, bytes.Repeat([]byte("old"), 100000), 0o644))

		// when
		err := zipfile.NewGeoSliceExporter().Export(twoSourceSlice(), path, true)

		// then
		require.NoError(t, err)
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		_, expected := exportSlice(t, twoSourceSlice())
		assert.Equal(t, expected, content)
	})

	t.Run("should report a destination in a missing directory as invalid and leave nothing behind", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "missing", "slice.zip")

		// when
		err := zipfile.NewGeoSliceExporter().Export(twoSourceSlice(), path, false)

		// then
		require.ErrorIs(t, err, domain.ErrSliceDestinationInvalid)
		entries, _ := os.ReadDir(dir)
		assert.Empty(t, entries)
	})

	t.Run("should leave only the slice file in the directory after a successful export", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		err := zipfile.NewGeoSliceExporter().Export(twoSourceSlice(), filepath.Join(dir, "slice.zip"), false)

		// then
		require.NoError(t, err)
		entries, _ := os.ReadDir(dir)
		require.Len(t, entries, 1)
		assert.Equal(t, "slice.zip", entries[0].Name())
	})
}
