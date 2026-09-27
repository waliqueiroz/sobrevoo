package zipfile_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/zipfile"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

// writeSliceFile writes content in a slice file of a temporary directory and
// returns its path.
func writeSliceFile(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "slice.zip")
	require.NoError(t, os.WriteFile(path, content, 0o600))
	return path
}

func Test_GeoSliceReader_Read(t *testing.T) {
	spec := helper.DefaultSliceFileSpec()
	content := helper.ValidSliceFile(spec)
	path := writeSliceFile(t, content)

	t.Run("should read the area, keeping whether it crosses the antimeridian", func(t *testing.T) {
		// given
		crossing := helper.DefaultSliceFileSpec()
		crossing.MinLon, crossing.MaxLon, crossing.CrossesAntimeridian = 179.99, -179.99, true

		// when
		plain, err1 := zipfile.NewGeoSliceReader().Read(path)
		wrapped, err2 := zipfile.NewGeoSliceReader().Read(writeSliceFile(t, helper.ValidSliceFile(crossing)))

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, domain.BoundingBox{MinLatitude: -23.004, MaxLatitude: -23, MinLongitude: -47, MaxLongitude: -46.996}, plain.Area)
		assert.Equal(t, domain.BoundingBox{MinLatitude: -23.004, MaxLatitude: -23, MinLongitude: 179.99, MaxLongitude: -179.99, CrossesAntimeridian: true}, wrapped.Area)
	})

	t.Run("should read the tiles of the base map, with their bytes exactly, and the ones that are missing", func(t *testing.T) {
		// given / when
		slice, err := zipfile.NewGeoSliceReader().Read(path)

		// then
		require.NoError(t, err)
		require.Len(t, slice.TileSets, 1)
		tileSet := slice.TileSets[0]
		assert.Equal(t, "map", tileSet.Source.Name)
		assert.Equal(t, domain.DataTypeBaseMap, tileSet.Source.Type)
		assert.Equal(t, domain.DataFormatMBTiles, tileSet.Source.Format)
		assert.Equal(t, "/data/map.mbtiles", tileSet.Source.Path)
		assert.Equal(t, "png", tileSet.Format)
		assert.Equal(t, domain.DetailLevel{Ideal: 17, Chosen: 16, Min: 0, Max: 16, Reason: "above the source's maximum level"}, tileSet.Detail)

		require.Len(t, tileSet.Tiles, 2)
		assert.Equal(t, domain.TileID{Level: 16, X: 24122, Y: 36869}, tileSet.Tiles[0].ID)
		assert.Equal(t, spec.BaseMaps[0].Tiles[0].Data, tileSet.Tiles[0].Data)
		assert.Equal(t, spec.BaseMaps[0].Tiles[1].Data, tileSet.Tiles[1].Data)
		assert.Equal(t, []domain.TileID{{Level: 16, X: 24124, Y: 36869}}, tileSet.Missing)
	})

	t.Run("should read the elevation grids, with a sample that has no value as such", func(t *testing.T) {
		// given / when
		slice, err := zipfile.NewGeoSliceReader().Read(path)

		// then
		require.NoError(t, err)
		require.Len(t, slice.Elevation, 1)
		grid := slice.Elevation[0]
		assert.Equal(t, "dem", grid.Source.Name)
		assert.Equal(t, domain.DataTypeElevation, grid.Source.Type)
		assert.Equal(t, 4, grid.Rows())
		assert.Equal(t, 4, grid.Cols())
		assert.Equal(t, -23.0, grid.NorthLatitude)
		assert.Equal(t, -47.0, grid.WestLongitude)
		assert.Equal(t, 0.001, grid.CellLatitude)
		assert.Equal(t, 0.001, grid.CellLongitude)

		first, hasFirst := grid.At(0, 0)
		last, hasLast := grid.At(3, 3)
		_, hasHole := grid.At(1, 1)
		assert.True(t, hasFirst)
		assert.Equal(t, 100.0, first)
		assert.True(t, hasLast)
		assert.Equal(t, 115.0, last)
		assert.False(t, hasHole)
		assert.Equal(t, 1, grid.NoValueCount())
	})

	t.Run("should have a summary that is the one of the manifest", func(t *testing.T) {
		// given / when
		slice, err := zipfile.NewGeoSliceReader().Read(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, 2, slice.Summary.TileCount)
		assert.Equal(t, 1, slice.Summary.MissingTileCount)
		assert.Equal(t, 16, slice.Summary.SampleCount)
		assert.Equal(t, 1, slice.Summary.NoValueSampleCount)
		assert.True(t, slice.Summary.HasElevationRange)
		assert.Equal(t, 100.0, slice.Summary.MinElevation)
		assert.Equal(t, 115.0, slice.Summary.MaxElevation)
		var tileBytes int64
		for _, tile := range spec.BaseMaps[0].Tiles {
			tileBytes += int64(len(tile.Data))
		}
		assert.Equal(t, tileBytes+16*4, slice.Summary.SizeBytes)
		require.Len(t, slice.Summary.Sources, 2)
		assert.Equal(t, "dem", slice.Summary.Sources[0].Source.Name)
		assert.Equal(t, "map", slice.Summary.Sources[1].Source.Name)
	})

	t.Run("should say which plan the slice was made from", func(t *testing.T) {
		// given / when
		slice, err := zipfile.NewGeoSliceReader().Read(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, spec.PlanID, slice.PlanID)
	})

	t.Run("should identify the file by the SHA-256 of all of it", func(t *testing.T) {
		// given
		sum := sha256.Sum256(content)

		// when
		slice, err := zipfile.NewGeoSliceReader().Read(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, hex.EncodeToString(sum[:]), slice.ContentID)
	})

	t.Run("should have another identification when one byte of a tile is another", func(t *testing.T) {
		// given
		other := helper.DefaultSliceFileSpec()
		changed := append([]byte(nil), other.BaseMaps[0].Tiles[1].Data...)
		changed[len(changed)-3] ^= 0x01
		other.BaseMaps[0].Tiles[1].Data = changed

		// when
		first, err1 := zipfile.NewGeoSliceReader().Read(path)
		second, err2 := zipfile.NewGeoSliceReader().Read(writeSliceFile(t, helper.ValidSliceFile(other)))

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, first.ContentID, second.ContentID)
	})

	t.Run("should read a slice of two base maps and two grids", func(t *testing.T) {
		// given
		two := helper.DefaultSliceFileSpec()
		two.Sources = append(two.Sources,
			helper.SliceSourceSpec{Name: "another-map", Type: "base map", Format: "MBTiles", Path: "/data/another.mbtiles"},
			helper.SliceSourceSpec{Name: "another-dem", Type: "elevation", Format: "GeoTIFF", Path: "/data/another.tif"})
		two.BaseMaps = append(two.BaseMaps, helper.SliceBaseMapSpec{
			Source: 2, TileFormat: "jpg", Ideal: 12, Chosen: 12, Min: 0, Max: 14, Reason: "within the source's range",
			Tiles: []helper.MBTile{{Z: 12, X: 1500, Y: 2300, Data: helper.JPEGTile(helper.Color{R: 1, G: 2, B: 3})}},
		})
		two.Grids = append(two.Grids, helper.SliceGridSpec{
			Source: 3, Rows: 2, Cols: 2, NorthLat: -23, WestLon: -46.996, CellLat: 0.002, CellLon: 0.002, Values: []float32{1, 2, 3, 4},
		})

		// when
		slice, err := zipfile.NewGeoSliceReader().Read(writeSliceFile(t, helper.ValidSliceFile(two)))

		// then
		require.NoError(t, err)
		require.Len(t, slice.TileSets, 2)
		assert.Equal(t, "another-map", slice.TileSets[0].Source.Name, "the tile sets are in the order of NewGeoSlice: by name")
		assert.Equal(t, "jpg", slice.TileSets[0].Format)
		assert.Equal(t, "map", slice.TileSets[1].Source.Name)
		require.Len(t, slice.Elevation, 2)
		assert.Equal(t, "dem", slice.Elevation[0].Source.Name)
		assert.Equal(t, "another-dem", slice.Elevation[1].Source.Name)
		assert.Equal(t, 20, slice.Summary.SampleCount)
	})

	t.Run("should report a file that is not there as an I/O error, with no sentinel", func(t *testing.T) {
		// given
		missing := filepath.Join(t.TempDir(), "nope.zip")

		// when
		_, err := zipfile.NewGeoSliceReader().Read(missing)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, fs.ErrNotExist)
		assert.NotErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.NotErrorIs(t, err, domain.ErrSliceFormatVersionUnsupported)
	})

	t.Run("should refuse a file that is not a ZIP, and one with no manifest", func(t *testing.T) {
		// given / when
		_, notZip := zipfile.NewGeoSliceReader().Read(writeSliceFile(t, helper.NotZipContent()))
		_, noManifest := zipfile.NewGeoSliceReader().Read(writeSliceFile(t, helper.SliceFileWithoutManifest()))

		// then
		assert.ErrorIs(t, notZip, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, notZip, "not a ZIP file")
		assert.ErrorIs(t, noManifest, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, noManifest, "manifest.json")
	})
}

func Test_GeoSliceReader_Read_InvalidFiles(t *testing.T) {
	read := func(t *testing.T, content []byte) error {
		t.Helper()
		_, err := zipfile.NewGeoSliceReader().Read(writeSliceFile(t, content))
		return err
	}

	t.Run("should refuse a ZIP that is cut short", func(t *testing.T) {
		// given / when
		err := read(t, helper.TruncatedSliceFile())

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
	})

	t.Run("should refuse a manifest that is not JSON", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithManifest("this is { not json"))

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "manifest.json is not valid JSON")
	})

	t.Run("should refuse a manifest that lacks a field, naming it", func(t *testing.T) {
		for _, field := range []string{"area", "summary", "sources", "base_map", "elevation"} {
			// given / when
			err := read(t, helper.SliceFileWithoutField(field))

			// then
			assert.ErrorIs(t, err, domain.ErrSliceFileInvalid, field)
			assert.ErrorContains(t, err, `"`+field+`"`, field)
		}
	})

	t.Run("should say a slice with no plan identification has to be generated again", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithoutPlanID())

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, `the slice has no plan identification; generate it again with "geodata slice --export"`)
	})

	t.Run("should refuse a plan identification that is not 64 hexadecimal characters", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithBadPlanID())

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "plan_id")
	})

	t.Run("should refuse a version of the format it does not know, saying the one found and the ones accepted", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithVersion(2))

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFormatVersionUnsupported)
		assert.NotErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "found 2, accepted: 1")
	})

	t.Run("should refuse a version that is missing or not a number as an invalid slice", func(t *testing.T) {
		// given / when
		missing := read(t, helper.SliceFileWithoutFormatVersion())
		text := read(t, helper.SliceFileWithTextFormatVersion())

		// then
		assert.ErrorIs(t, missing, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, missing, "format_version")
		assert.ErrorIs(t, text, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, text, "format_version")
	})

	t.Run("should refuse a tile count that is not the one of the content", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithWrongCounts("tile_count"))

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "summary.tile_count is 3 but the file has 2")
	})

	t.Run("should refuse a count of missing tiles that is not the one of the content", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithWrongCounts("missing_tile_count"))

		// then
		assert.ErrorContains(t, err, "summary.missing_tile_count is 2 but the file has 1")
	})

	t.Run("should refuse counts of samples that are not the ones of the content", func(t *testing.T) {
		// given / when
		samples := read(t, helper.SliceFileWithWrongCounts("sample_count"))
		noValue := read(t, helper.SliceFileWithWrongCounts("no_value_sample_count"))

		// then
		assert.ErrorContains(t, samples, "summary.sample_count is 17 but the file has 16")
		assert.ErrorContains(t, noValue, "summary.no_value_sample_count is 2 but the file has 1")
	})

	t.Run("should refuse an elevation range and a size that are not the ones of the content", func(t *testing.T) {
		// given / when
		elevation := read(t, helper.SliceFileWithWrongCounts("elevation_m"))
		size := read(t, helper.SliceFileWithWrongCounts("size_bytes"))

		// then
		assert.ErrorIs(t, elevation, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, elevation, "summary.elevation_m")
		assert.ErrorIs(t, size, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, size, "summary.size_bytes")
	})

	t.Run("should refuse a tile or a grid the manifest lists and the file lacks", func(t *testing.T) {
		// given / when
		tile := read(t, helper.SliceFileWithoutEntry("tiles/000/16/24122/36869.png"))
		grid := read(t, helper.SliceFileWithoutEntry("elevation/000.f32"))

		// then
		assert.ErrorIs(t, tile, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, tile, "tiles/000/16/24122/36869.png")
		assert.ErrorIs(t, grid, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, grid, "elevation/000.f32")
	})

	t.Run("should refuse an entry the manifest does not list", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithExtraEntry())

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "notes.txt")
	})

	t.Run("should refuse a tile whose size is not the one of the manifest", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithWrongTileSize())

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "bytes")
	})

	t.Run("should refuse a grid whose entry does not hold the samples of the manifest", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithWrongGridSize())

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "elevation/000.f32")
	})

	t.Run("should refuse an entry that is compressed: only the store method is accepted", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithCompressedEntry())

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "compress")
	})

	t.Run("should refuse a chosen level outside what the source offers", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithChosenOutOfRange())

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "level.chosen is 17 but the source offers 0 to 16")
	})

	t.Run("should refuse a base map that points at a source that is not listed", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithBadSourceIndex())

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "source 9")
	})

	t.Run("should refuse an entry whose checksum is wrong", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithCorruptEntry())

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "elevation/000.f32")
	})

	t.Run("should refuse two entries with the same name", func(t *testing.T) {
		// given / when
		err := read(t, helper.SliceFileWithDuplicateEntry())

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "twice")
	})

	t.Run("should refuse an entry whose name leaves the directory, or is absolute", func(t *testing.T) {
		for _, name := range []string{"../evil.txt", "/etc/evil.txt", "tiles/../../evil.txt", `tiles\..\evil.txt`} {
			// given / when
			err := read(t, helper.SliceFileWithUnsafeEntry(name))

			// then
			assert.ErrorIs(t, err, domain.ErrSliceFileInvalid, name)
			assert.ErrorContains(t, err, "unsafe", name)
		}
	})

	t.Run("should refuse a file bigger than the biggest slice", func(t *testing.T) {
		// given
		restore := zipfile.SetMaxSliceBytes(1000)
		defer restore()

		// when
		err := read(t, helper.ValidSliceFile(helper.DefaultSliceFileSpec()))

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "bigger than a slice can be")
	})
}

func Test_GeoSliceReader_RoundTrip(t *testing.T) {
	t.Run("should read back what the exporter wrote, without loss", func(t *testing.T) {
		// given
		slice := twoSourceSlice()
		slice.PlanID = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
		path := filepath.Join(t.TempDir(), "slice.zip")
		require.NoError(t, zipfile.NewGeoSliceExporter().Export(slice, path, false))

		// when
		read, err := zipfile.NewGeoSliceReader().Read(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, slice.Area, read.Area)
		assert.Equal(t, slice.PlanID, read.PlanID)
		assert.Equal(t, slice.Summary.TileCount, read.Summary.TileCount)
		assert.Equal(t, slice.Summary.MissingTileCount, read.Summary.MissingTileCount)
		assert.Equal(t, slice.Summary.SampleCount, read.Summary.SampleCount)
		assert.Equal(t, slice.Summary.NoValueSampleCount, read.Summary.NoValueSampleCount)
		assert.Equal(t, slice.Summary.SizeBytes, read.Summary.SizeBytes)
		require.Len(t, read.TileSets, len(slice.TileSets))
		for i, want := range slice.TileSets {
			got := read.TileSets[i]
			assert.Equal(t, want.Source.Name, got.Source.Name)
			assert.Equal(t, want.Format, got.Format)
			assert.Equal(t, want.Detail.Chosen, got.Detail.Chosen)
			assert.Equal(t, want.Tiles, got.Tiles)
			assert.Equal(t, want.Missing, got.Missing)
		}
		require.Len(t, read.Elevation, len(slice.Elevation))
		for i, want := range slice.Elevation {
			got := read.Elevation[i]
			require.Equal(t, want.Rows(), got.Rows())
			require.Equal(t, want.Cols(), got.Cols())
			for r := 0; r < want.Rows(); r++ {
				for c := 0; c < want.Cols(); c++ {
					wantValue, wantHas := want.At(r, c)
					gotValue, gotHas := got.At(r, c)
					assert.Equal(t, wantHas, gotHas)
					assert.Equal(t, wantValue, gotValue)
				}
			}
		}
	})
}
